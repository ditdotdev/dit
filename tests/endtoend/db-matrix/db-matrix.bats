#!/usr/bin/env bats

# Load shared test helpers
load '../test_helper'

# S3 URI base for testing — unique per run to avoid collisions between
# concurrent runs. See issue datadatdat/datadatdat-server#118.
RUN_SUFFIX="${E2E_RUN_SUFFIX:-local}"
URI_BASE="s3://datadatdat-testdata/e2etest/${RUN_SUFFIX}"

# Helper to run tests for a specific database version
test_database() {
  local db_version="$1"
  
  # Extract database name - handle registry paths like datadatdat/mssql-server:2017
  local db
  if [[ "$db_version" == *"/"* ]]; then
    # Has registry path - extract just the image name
    db="${db_version##*/}"  # Get everything after last /
    db="${db%%:*}"          # Remove version tag
  else
    # Simple format like postgres:16
    db="${db_version%%:*}"
  fi
  
  local version="${db_version#*:}"
  local repo_name="${db}-test"
  # Use unique S3 path for each database version to avoid parallel test conflicts
  # Sanitize the version to be filesystem-safe (replace / and : with -)
  local safe_version="${version//\//-}"
  safe_version="${safe_version//:/-}"
  local URI="${URI_BASE}/${db}-${safe_version}"
  
  echo "  Testing $db:$version"
  
  # Run - handle special database requirements
  if [[ "$db_version" == *"mssql"* ]]; then
    # SQL Server requires EULA acceptance and SA password
    "$D3" run -n "$repo_name" \
      -e "ACCEPT_EULA=Y" \
      -e "MSSQL_SA_PASSWORD=YourStrong!Passw0rd" \
      -e "MSSQL_PID=Developer" \
      "$db_version" || return 1
  elif [[ "$db_version" == *"db2"* ]]; then
    # IBM Db2 requires both --privileged (to remount /database with suid) and
    # --disable-port-mapping (port 22/SSH conflicts with runner's sshd).
    "$D3" run -n "$repo_name" \
      --privileged \
      --disable-port-mapping \
      -e "LICENSE=accept" \
      -e "DB2INST1_PASSWORD=YourStrong!Passw0rd" \
      -e "DBNAME=testdb" \
      "$db_version" || return 1
  elif [[ "$db_version" == *"dynamodb"* ]]; then
    # DynamoDB Local - disable port mapping to avoid conflicts
    "$D3" run -n "$repo_name" \
      --disable-port-mapping \
      "$db_version" || return 1
  elif [[ "$db_version" == *"oracle"* ]]; then
    # Oracle requires password and has no exposed ports
    "$D3" run -n "$repo_name" \
      --disable-port-mapping \
      -e "ORACLE_PASSWORD=YourStrong!Passw0rd" \
      "$db_version" || return 1
  elif [[ "$db_version" == *"minio"* ]]; then
    # MinIO requires root user credentials
    "$D3" run -n "$repo_name" \
      -e "MINIO_ROOT_USER=minioadmin" \
      -e "MINIO_ROOT_PASSWORD=minioadmin" \
      "$db_version" || return 1
  elif [[ "$db_version" == *"arangodb"* ]]; then
    # ArangoDB requires auth configuration
    "$D3" run -n "$repo_name" \
      -e "ARANGO_NO_AUTH=1" \
      "$db_version" || return 1
  elif [[ "$db_version" == *"tigergraph"* ]]; then
    # TigerGraph runs sshd on port 22 (conflicts with runner).
    # Services must be started manually via gadmin after container launch.
    "$D3" run -n "$repo_name" \
      --disable-port-mapping \
      "$db_version" || return 1
    # Wait for container to be running, then start TigerGraph services
    local tg_timeout=30
    local tg_elapsed=0
    while ! docker ps --filter "name=$repo_name" --format "{{.Status}}" | grep -q "Up"; do
      if [ $tg_elapsed -ge $tg_timeout ]; then
        echo "Timeout waiting for container $repo_name to start"
        return 1
      fi
      sleep 1
      ((tg_elapsed++))
    done
    echo "  Starting TigerGraph services..."
    local tg_gadmin=/home/tigergraph/tigergraph/app/cmd/gadmin
    echo "  Starting TigerGraph infrastructure..."
    docker exec -u tigergraph "$repo_name" bash -c "$tg_gadmin start infra" || return 1
    docker exec -u tigergraph "$repo_name" bash -c "$tg_gadmin start all" || return 1
    echo "  TigerGraph services started"
  else
    "$D3" run -n "$repo_name" "$db_version" || return 1
  fi
  
  # Commit
  local commit_output
  commit_output=$("$D3" commit -m "Test Commit" "$repo_name") || return 1
  local commit_guid=$(echo "$commit_output" | grep -o 'Commit [a-f0-9]\+' | awk '{print $2}')
  [[ -n "$commit_guid" ]] || return 1
  
  # Checkout
  "$D3" checkout --commit "$commit_guid" "$repo_name" || return 1
  # Verify container is running after checkout
  local timeout=30
  local elapsed=0
  while ! docker ps --filter "name=$repo_name" --format "{{.Status}}" | grep -q "Up"; do
    if [ $elapsed -ge $timeout ]; then
      echo "Timeout waiting for container $repo_name to be running after checkout"
      return 1
    fi
    sleep 1
    ((elapsed++))
  done
  
  # Remote add
  "$D3" remote add -r s3 "$URI" "$repo_name" || return 1
  
  # Remote ls
  "$D3" remote ls "$repo_name" | grep -q "$URI" || return 1
  
  # Remote log - missing commit
  ! "$D3" remote log "$repo_name" | grep -q "Commit $commit_guid" || return 1
  
  # Push
  "$D3" push -r s3 -c "$commit_guid" "$repo_name" || return 1

  # Remote log - has commit
  "$D3" remote log "$repo_name" | grep -q "Commit $commit_guid" || return 1
  
  # Delete commit
  "$D3" delete -c "$commit_guid" "$repo_name" || return 1
  
  # Log - missing commit
  ! "$D3" log "$repo_name" | grep -q "$commit_guid" || return 1
  
  # Pull
  "$D3" pull -c "$commit_guid" -r s3 "$repo_name" || return 1
  
  # Log - has commit
  "$D3" log "$repo_name" | grep -q "commit $commit_guid" || return 1
  
  # Rm
  "$D3" rm -f "$repo_name" || return 1
  # Verify repository is removed from server
  echo "DEBUG: Verifying $repo_name is removed from server..."
  local timeout=5
  local elapsed=0
  while true; do
    # Check PostgreSQL directly for the repository
    if docker exec datadatdat-docker-server psql -U postgres -d datadatdat -tAc "SELECT name FROM repositories WHERE name='$repo_name';" | grep -q "$repo_name"; then
      echo "DEBUG: Repository $repo_name still exists in PostgreSQL after ${elapsed}s"
      if [ $elapsed -ge $timeout ]; then
        echo "ERROR: Timeout waiting for repository $repo_name metadata to be removed from server (waited ${timeout}s)"
        echo "DEBUG: Current repositories in PostgreSQL:"
        docker exec datadatdat-docker-server psql -U postgres -d datadatdat -c "SELECT name FROM repositories;"
        return 1
      fi
      sleep 1
      ((elapsed++))
    else
      echo "DEBUG: Repository $repo_name successfully removed from PostgreSQL after ${elapsed}s"
      break
    fi
  done

  # Final check right before clone
  echo "DEBUG: [$(date +%H:%M:%S.%3N)] Final PostgreSQL check before clone..."
  if docker exec datadatdat-docker-server psql -U postgres -d datadatdat -tAc "SELECT name FROM repositories WHERE name='$repo_name';" | grep -q "$repo_name"; then
    echo "ERROR: [$(date +%H:%M:%S.%3N)] Repository $repo_name EXISTS in PostgreSQL right before clone!"
  else
    echo "DEBUG: [$(date +%H:%M:%S.%3N)] Repository $repo_name NOT in PostgreSQL - proceeding with clone"
  fi

  # Clone
  if [[ "$db_version" == *"dynamodb"* ]]; then
    "$D3" clone -n "$repo_name" -P "$URI" || return 1
  else
    "$D3" clone -n "$repo_name" "$URI" || return 1
  fi
  # Verify container is running after clone
  local timeout=30
  local elapsed=0
  while ! docker ps --filter "name=$repo_name" --format "{{.Status}}" | grep -q "Up"; do
    if [ $elapsed -ge $timeout ]; then
      echo "Timeout waiting for container $repo_name to be running after clone"
      return 1
    fi
    sleep 1
    ((elapsed++))
  done
  
  # Note: The original YAML test expected duplicate clone to fail, but current implementation
  # allows multiple repositories to clone from the same S3 URI (which is valid behavior)
  # Just verify the cloned repository exists
  "$D3" log "$repo_name" | grep -q "commit" || return 1
  
  # Log - has commit (after clone)
  "$D3" log "$repo_name" | grep -q "commit $commit_guid" || return 1
  
  # Remove S3 assets
  aws s3 rm "$URI" --recursive || return 1
  
  # Rm (final)
  "$D3" rm -f "$repo_name" || return 1
  # Verify repository metadata and container are fully removed
  local timeout=30
  local elapsed=0
  while curl -s http://localhost:5001/repositories | grep -q "\"$repo_name\"" || docker ps -a --filter "name=$repo_name" --format "{{.Names}}" | grep -q "^$repo_name$"; do
    if [ $elapsed -ge $timeout ]; then
      echo "Timeout waiting for final cleanup of $repo_name"
      return 1
    fi
    sleep 1
    ((elapsed++))
  done

  echo "  ✓ $db:$version completed"
}

