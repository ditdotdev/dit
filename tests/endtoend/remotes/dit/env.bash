#!/usr/bin/env bash

# Environment configuration for dit E2E server tests.
# Usage:
#   make e2e-server              # runs against local DEV (default)
#   ENV=PROD make e2e-server     # runs against production

ENV="${ENV:-DEV}"

# API key is the same in both environments
export DIT_API_KEY="${DIT_API_KEY:?set DIT_API_KEY (CI secret / local env) — see DEVELOPING.md}"

if [[ "$ENV" == "PROD" ]]; then
  # ---- Production (ECS on AWS) ----
  export GATEWAY="https://dit.dev"
  export WEB_UI="https://dit.dev"
  export AUTH_SERVER="https://dit.dev"
  export REMOTE_URL="https://dit.dev"
  # In prod, nginx fronts the api-gateway and answers /health itself with a
  # static "OK\n" (LB liveness; see dit-remote-server user-data.sh `location
  # /health { return 200 "OK\n"; }`). The api-gateway's own "healthy" body
  # (what DEV hits directly at :8080) is shadowed by nginx, so PROD must expect
  # "OK", not "healthy".
  export HEALTH_EXPECT="OK"

  # Org/namespace prefixes (production uses e2e- prefix to avoid collisions)
  export TEST_ORG="e2e-testorg"
  export WEB_TEST_ORG="e2e-webtest"
  export AUTH_TEST_ORG="authtest"
  export APIKEY_TEST_ORG="apikeytest"

  # Download API test version (production may lag behind dev)
  export DOWNLOAD_TEST_VERSION="${DOWNLOAD_TEST_VERSION:-v1.9.8}"

  # RDS / SSH access for database commands
  export EC2_HOST="${EC2_HOST:-ec2-user@100.22.249.49}"
  export SSH_KEY="${SSH_KEY:-c:/dev/dit/dit-remote-server/dit-ecs-host.pem}"
  # The RDS instance kept its datadatdat name through the rebrand (like the ECS
  # resources); user + database are datadatdat too. These match the live
  # /dit/prod/database/url secret.
  export RDS_ENDPOINT="${RDS_ENDPOINT:-<prod-rds-endpoint>}"
  export RDS_PASSWORD="${RDS_PASSWORD:?set RDS_PASSWORD (CI secret / local env) for PROD tests}"
  export RDS_USER="${RDS_USER:-datadatdat}"
  export RDS_DATABASE="${RDS_DATABASE:-datadatdat}"

  # SSH opts: StrictHostKeyChecking=no + UserKnownHostsFile=/dev/null avoid the
  # interactive host-key prompt when the EC2 host is replaced; BatchMode fails
  # fast instead of hanging.
  # LogLevel=ERROR silences the "Permanently added <host> to known hosts"
  # warning that UserKnownHostsFile=/dev/null prints on every connection,
  # which would otherwise pollute captured psql output.
  SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o BatchMode=yes -o ConnectTimeout=10 -o LogLevel=ERROR"
  # The ECS host reaches RDS but has no psql client, so run psql via a
  # throwaway postgres container on the instance's network (--network host) so
  # it can resolve the VPC-internal RDS endpoint. Pull only if missing, quietly,
  # so docker progress never pollutes the captured command output.
  PSQL_IMAGE="postgres:17-alpine"
  _psql_via_host() {
    ssh $SSH_OPTS -i "${SSH_KEY}" "${EC2_HOST}" \
      "docker image inspect ${PSQL_IMAGE} >/dev/null 2>&1 || docker pull -q ${PSQL_IMAGE} >/dev/null 2>&1; docker run -i --rm --network host -e PGPASSWORD='${RDS_PASSWORD}' ${PSQL_IMAGE} psql -h ${RDS_ENDPOINT} -U ${RDS_USER} -d ${RDS_DATABASE} $*"
  }

  # Helper: execute SQL against production RDS via the ECS host
  run_sql() {
    _psql_via_host "$@" </dev/null
  }

  # Helper: execute SQL (with flags like -t -A) against production RDS
  # Uses stdin to avoid SSH double-quote escaping issues with JSON literals
  run_sql_raw() {
    echo "$1" | _psql_via_host -t -A
  }

  # Helper: execute SQL command (returns table output) against production RDS
  # Uses stdin to avoid SSH double-quote escaping issues with JSON literals
  run_sql_cmd() {
    echo "$1" | _psql_via_host
  }

  # Minio is not available in production (uses real S3)
  has_minio() { return 1; }

  # Container checks go through SSH
  check_container_running() {
    local filter="$1"
    ssh $SSH_OPTS -i "${SSH_KEY}" "${EC2_HOST}" "docker ps --filter name=${filter} --format '{{.Status}}'"
  }

else
  # ---- DEV (local docker-compose) ----
  export GATEWAY="http://127.0.0.1:8080"
  export WEB_UI="http://127.0.0.1:3000"
  export AUTH_SERVER="http://127.0.0.1:8085"
  export REMOTE_URL="http://dit-api-gateway:8080"
  export DIT_HOST_GATEWAY="${DIT_HOST_GATEWAY:-http://127.0.0.1:8080}"
  export HEALTH_EXPECT="healthy"

  # Org/namespace prefixes
  export TEST_ORG="testorg"
  export WEB_TEST_ORG="webtest"
  export AUTH_TEST_ORG="authtest"
  export APIKEY_TEST_ORG="apikeytest"

  # Download API test version (local dev releases bucket is seeded with v1.9.3)
  export DOWNLOAD_TEST_VERSION="${DOWNLOAD_TEST_VERSION:-v1.9.8}"

  # Helper: execute SQL against local postgres container
  run_sql() {
    docker exec dit-postgres psql -U dit -d dit "$@"
  }

  # Helper: execute SQL (raw output, no headers) against local postgres
  run_sql_raw() {
    docker exec dit-postgres psql -U dit -d dit -t -A -c "$1"
  }

  # Helper: execute SQL command against local postgres
  run_sql_cmd() {
    docker exec dit-postgres psql -U dit -d dit -c "$1"
  }

  # Minio is available in dev
  has_minio() { return 0; }

  # Container checks via local docker
  check_container_running() {
    local filter="$1"
    docker ps --filter "name=${filter}" --format '{{.Status}}'
  }
fi

# Convenience: check if running in prod
is_prod() { [[ "$ENV" == "PROD" ]]; }
is_dev()  { [[ "$ENV" != "PROD" ]]; }

# Retry a command until its output contains expected text.
# Sets $output and $status like BATS `run`.
# Usage: run_until_output_contains <expected_text> <max_attempts> <delay_seconds> <command...>
run_until_output_contains() {
  local expected="$1"
  local max_attempts="$2"
  local delay="$3"
  shift 3
  local attempt=1
  while [ "$attempt" -le "$max_attempts" ]; do
    output=$("$@" 2>&1) && status=0 || status=$?
    if echo "$output" | grep -qF "$expected"; then
      return 0
    fi
    if [ "$attempt" -lt "$max_attempts" ]; then
      sleep "$delay"
    fi
    attempt=$((attempt + 1))
  done
  return 1
}
