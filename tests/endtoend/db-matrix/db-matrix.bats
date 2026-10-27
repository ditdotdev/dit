#!/usr/bin/env bats
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# Load shared test helpers
load '../test_helper'

# S3 URI base for testing — unique per run to avoid collisions between
# concurrent runs. See issue ditdotdev/dit-server#118.
RUN_SUFFIX="${E2E_RUN_SUFFIX:-local}"
URI_BASE="s3://dit-testdata/e2etest/${RUN_SUFFIX}"

# Helper to run tests for a specific database version
test_database() {
  local db_version="$1"
  
  # Extract database name - handle registry paths like ditdotdev/mssql-server:2017
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
      tg_elapsed=$((tg_elapsed + 1))
    done
    echo "  Starting TigerGraph services..."
    local tg_gadmin=/home/tigergraph/tigergraph/app/cmd/gadmin
    echo "  Starting TigerGraph infrastructure..."
    docker exec -u tigergraph "$repo_name" bash -c "$tg_gadmin start infra" || return 1
    docker exec -u tigergraph "$repo_name" bash -c "$tg_gadmin start all" || return 1
    echo "  TigerGraph services started"
  elif [[ "$db_version" == postgres:* ]]; then
    # The official image exits at once unless a superuser password policy is
    # set; trust auth keeps the matrix free of secrets.
    "$D3" run -n "$repo_name"       -e "POSTGRES_HOST_AUTH_METHOD=trust"       "$db_version" || return 1
  elif [[ "$db_version" == mysql:* ]]; then
    # MySQL refuses to start without a root password policy; allow empty.
    "$D3" run -n "$repo_name"       -e "MYSQL_ALLOW_EMPTY_PASSWORD=yes"       "$db_version" || return 1
  elif [[ "$db_version" == mariadb:* ]]; then
    # MariaDB likewise needs an explicit root password policy.
    "$D3" run -n "$repo_name"       -e "MARIADB_ALLOW_EMPTY_ROOT_PASSWORD=yes"       "$db_version" || return 1
  else
    "$D3" run -n "$repo_name" "$db_version" || return 1
  fi
  wait_for_up "$repo_name" 60 || return 1

  # dit cp: load a marker directory from the host into every volume the image
  # declares. dit cp stops the container, copies, and starts it again, so this
  # also proves the engine tolerates a foreign entry in its data directory and
  # comes back up. The marker is read back after checkout below, so the commit
  # the rest of this flow pushes, pulls and clones carries real user data.
  # The host directory lives under the repo (not /tmp): on Windows Docker
  # Desktop only paths under the user's drive are visible to docker cp.
  local cp_src="${REPO_ROOT}/.dit-cp-check-${db}"
  rm -rf "$cp_src"
  mkdir -p "$cp_src/.dit-cp-check"
  echo "$db_version" > "$cp_src/.dit-cp-check/marker"
  local cp_vols
  cp_vols=$(docker image inspect "$db_version" --format '{{range $p, $_ := .Config.Volumes}}{{$p}} {{end}}')
  local cp_vol
  for cp_vol in $cp_vols; do
    "$D3" cp -s "$cp_src" -d "$cp_vol" "$repo_name" || { rm -rf "$cp_src"; return 1; }
  done
  rm -rf "$cp_src"
  wait_for_up "$repo_name" 60 || return 1

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
    elapsed=$((elapsed + 1))
  done
  # The dit cp marker must have survived commit and checkout in every volume.
  for cp_vol in $cp_vols; do
    local cp_got
    cp_got=$(read_marker "$repo_name" "$cp_vol/.dit-cp-check/marker")
    if [[ "$cp_got" != "$db_version" ]]; then
      echo "  dit cp marker missing from $cp_vol in $repo_name after checkout"
      return 1
    fi
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
    if docker exec dit-docker-server psql -U postgres -d dit -tAc "SELECT name FROM repositories WHERE name='$repo_name';" | grep -q "$repo_name"; then
      echo "DEBUG: Repository $repo_name still exists in PostgreSQL after ${elapsed}s"
      if [ $elapsed -ge $timeout ]; then
        echo "ERROR: Timeout waiting for repository $repo_name metadata to be removed from server (waited ${timeout}s)"
        echo "DEBUG: Current repositories in PostgreSQL:"
        docker exec dit-docker-server psql -U postgres -d dit -c "SELECT name FROM repositories;"
        return 1
      fi
      sleep 1
      elapsed=$((elapsed + 1))
    else
      echo "DEBUG: Repository $repo_name successfully removed from PostgreSQL after ${elapsed}s"
      break
    fi
  done

  # Final check right before clone
  echo "DEBUG: [$(date +%H:%M:%S.%3N)] Final PostgreSQL check before clone..."
  if docker exec dit-docker-server psql -U postgres -d dit -tAc "SELECT name FROM repositories WHERE name='$repo_name';" | grep -q "$repo_name"; then
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
    elapsed=$((elapsed + 1))
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
    elapsed=$((elapsed + 1))
  done

  # Migrate an unmanaged container of the same image into a repository.
  test_migrate "$db_version" "$db" || return 1

  echo "  ✓ $db:$version completed"
}

