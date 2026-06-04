#!/usr/bin/env bats

# E2E Billing & Usage Workflow Tests for Dit Remote Server
# Tests storage snapshots, usage API, billing subscription endpoints, and webhook validation.

# Load shared test helpers
load '../../test_helper'
load 'env'

# Admin API key (seeded by Liquibase)
ADMIN_KEY="02b31569a9052bc4b3cf1c3819d4fc048d34c96eca21f2b8e2359b5ecdfec93a"

# Test API key for billing user (raw hex key)
BILLING_USER_KEY="cc11111122223333444455556666777788889999aaaabbbbccccddddeeeeffff"

# ========================================
# Setup / Teardown
# ========================================

setup_file() {
  run curl -s "$GATEWAY/health"
  [[ "$output" == *"${HEALTH_EXPECT}"* ]] || { echo "Gateway not running"; return 1; }

  run curl -s "$AUTH_SERVER/health"
  [[ "$output" == *"${HEALTH_EXPECT}"* ]] || { echo "Auth server not running"; return 1; }

  if is_dev; then
    run docker exec dit-postgres pg_isready -U dit
    [[ "$output" == *"accepting connections"* ]] || { echo "Postgres not ready"; return 1; }
  else
    run run_sql_raw "SELECT 1;"
    [[ "$output" == *"1"* ]] || { echo "Postgres not ready"; return 1; }
  fi
}

teardown_file() {
  # Best-effort cleanup — DB only (billing user is test-only)
  run_sql_cmd "DELETE FROM storage_snapshots WHERE user_id IN (SELECT id FROM users WHERE github_login = 'billinguser');
     DELETE FROM billing_subscriptions WHERE user_id IN (SELECT id FROM users WHERE github_login = 'billinguser');
     DELETE FROM api_keys WHERE user_id IN (SELECT id FROM users WHERE github_login = 'billinguser');
     DELETE FROM whitelisted_users WHERE user_id IN (SELECT id FROM users WHERE github_login = 'billinguser');
     DELETE FROM namespaces WHERE name = 'billinguser';
     DELETE FROM users WHERE github_login = 'billinguser';" 2>/dev/null || true
}

# ========================================
# Health Checks
# ========================================

@test "billing: verify auth server is running" {
  run curl -s "$AUTH_SERVER/health"
  assert_success
  assert_output --partial "${HEALTH_EXPECT}"
}

@test "billing: verify api-gateway is running" {
  run curl -s "$GATEWAY/health"
  assert_success
  assert_output --partial "${HEALTH_EXPECT}"
}

@test "billing: verify postgres is ready" {
  is_dev || skip "Local postgres check only for DEV"
  run docker exec dit-postgres pg_isready -U dit
  assert_success
  assert_output --partial "accepting connections"
}

# ========================================
# Test User Bootstrap (DB — no user creation API)
# ========================================

@test "billing: cleanup existing test data from previous runs" {
  run run_sql_cmd "DELETE FROM storage_snapshots WHERE user_id IN (SELECT id FROM users WHERE github_login = 'billinguser');
     DELETE FROM billing_subscriptions WHERE user_id IN (SELECT id FROM users WHERE github_login = 'billinguser');
     DELETE FROM api_keys WHERE user_id IN (SELECT id FROM users WHERE github_login = 'billinguser');
     DELETE FROM whitelisted_users WHERE user_id IN (SELECT id FROM users WHERE github_login = 'billinguser');
     DELETE FROM namespaces WHERE name = 'billinguser';
     DELETE FROM users WHERE github_login = 'billinguser';"
  assert_success
}

