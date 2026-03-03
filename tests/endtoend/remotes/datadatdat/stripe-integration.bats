#!/usr/bin/env bats

# Stripe Integration Validation Tests
# Tests real Stripe API calls (checkout, portal) and webhook processing
# with self-signed HMAC payloads.
#
# Requires:
#   STRIPE_SECRET_KEY   — Stripe test-mode secret key (sk_test_...)
#   STRIPE_WEBHOOK_SECRET — Stripe webhook signing secret (whsec_...)
# If not set, tests are skipped.

# Load shared test helpers
load '../../test_helper'

# Test API key for stripe integration user (raw hex key)
STRIPE_USER_KEY="ee11111122223333444455556666777788889999aaaabbbbccccddddeeeeffff"

GATEWAY="http://127.0.0.1:8080"
AUTH_SERVER="http://127.0.0.1:8085"

# ========================================
# Setup / Teardown
# ========================================

setup_file() {
  # Try env vars first, then fall back to reading from .env.auth
  if [[ -z "$STRIPE_SECRET_KEY" ]]; then
    ENV_AUTH="/c/dev/datadatdat/datadatdat-remote-server/.env.auth"
    if [[ -f "$ENV_AUTH" ]]; then
      STRIPE_SECRET_KEY=$(grep '^STRIPE_SECRET_KEY=' "$ENV_AUTH" | cut -d= -f2)
      export STRIPE_SECRET_KEY
    fi
  fi
  if [[ -z "$STRIPE_WEBHOOK_SECRET" ]]; then
    ENV_AUTH="/c/dev/datadatdat/datadatdat-remote-server/.env.auth"
    if [[ -f "$ENV_AUTH" ]]; then
      STRIPE_WEBHOOK_SECRET=$(grep '^STRIPE_WEBHOOK_SECRET=' "$ENV_AUTH" | cut -d= -f2)
      export STRIPE_WEBHOOK_SECRET
    fi
  fi

  if [[ -z "$STRIPE_SECRET_KEY" || -z "$STRIPE_WEBHOOK_SECRET" ]]; then
    echo "STRIPE_SECRET_KEY and STRIPE_WEBHOOK_SECRET required — skipping" >&3
    return 1
  fi

  # Persist secrets for use in individual tests
  echo "$STRIPE_SECRET_KEY" > "$BATS_TMPDIR/stripe_secret_key.txt"
  echo "$STRIPE_WEBHOOK_SECRET" > "$BATS_TMPDIR/stripe_webhook_secret.txt"

  run curl -s "$GATEWAY/health"
  [[ "$output" == *"healthy"* ]] || { echo "Gateway not running"; return 1; }

  run curl -s "$AUTH_SERVER/health"
  [[ "$output" == *"healthy"* ]] || { echo "Auth server not running"; return 1; }

  run docker exec datadatdat-postgres pg_isready -U datadatdat
  [[ "$output" == *"accepting connections"* ]] || { echo "Postgres not ready"; return 1; }
}

teardown_file() {
  # Best-effort cleanup — DB
  docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "DELETE FROM billing_subscriptions WHERE user_id IN (SELECT id FROM users WHERE github_login = 'stripeuser');
     DELETE FROM storage_snapshots WHERE user_id IN (SELECT id FROM users WHERE github_login = 'stripeuser');
     DELETE FROM api_keys WHERE user_id IN (SELECT id FROM users WHERE github_login = 'stripeuser');
     DELETE FROM whitelisted_users WHERE user_id IN (SELECT id FROM users WHERE github_login = 'stripeuser');
     DELETE FROM namespaces WHERE name = 'stripeuser';
     DELETE FROM users WHERE github_login = 'stripeuser';" 2>/dev/null || true

  # Best-effort cleanup — Stripe customer
  if [[ -f "$BATS_TMPDIR/stripe_customer_id.txt" ]]; then
    CUSTOMER_ID=$(cat "$BATS_TMPDIR/stripe_customer_id.txt")
    SK=$(cat "$BATS_TMPDIR/stripe_secret_key.txt" 2>/dev/null || true)
    if [[ -n "$CUSTOMER_ID" && -n "$SK" ]]; then
      curl -s -u "${SK}:" -X DELETE \
        "https://api.stripe.com/v1/customers/${CUSTOMER_ID}" >/dev/null 2>&1 || true
    fi
  fi
}