# Environment every image needs to boot when started outside dit. These mirror
# the -e values the dit run branches in test_database pass; keep both in sync.
db_env_args() {
  local db_version="$1"
  if [[ "$db_version" == *"mssql"* ]]; then
    echo "-e ACCEPT_EULA=Y -e MSSQL_SA_PASSWORD=YourStrong!Passw0rd -e MSSQL_PID=Developer"
  elif [[ "$db_version" == *"db2"* ]]; then
    echo "--privileged -e LICENSE=accept -e DB2INST1_PASSWORD=YourStrong!Passw0rd -e DBNAME=testdb"
  elif [[ "$db_version" == *"oracle"* ]]; then
    echo "-e ORACLE_PASSWORD=YourStrong!Passw0rd"
  elif [[ "$db_version" == *"minio"* ]]; then
    echo "-e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin"
  elif [[ "$db_version" == *"arangodb"* ]]; then
    echo "-e ARANGO_NO_AUTH=1"
  elif [[ "$db_version" == postgres:* ]]; then
    echo "-e POSTGRES_HOST_AUTH_METHOD=trust"
  elif [[ "$db_version" == mysql:* ]]; then
    echo "-e MYSQL_ALLOW_EMPTY_PASSWORD=yes"
  elif [[ "$db_version" == mariadb:* ]]; then
    echo "-e MARIADB_ALLOW_EMPTY_ROOT_PASSWORD=yes"
  fi
}

# Print the contents of a file inside a running dit-managed container.
#
# Never use docker cp for this. docker cp mounts the container's volumes for
# the copy and unmounts them afterwards; with the dit volume driver that
# Unmount deactivates the ZFS dataset underneath the running database
# (ditdotdev/dit#260). docker exec reads through the container's own mount.
# Images without a shell (distroless) fall back to a stop, docker cp, start.
read_marker() {
  local container="$1" path="$2" out
  if out=$(docker exec "$container" cat "$path" 2>/dev/null); then
    printf '%s' "$out"
    return 0
  fi
  if docker exec "$container" cat /dev/null >/dev/null 2>&1; then
    return 1   # cat works, the file is genuinely missing
  fi
  docker stop "$container" >/dev/null 2>&1 || return 1
  out=$(MSYS_NO_PATHCONV=1 docker cp "$container:$path" - 2>/dev/null | tar -xO 2>/dev/null)
  docker start "$container" >/dev/null 2>&1
  printf '%s' "$out"
}

# Wait until a container reports Up, or fail after $2 seconds.
wait_for_up() {
  local name="$1" timeout="${2:-60}" elapsed=0
  while ! docker ps --filter "name=^${name}$" --format "{{.Status}}" | grep -q "Up"; do
    if [ "$elapsed" -ge "$timeout" ]; then
      echo "Timeout waiting for container $name to be running"
      docker logs --tail 20 "$name" 2>&1 | sed 's/^/    | /'
      return 1
    fi
    sleep 1
    elapsed=$((elapsed + 1))
  done
}