@test "billing: create billinguser (whitelisted)" {
  run run_sql_cmd "INSERT INTO users (github_id, github_login, github_email, github_name, github_avatar_url, is_whitelisted, is_admin, created_at, updated_at)
     VALUES (400001, 'billinguser', 'billing@test.com', 'Billing User', '', true, false, NOW(), NOW())
     ON CONFLICT (github_id) DO UPDATE SET is_whitelisted = true, is_admin = false, github_avatar_url = '';
     INSERT INTO whitelisted_users (user_id, approved_by, approved_at, reason)
     VALUES ((SELECT id FROM users WHERE github_login = 'billinguser'), (SELECT id FROM users WHERE github_login = 'billinguser'), NOW(), 'E2E Billing Test User')
     ON CONFLICT (user_id) DO NOTHING;"
  assert_success
  assert_output --partial "INSERT"
}

@test "billing: get billinguser user ID" {
  run run_sql_raw "SELECT id FROM users WHERE github_login = 'billinguser' LIMIT 1;"
  assert_success
  BILLING_USER_ID=$(echo "$output" | tr -d '[:space:]')
  echo "$BILLING_USER_ID" > "$BATS_TMPDIR/billing_user_id.txt"
  [[ -n "$BILLING_USER_ID" ]]
}

@test "billing: create API key for billinguser" {
  BILLING_USER_ID=$(cat "$BATS_TMPDIR/billing_user_id.txt")
  KEY_HASH=$(echo -n "$BILLING_USER_KEY" | sha256sum | cut -d' ' -f1)
  KEY_PREFIX="${BILLING_USER_KEY:0:8}"

  run run_sql_cmd "INSERT INTO api_keys (user_id, key_hash, key_prefix, name, created_at)
     VALUES ('${BILLING_USER_ID}', '${KEY_HASH}', '${KEY_PREFIX}', 'Billing Test Key', NOW());"
  assert_success
  assert_output --partial "INSERT"
}

@test "billing: verify billinguser API key authenticates" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $BILLING_USER_KEY" \
    "$GATEWAY/health"
  assert_success
  assert_output "200"
}

# ========================================
# Storage Snapshots
# ========================================

@test "billing: insert test storage snapshot" {
  BILLING_USER_ID=$(cat "$BATS_TMPDIR/billing_user_id.txt")

  run run_sql_cmd "INSERT INTO storage_snapshots (user_id, recorded_at, total_bytes, repo_details)
     VALUES ('${BILLING_USER_ID}', NOW() - INTERVAL '2 days', 1048576,
       '[{\"repo\": \"billinguser/test-repo\", \"bytes\": 1048576}]'::jsonb);"
  assert_success
  assert_output --partial "INSERT"
}

@test "billing: insert second storage snapshot (higher usage)" {
  BILLING_USER_ID=$(cat "$BATS_TMPDIR/billing_user_id.txt")

  run run_sql_cmd "INSERT INTO storage_snapshots (user_id, recorded_at, total_bytes, repo_details)
     VALUES ('${BILLING_USER_ID}', NOW() - INTERVAL '1 day', 2097152,
       '[{\"repo\": \"billinguser/test-repo\", \"bytes\": 1572864}, {\"repo\": \"billinguser/test-repo-2\", \"bytes\": 524288}]'::jsonb);"
  assert_success
  assert_output --partial "INSERT"
}

@test "billing: insert third storage snapshot (current)" {
  BILLING_USER_ID=$(cat "$BATS_TMPDIR/billing_user_id.txt")

  run run_sql_cmd "INSERT INTO storage_snapshots (user_id, recorded_at, total_bytes, repo_details)
     VALUES ('${BILLING_USER_ID}', NOW(), 1572864,
       '[{\"repo\": \"billinguser/test-repo\", \"bytes\": 1048576}, {\"repo\": \"billinguser/test-repo-2\", \"bytes\": 524288}]'::jsonb);"
  assert_success
  assert_output --partial "INSERT"
}

# ========================================
# Usage API — Authenticated Requests
# ========================================

@test "billing: GET /api/v1/usage returns snapshots" {
  run curl -s -H "X-API-Key: $BILLING_USER_KEY" \
    "$AUTH_SERVER/api/v1/usage"
  assert_success
  assert_output --partial "totalBytes"
  assert_output --partial "recordedAt"
}