@test "database matrix test" {
  # If DATABASE_VERSION is set (from GitHub Actions matrix), use it
  # Otherwise, run all databases locally for backward compatibility
  if [[ -n "$DATABASE_VERSION" ]]; then
    test_database "$DATABASE_VERSION"
  else
    echo "Running all databases (local mode)"
    test_database "postgres:16"
    test_database "postgres:15"
    test_database "mongo:7"
    test_database "mongo:6"
  fi
}

# Cleanup after all tests
teardown_file() {
  # Best effort cleanup
  # Clean up based on DATABASE_VERSION if set, otherwise clean all
  if [[ -n "$DATABASE_VERSION" ]]; then
    local db="${DATABASE_VERSION%%:*}"
    local version="${DATABASE_VERSION#*:}"
    "$D3" rm -f "${db}-test" 2>/dev/null || true
    "$D3" rm -f "${db}-test2" 2>/dev/null || true
    # Clean up S3 assets for this specific database
    aws s3 rm "${URI_BASE}/${db}-${version}" --recursive 2>/dev/null || true
  else
    for db_version in "postgres:16" "postgres:15" "mongo:7" "mongo:6"; do
      local db="${db_version%%:*}"
      local version="${db_version#*:}"
      "$D3" rm -f "${db}-test" 2>/dev/null || true
      "$D3" rm -f "${db}-test2" 2>/dev/null || true
      # Clean up S3 assets for this specific database
      aws s3 rm "${URI_BASE}/${db}-${version}" --recursive 2>/dev/null || true
    done
  fi
  cleanup_stale_aws_processes
}