# ========================================
# Health Checks
# ========================================

@test "stripe: verify auth server is running" {
  run curl -s "$AUTH_SERVER/health"
  assert_success
  assert_output --partial "healthy"
}

@test "stripe: verify billing routes are registered (not 404)" {
  # An unauthenticated POST to /billing/checkout should return 401, not 404
  run curl -s -o /dev/null -w "%{http_code}" -X POST \
    "$AUTH_SERVER/api/v1/billing/checkout"
  assert_success
  assert_output "401"
}

# ========================================
# Test User Bootstrap
# ========================================

@test "stripe: cleanup existing stripeuser data" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "DELETE FROM billing_subscriptions WHERE user_id IN (SELECT id FROM users WHERE github_login = 'stripeuser');
     DELETE FROM storage_snapshots WHERE user_id IN (SELECT id FROM users WHERE github_login = 'stripeuser');
     DELETE FROM api_keys WHERE user_id IN (SELECT id FROM users WHERE github_login = 'stripeuser');
     DELETE FROM whitelisted_users WHERE user_id IN (SELECT id FROM users WHERE github_login = 'stripeuser');
     DELETE FROM namespaces WHERE name = 'stripeuser';
     DELETE FROM users WHERE github_login = 'stripeuser';"
  assert_success
}

@test "stripe: create stripeuser (whitelisted)" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "INSERT INTO users (github_id, github_login, github_email, github_name, github_avatar_url, is_whitelisted, is_admin, created_at, updated_at)
     VALUES (500001, 'stripeuser', 'stripe@test.com', 'Stripe Test User', '', true, false, NOW(), NOW())
     ON CONFLICT (github_id) DO UPDATE SET is_whitelisted = true, is_admin = false, github_avatar_url = '';
     INSERT INTO whitelisted_users (user_id, approved_by, approved_at, reason)
     VALUES ((SELECT id FROM users WHERE github_login = 'stripeuser'), (SELECT id FROM users WHERE github_login = 'stripeuser'), NOW(), 'E2E Stripe Integration Test')
     ON CONFLICT (user_id) DO NOTHING;"
  assert_success
  assert_output --partial "INSERT"
}

@test "stripe: get stripeuser ID" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT id FROM users WHERE github_login = 'stripeuser' LIMIT 1;"
  assert_success
  STRIPE_USER_ID=$(echo "$output" | tr -d '[:space:]')
  echo "$STRIPE_USER_ID" > "$BATS_TMPDIR/stripe_user_id.txt"
  [[ -n "$STRIPE_USER_ID" ]]
}

@test "stripe: create API key for stripeuser" {
  STRIPE_USER_ID=$(cat "$BATS_TMPDIR/stripe_user_id.txt")
  KEY_HASH=$(echo -n "$STRIPE_USER_KEY" | sha256sum | cut -d' ' -f1)
  KEY_PREFIX="${STRIPE_USER_KEY:0:8}"

  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "INSERT INTO api_keys (user_id, key_hash, key_prefix, name, created_at)
     VALUES ('${STRIPE_USER_ID}', '${KEY_HASH}', '${KEY_PREFIX}', 'Stripe Integration Test Key', NOW());"
  assert_success
  assert_output --partial "INSERT"
}

@test "stripe: verify stripeuser API key authenticates" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $STRIPE_USER_KEY" \
    "$GATEWAY/health"
  assert_success
  assert_output "200"
}

# ========================================
# Real Stripe API — Checkout Session
# ========================================

@test "stripe: POST /billing/checkout creates real Stripe Checkout session" {
  run curl -s -H "X-API-Key: $STRIPE_USER_KEY" \
    -X POST "$AUTH_SERVER/api/v1/billing/checkout"
  assert_success
  # Response must contain a real Stripe Checkout URL
  assert_output --partial '"url"'
  assert_output --partial "checkout.stripe.com"
}