@test "billing: GET /api/v1/usage with date range params" {
  FROM=$(date -u -d "3 days ago" +%Y-%m-%d)
  TO=$(date -u +%Y-%m-%d)

  run curl -s -H "X-API-Key: $BILLING_USER_KEY" \
    "$AUTH_SERVER/api/v1/usage?from=${FROM}&to=${TO}"
  assert_success
  assert_output --partial "totalBytes"
}

@test "billing: GET /api/v1/usage/current returns latest snapshot" {
  run curl -s -H "X-API-Key: $BILLING_USER_KEY" \
    "$AUTH_SERVER/api/v1/usage/current"
  assert_success
  assert_output --partial "totalBytes"
  assert_output --partial "repoDetails"
}

@test "billing: GET /api/v1/usage/peak returns max bytes in period" {
  run curl -s -H "X-API-Key: $BILLING_USER_KEY" \
    "$AUTH_SERVER/api/v1/usage/peak"
  assert_success
  assert_output --partial "peakBytes"
  assert_output --partial "peakDate"
}

@test "billing: peak usage returns the correct maximum" {
  run curl -s -H "X-API-Key: $BILLING_USER_KEY" \
    "$AUTH_SERVER/api/v1/usage/peak"
  assert_success
  # The highest snapshot is 2097152 bytes (2 MB)
  assert_output --partial "2097152"
}

@test "billing: GET /api/v1/usage/export returns CSV" {
  # Use -i to include headers in output (avoids temp file issues on Windows)
  run curl -si -H "X-API-Key: $BILLING_USER_KEY" \
    "$AUTH_SERVER/api/v1/usage/export"
  assert_success

  # Verify Content-Type header and CSV content in combined output
  assert_output --partial "text/csv"
  assert_output --partial "Date"
  assert_output --partial "Total Bytes"
}

# ========================================
# Usage API — Unauthenticated Requests (expect 401)
# ========================================

@test "billing: unauthenticated GET /api/v1/usage returns 401" {
  run curl -s -o /dev/null -w "%{http_code}" \
    "$AUTH_SERVER/api/v1/usage"
  assert_success
  assert_output "401"
}

@test "billing: unauthenticated GET /api/v1/usage/current returns 401" {
  run curl -s -o /dev/null -w "%{http_code}" \
    "$AUTH_SERVER/api/v1/usage/current"
  assert_success
  assert_output "401"
}

@test "billing: unauthenticated GET /api/v1/usage/peak returns 401" {
  run curl -s -o /dev/null -w "%{http_code}" \
    "$AUTH_SERVER/api/v1/usage/peak"
  assert_success
  assert_output "401"
}

@test "billing: unauthenticated GET /api/v1/usage/export returns 401" {
  run curl -s -o /dev/null -w "%{http_code}" \
    "$AUTH_SERVER/api/v1/usage/export"
  assert_success
  assert_output "401"
}

# ========================================
# Billing Subscription — No Active Subscription
# ========================================

@test "billing: GET /api/v1/billing/subscription with no subscription" {
  run curl -s -H "X-API-Key: $BILLING_USER_KEY" \
    "$AUTH_SERVER/api/v1/billing/subscription"
  assert_success
  assert_output --partial '"status":"none"'
}

# ========================================
# Billing Subscription — Unauthenticated Requests (expect 401)
# ========================================

@test "billing: unauthenticated GET /api/v1/billing/subscription returns 401" {
  run curl -s -o /dev/null -w "%{http_code}" \
    "$AUTH_SERVER/api/v1/billing/subscription"
  assert_success
  assert_output "401"
}

@test "billing: unauthenticated POST /api/v1/billing/checkout returns 401" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST \
    "$AUTH_SERVER/api/v1/billing/checkout"
  assert_success
  assert_output "401"
}

@test "billing: unauthenticated POST /api/v1/billing/portal returns 401" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST \
    "$AUTH_SERVER/api/v1/billing/portal"
  assert_success
  assert_output "401"
}