# dit migrate: start the image as a plain, unmanaged docker container, stop
# it, migrate it into a dit repository, and prove the declared volumes came
# across. The proof is a marker file dropped into every declared volume of the
# stopped source with docker cp (works on stopped containers and on images
# with no shell) and read back the same way from the migrated container. If
# the marker is missing the copy did not happen, whatever else looks healthy
# (ditdotdev/dit#256 shipped an empty database for weeks because nothing
# checked this).
test_migrate() {
  local db_version="$1"
  local db="$2"
  local src="${db}-migrate-src"
  local dest="${db}-migrated"
  # Host-side marker lives under the repo, not /tmp: on Windows Docker Desktop
  # only paths under the user's drive are visible to docker cp.
  local marker_dir="${REPO_ROOT}/.dit-migrate-check-${db}"
  rm -rf "$marker_dir"
  mkdir -p "$marker_dir"
  echo "$db_version" > "$marker_dir/marker"

  echo "  Migrating unmanaged $db_version container"
  docker rm -f "$src" >/dev/null 2>&1 || true
  "$D3" rm -f "$dest" >/dev/null 2>&1 || true

  # shellcheck disable=SC2046  # env args are deliberately word-split
  docker run -d --name "$src" $(db_env_args "$db_version") "$db_version" >/dev/null || return 1
  wait_for_up "$src" 60 || return 1

  # The volumes migrate will carry over are the ones the image declares.
  local vols
  vols=$(docker image inspect "$db_version" --format '{{range $p, $_ := .Config.Volumes}}{{$p}} {{end}}')
  if [[ -z "$vols" ]]; then
    # Nothing to migrate: dit refuses up front and must not create a repo.
    echo "  $db_version declares no VOLUME; expecting migrate to refuse"
    docker stop "$src" >/dev/null || return 1
    ! "$D3" migrate -s "$src" "$dest" || return 1
    ! "$D3" ls | grep -q "$dest" || return 1
    docker rm -f "$src" >/dev/null 2>&1 || true
    return 0
  fi

  # Give the database a moment to initialize its data directory, then stop it
  # cleanly so the files are consistent, and plant a marker in every volume.
  sleep 5
  docker stop -t 60 "$src" >/dev/null || return 1
  local vol
  local marker_host
  marker_host="$(get_docker_path "$marker_dir")"
  for vol in $vols; do
    MSYS_NO_PATHCONV=1 docker cp "$marker_host/." "$src:$vol/.dit-migrate-check" || return 1
  done

  local out
  out=$("$D3" migrate -s "$src" "$dest" 2>&1) || { echo "$out"; return 1; }
  echo "$out" | grep -q "migrated to controlled environment $dest" || { echo "$out"; return 1; }
  ! echo "$out" | grep -qi "failed to copy" || { echo "$out"; return 1; }

  wait_for_up "$dest" 60 || return 1
  for vol in $vols; do
    local got
    got=$(read_marker "$dest" "$vol/.dit-migrate-check/marker")
    if [[ "$got" != "$db_version" ]]; then
      echo "  marker missing from $vol in $dest: migrate did not carry the volume across"
      return 1
    fi
  done

  # The migrated repository must be usable as a normal dit repository.
  "$D3" commit -m "After migrate" "$dest" | grep -q "Commit" || return 1
  "$D3" log "$dest" | grep -q "Initial Migration" || return 1

  "$D3" rm -f "$dest" || return 1
  docker rm -f "$src" >/dev/null 2>&1 || true
  rm -rf "$marker_dir"
  echo "  ✓ $db_version migrate completed"
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
    "$D3" rm -f "${db}-migrated" 2>/dev/null || true
    docker rm -f "${db}-migrate-src" 2>/dev/null || true
    # Clean up S3 assets for this specific database
    aws s3 rm "${URI_BASE}/${db}-${version}" --recursive 2>/dev/null || true
  else
    for db_version in "postgres:16" "postgres:15" "mongo:7" "mongo:6"; do
      local db="${db_version%%:*}"
      local version="${db_version#*:}"
      "$D3" rm -f "${db}-test" 2>/dev/null || true
      "$D3" rm -f "${db}-test2" 2>/dev/null || true
      "$D3" rm -f "${db}-migrated" 2>/dev/null || true
      docker rm -f "${db}-migrate-src" 2>/dev/null || true
      # Clean up S3 assets for this specific database
      aws s3 rm "${URI_BASE}/${db}-${version}" --recursive 2>/dev/null || true
    done
  fi
  rm -rf "${REPO_ROOT}"/.dit-cp-check-* "${REPO_ROOT}"/.dit-migrate-check-* 2>/dev/null || true
  cleanup_stale_aws_processes
}