@test "stripe: verify stripe_customer_id saved to database" {
  STRIPE_USER_ID=$(cat "$BATS_TMPDIR/stripe_user_id.txt")

  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT stripe_customer_id FROM users WHERE id = '${STRIPE_USER_ID}';"
  assert_success
  CUSTOMER_ID=$(echo "$output" | tr -d '[:space:]')
  # Must start with cus_ (real Stripe customer ID)
  [[ "$CUSTOMER_ID" == cus_* ]]
  echo "$CUSTOMER_ID" > "$BATS_TMPDIR/stripe_customer_id.txt"
}

# ========================================
# Real Stripe API — Portal Session
# ========================================

@test "stripe: POST /billing/portal creates real Stripe Portal session" {
  run curl -s -H "X-API-Key: $STRIPE_USER_KEY" \
    -X POST "$AUTH_SERVER/api/v1/billing/portal"
  assert_success
  # Response must contain a real Stripe Billing Portal URL
  assert_output --partial '"url"'
  assert_output --partial "billing.stripe.com"
}

# ========================================
# Webhook Processing — Self-signed HMAC
# ========================================

@test "stripe: webhook checkout.session.completed creates subscription" {
  CUSTOMER_ID=$(cat "$BATS_TMPDIR/stripe_customer_id.txt")
  WEBHOOK_SECRET=$(cat "$BATS_TMPDIR/stripe_webhook_secret.txt")
  TIMESTAMP=$(date +%s)
  SUB_ID="sub_test_$(date +%s)"
  echo "$SUB_ID" > "$BATS_TMPDIR/stripe_sub_id.txt"

  PAYLOAD="{\"id\":\"evt_test_${TIMESTAMP}\",\"type\":\"checkout.session.completed\",\"data\":{\"object\":{\"id\":\"cs_test_${TIMESTAMP}\",\"object\":\"checkout.session\",\"customer\":\"${CUSTOMER_ID}\",\"subscription\":\"${SUB_ID}\"}}}"

  SIGNED_PAYLOAD="${TIMESTAMP}.${PAYLOAD}"
  SIGNATURE=$(printf '%s' "$SIGNED_PAYLOAD" | openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" -hex 2>/dev/null | sed 's/^.* //')

  run curl -s -o /dev/null -w "%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -H "Stripe-Signature: t=${TIMESTAMP},v1=${SIGNATURE}" \
    -d "$PAYLOAD" \
    "$AUTH_SERVER/api/v1/billing/webhook"
  assert_success
  assert_output "200"
}

@test "stripe: verify billing subscription created with status active" {
  STRIPE_USER_ID=$(cat "$BATS_TMPDIR/stripe_user_id.txt")

  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT status FROM billing_subscriptions WHERE user_id = '${STRIPE_USER_ID}';"
  assert_success
  STATUS=$(echo "$output" | tr -d '[:space:]')
  [[ "$STATUS" == "active" ]]
}

@test "stripe: GET /billing/subscription returns active" {
  run curl -s -H "X-API-Key: $STRIPE_USER_KEY" \
    "$AUTH_SERVER/api/v1/billing/subscription"
  assert_success
  assert_output --partial '"status":"active"'
}

# ========================================
# Webhook — Subscription Updated
# ========================================

@test "stripe: webhook customer.subscription.updated changes status" {
  CUSTOMER_ID=$(cat "$BATS_TMPDIR/stripe_customer_id.txt")
  SUB_ID=$(cat "$BATS_TMPDIR/stripe_sub_id.txt")
  WEBHOOK_SECRET=$(cat "$BATS_TMPDIR/stripe_webhook_secret.txt")
  TIMESTAMP=$(date +%s)

  PAYLOAD="{\"id\":\"evt_upd_${TIMESTAMP}\",\"type\":\"customer.subscription.updated\",\"data\":{\"object\":{\"id\":\"${SUB_ID}\",\"object\":\"subscription\",\"customer\":\"${CUSTOMER_ID}\",\"status\":\"past_due\",\"cancel_at_period_end\":false,\"items\":{\"data\":[{\"current_period_start\":${TIMESTAMP},\"current_period_end\":$((TIMESTAMP + 2592000))}]}}}}"

  SIGNED_PAYLOAD="${TIMESTAMP}.${PAYLOAD}"
  SIGNATURE=$(printf '%s' "$SIGNED_PAYLOAD" | openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" -hex 2>/dev/null | sed 's/^.* //')

  run curl -s -o /dev/null -w "%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -H "Stripe-Signature: t=${TIMESTAMP},v1=${SIGNATURE}" \
    -d "$PAYLOAD" \
    "$AUTH_SERVER/api/v1/billing/webhook"
  assert_success
  assert_output "200"
}

@test "stripe: verify subscription status updated to past_due" {
  STRIPE_USER_ID=$(cat "$BATS_TMPDIR/stripe_user_id.txt")

  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT status FROM billing_subscriptions WHERE user_id = '${STRIPE_USER_ID}';"
  assert_success
  STATUS=$(echo "$output" | tr -d '[:space:]')
  [[ "$STATUS" == "past_due" ]]
}

# ========================================
# Webhook — Subscription Deleted
# ========================================

@test "stripe: webhook customer.subscription.deleted sets canceled" {
  CUSTOMER_ID=$(cat "$BATS_TMPDIR/stripe_customer_id.txt")
  SUB_ID=$(cat "$BATS_TMPDIR/stripe_sub_id.txt")
  WEBHOOK_SECRET=$(cat "$BATS_TMPDIR/stripe_webhook_secret.txt")
  TIMESTAMP=$(date +%s)

  PAYLOAD="{\"id\":\"evt_del_${TIMESTAMP}\",\"type\":\"customer.subscription.deleted\",\"data\":{\"object\":{\"id\":\"${SUB_ID}\",\"object\":\"subscription\",\"customer\":\"${CUSTOMER_ID}\",\"status\":\"canceled\"}}}"

  SIGNED_PAYLOAD="${TIMESTAMP}.${PAYLOAD}"
  SIGNATURE=$(printf '%s' "$SIGNED_PAYLOAD" | openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" -hex 2>/dev/null | sed 's/^.* //')

  run curl -s -o /dev/null -w "%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -H "Stripe-Signature: t=${TIMESTAMP},v1=${SIGNATURE}" \
    -d "$PAYLOAD" \
    "$AUTH_SERVER/api/v1/billing/webhook"
  assert_success
  assert_output "200"
}

@test "stripe: verify subscription status is canceled" {
  STRIPE_USER_ID=$(cat "$BATS_TMPDIR/stripe_user_id.txt")

  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT status FROM billing_subscriptions WHERE user_id = '${STRIPE_USER_ID}';"
  assert_success
  STATUS=$(echo "$output" | tr -d '[:space:]')
  [[ "$STATUS" == "canceled" ]]
}

@test "stripe: GET /billing/subscription reflects canceled status" {
  run curl -s -H "X-API-Key: $STRIPE_USER_KEY" \
    "$AUTH_SERVER/api/v1/billing/subscription"
  assert_success
  assert_output --partial '"status":"canceled"'
}

# ========================================
# Cleanup
# ========================================

@test "stripe: cleanup billing subscriptions" {
  STRIPE_USER_ID=$(cat "$BATS_TMPDIR/stripe_user_id.txt")

  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "DELETE FROM billing_subscriptions WHERE user_id = '${STRIPE_USER_ID}';"
  assert_success
}

@test "stripe: cleanup Stripe test customer" {
  CUSTOMER_ID=$(cat "$BATS_TMPDIR/stripe_customer_id.txt" 2>/dev/null || true)
  SK=$(cat "$BATS_TMPDIR/stripe_secret_key.txt" 2>/dev/null || true)

  if [[ -n "$CUSTOMER_ID" && -n "$SK" ]]; then
    run curl -s -u "${SK}:" -X DELETE \
      "https://api.stripe.com/v1/customers/${CUSTOMER_ID}"
    assert_success
    assert_output --partial '"deleted"'
  fi
}

@test "stripe: cleanup API keys and test user" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "DELETE FROM api_keys WHERE user_id IN (SELECT id FROM users WHERE github_login = 'stripeuser');
     DELETE FROM whitelisted_users WHERE user_id IN (SELECT id FROM users WHERE github_login = 'stripeuser');
     DELETE FROM namespaces WHERE name = 'stripeuser';
     DELETE FROM users WHERE github_login = 'stripeuser';"
  assert_success
}