# ========================================
# Billing Subscription — Simulated Checkout + Active Subscription
# ========================================
# In E2E tests we cannot complete a real Stripe Checkout (requires browser).
# Instead we simulate the webhook effect by inserting a subscription directly,
# then verify the subscription API returns it correctly.

@test "billing: simulate checkout — insert active subscription" {
  BILLING_USER_ID=$(cat "$BATS_TMPDIR/billing_user_id.txt")

  run run_sql_cmd "INSERT INTO billing_subscriptions (user_id, stripe_subscription_id, stripe_customer_id, status, plan_name, current_period_start, current_period_end, cancel_at_period_end)
     VALUES ('${BILLING_USER_ID}', 'sub_test_billing_e2e', 'cus_test_billing_e2e', 'active', 'storage', NOW(), NOW() + INTERVAL '30 days', false);"
  assert_success
  assert_output --partial "INSERT"
}

@test "billing: GET /api/v1/billing/subscription returns active subscription" {
  run curl -s -H "X-API-Key: $BILLING_USER_KEY" \
    "$AUTH_SERVER/api/v1/billing/subscription"
  assert_success
  assert_output --partial '"status":"active"'
  assert_output --partial '"planName":"storage"'
  assert_output --partial '"stripeSubscriptionId":"sub_test_billing_e2e"'
}

@test "billing: subscription response includes period dates" {
  run curl -s -H "X-API-Key: $BILLING_USER_KEY" \
    "$AUTH_SERVER/api/v1/billing/subscription"
  assert_success
  assert_output --partial '"currentPeriodStart"'
  assert_output --partial '"currentPeriodEnd"'
  assert_output --partial '"cancelAtPeriodEnd":false'
}

@test "billing: simulate subscription canceled at period end" {
  BILLING_USER_ID=$(cat "$BATS_TMPDIR/billing_user_id.txt")

  run run_sql_cmd "UPDATE billing_subscriptions SET cancel_at_period_end = true, updated_at = NOW()
     WHERE user_id = '${BILLING_USER_ID}';"
  assert_success
}

@test "billing: subscription shows cancelAtPeriodEnd true" {
  run curl -s -H "X-API-Key: $BILLING_USER_KEY" \
    "$AUTH_SERVER/api/v1/billing/subscription"
  assert_success
  assert_output --partial '"cancelAtPeriodEnd":true'
  assert_output --partial '"status":"active"'
}

@test "billing: simulate subscription fully canceled" {
  BILLING_USER_ID=$(cat "$BATS_TMPDIR/billing_user_id.txt")

  run run_sql_cmd "UPDATE billing_subscriptions SET status = 'canceled', canceled_at = NOW(), updated_at = NOW()
     WHERE user_id = '${BILLING_USER_ID}';"
  assert_success
}

@test "billing: subscription shows canceled status" {
  run curl -s -H "X-API-Key: $BILLING_USER_KEY" \
    "$AUTH_SERVER/api/v1/billing/subscription"
  assert_success
  assert_output --partial '"status":"canceled"'
  assert_output --partial '"canceledAt"'
}

@test "billing: reset subscription to active for remaining tests" {
  BILLING_USER_ID=$(cat "$BATS_TMPDIR/billing_user_id.txt")

  run run_sql_cmd "UPDATE billing_subscriptions SET status = 'active', cancel_at_period_end = false, canceled_at = NULL, updated_at = NOW()
     WHERE user_id = '${BILLING_USER_ID}';"
  assert_success
}

# ========================================
# Usage Reporting — Database Fields
# ========================================

@test "billing: verify usage_reported_for_period is null initially" {
  BILLING_USER_ID=$(cat "$BATS_TMPDIR/billing_user_id.txt")

  run run_sql_raw "SELECT usage_reported_for_period IS NULL FROM billing_subscriptions WHERE user_id = '${BILLING_USER_ID}';"
  assert_success
  RESULT=$(echo "$output" | tr -d '[:space:]')
  [[ "$RESULT" == "t" ]]
}

