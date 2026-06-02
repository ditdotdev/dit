#!/usr/bin/env bash

# Environment configuration for dit E2E server tests.
# Usage:
#   make e2e-server              # runs against local DEV (default)
#   ENV=PROD make e2e-server     # runs against production

ENV="${ENV:-DEV}"

# API key is the same in both environments
export DIT_API_KEY="${DIT_API_KEY:-02b31569a9052bc4b3cf1c3819d4fc048d34c96eca21f2b8e2359b5ecdfec93a}"

if [[ "$ENV" == "PROD" ]]; then
  # ---- Production (ECS on AWS) ----
  export GATEWAY="https://dit.dev"
  export WEB_UI="https://dit.dev"
  export AUTH_SERVER="https://dit.dev"
  export REMOTE_URL="https://dit.dev"
  export HEALTH_EXPECT="healthy"

  # Org/namespace prefixes (production uses e2e- prefix to avoid collisions)
  export TEST_ORG="e2e-testorg"
  export WEB_TEST_ORG="e2e-webtest"
  export AUTH_TEST_ORG="authtest"
  export APIKEY_TEST_ORG="apikeytest"

  # Download API test version (production may lag behind dev)
  export DOWNLOAD_TEST_VERSION="${DOWNLOAD_TEST_VERSION:-v1.9.2}"

  # RDS / SSH access for database commands
  export EC2_HOST="${EC2_HOST:-ec2-user@100.22.249.49}"
  export SSH_KEY="${SSH_KEY:-c:/dev/ditdotdev/dit-remote-server/dit-ecs-host.pem}"
  export RDS_ENDPOINT="${RDS_ENDPOINT:-dit-postgres-prod.cdsu2uig4uc0.us-west-2.rds.amazonaws.com}"
  export RDS_PASSWORD="${RDS_PASSWORD:-v1fsAuY1CAUhtVixni2BRqiQtET4bJwf}"
  export RDS_USER="${RDS_USER:-dit}"
  export RDS_DATABASE="${RDS_DATABASE:-dit}"

  # Helper: execute SQL against production RDS via SSH tunnel
  run_sql() {
    ssh -i "${SSH_KEY}" "${EC2_HOST}" "PGPASSWORD='${RDS_PASSWORD}' psql -h ${RDS_ENDPOINT} -U ${RDS_USER} -d ${RDS_DATABASE} $*"
  }

  # Helper: execute SQL (with flags like -t -A) against production RDS
  # Uses stdin to avoid SSH double-quote escaping issues with JSON literals
  run_sql_raw() {
    echo "$1" | ssh -i "${SSH_KEY}" "${EC2_HOST}" "PGPASSWORD='${RDS_PASSWORD}' psql -h ${RDS_ENDPOINT} -U ${RDS_USER} -d ${RDS_DATABASE} -t -A"
  }

  # Helper: execute SQL command (returns table output) against production RDS
  # Uses stdin to avoid SSH double-quote escaping issues with JSON literals
  run_sql_cmd() {
    echo "$1" | ssh -i "${SSH_KEY}" "${EC2_HOST}" "PGPASSWORD='${RDS_PASSWORD}' psql -h ${RDS_ENDPOINT} -U ${RDS_USER} -d ${RDS_DATABASE}"
  }

  # Minio is not available in production (uses real S3)
  has_minio() { return 1; }

  # Container checks go through SSH
  check_container_running() {
    local filter="$1"
    ssh -i "${SSH_KEY}" "${EC2_HOST}" "docker ps --filter name=${filter} --format '{{.Status}}'"
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

  # Download API test version
  export DOWNLOAD_TEST_VERSION="${DOWNLOAD_TEST_VERSION:-v1.9.2}"

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