@test "billing: simulate usage reported — mark as reported" {
  BILLING_USER_ID=$(cat "$BATS_TMPDIR/billing_user_id.txt")

  run run_sql_cmd "UPDATE billing_subscriptions SET usage_reported_for_period = NOW(), updated_at = NOW()
     WHERE user_id = '${BILLING_USER_ID}';"
  assert_success
}

@test "billing: verify usage_reported_for_period is set" {
  BILLING_USER_ID=$(cat "$BATS_TMPDIR/billing_user_id.txt")

  run run_sql_raw "SELECT usage_reported_for_period IS NOT NULL FROM billing_subscriptions WHERE user_id = '${BILLING_USER_ID}';"
  assert_success
  RESULT=$(echo "$output" | tr -d '[:space:]')
  [[ "$RESULT" == "t" ]]
}

# ========================================
# Webhook Endpoint — Signature Validation
# ========================================

@test "billing: webhook endpoint rejects missing signature" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d '{"type":"test"}' \
    "$AUTH_SERVER/api/v1/billing/webhook"
  assert_success
  assert_output "400"
}

@test "billing: webhook endpoint rejects bad signature" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -H "Stripe-Signature: t=1234567890,v1=invalidsignature" \
    -d '{"type":"test"}' \
    "$AUTH_SERVER/api/v1/billing/webhook"
  assert_success
  assert_output "400"
}

# ========================================
# Database Verification
# ========================================

@test "billing: verify storage_snapshots table has data" {
  BILLING_USER_ID=$(cat "$BATS_TMPDIR/billing_user_id.txt")

  run run_sql_raw "SELECT COUNT(*) FROM storage_snapshots WHERE user_id = '${BILLING_USER_ID}';"
  assert_success
  COUNT=$(echo "$output" | tr -d '[:space:]')
  [[ "$COUNT" -ge 3 ]]
}

@test "billing: verify snapshot repo_details contains test repo" {
  BILLING_USER_ID=$(cat "$BATS_TMPDIR/billing_user_id.txt")

  run run_sql_raw "SELECT repo_details FROM storage_snapshots WHERE user_id = '${BILLING_USER_ID}' ORDER BY recorded_at DESC LIMIT 1;"
  assert_success
  assert_output --partial "billinguser/test-repo"
}

@test "billing: verify peak bytes across snapshots" {
  BILLING_USER_ID=$(cat "$BATS_TMPDIR/billing_user_id.txt")

  run run_sql_raw "SELECT MAX(total_bytes) FROM storage_snapshots WHERE user_id = '${BILLING_USER_ID}';"
  assert_success
  PEAK=$(echo "$output" | tr -d '[:space:]')
  [[ "$PEAK" -eq 2097152 ]]
}

# ========================================
# Cleanup
# ========================================

@test "billing: cleanup storage snapshots" {
  BILLING_USER_ID=$(cat "$BATS_TMPDIR/billing_user_id.txt")

  run run_sql_cmd "DELETE FROM storage_snapshots WHERE user_id = '${BILLING_USER_ID}';"
  assert_success
}

@test "billing: cleanup billing subscriptions" {
  BILLING_USER_ID=$(cat "$BATS_TMPDIR/billing_user_id.txt")

  run run_sql_cmd "DELETE FROM billing_subscriptions WHERE user_id = '${BILLING_USER_ID}';"
  assert_success
}

@test "billing: cleanup API keys and test user" {
  run run_sql_cmd "DELETE FROM api_keys WHERE user_id IN (SELECT id FROM users WHERE github_login = 'billinguser');
     DELETE FROM whitelisted_users WHERE user_id IN (SELECT id FROM users WHERE github_login = 'billinguser');
     DELETE FROM namespaces WHERE name = 'billinguser';
     DELETE FROM users WHERE github_login = 'billinguser';"
  assert_success
}
