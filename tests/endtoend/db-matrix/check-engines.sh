#!/usr/bin/env bash
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Local harness for the per-image tables in db-matrix-engines.bash (#262).
# For each image given on the command line it builds the wrapper Dockerfile if
# one exists here (so a Dockerfile change can be checked before it is
# published), starts a plain container with the image's boot table, waits for
# the engine, writes the check object, restarts the container the way dit cp
# and dit checkout do, waits again and reads the object back. No dit install
# is needed, so a new engine's table can be developed here before it costs a
# full matrix run.
#
#   tests/endtoend/db-matrix/check-engines.sh postgres:16 mongo:7 ditdotdev/redis:7
#
# The workflow itself only pulls: once a wrapper Dockerfile passes here, push
# it to Docker Hub for both runner architectures before the matrix runs:
#
#   docker buildx build --platform linux/amd64,linux/arm64 \
#     -t ditdotdev/<image>:<tag> -f <image>-<tag>.Dockerfile --push .
#
# Prints one PASS / FAIL / NO-CLIENT line per image and exits non-zero if any
# image failed.
set -u

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# docker on Windows is a native binary: hand it a Windows path as the build
# context when running from Git Bash (pwd -W), a POSIX path everywhere else.
context="$(cd "$here" && { pwd -W 2>/dev/null || pwd; })"
# shellcheck source=db-matrix-engines.bash
source "$here/db-matrix-engines.bash"

failed=0
for img in "$@"; do
  name="${img##*/}"
  dockerfile="$here/${name/:/-}.Dockerfile"
  c="dit-check-${name//[^A-Za-z0-9_.-]/-}"
  docker rm -f "$c" >/dev/null 2>&1 || true

  if [[ -f "$dockerfile" ]]; then
    docker build -q -t "$img" -f "$dockerfile" "$context" >/dev/null || { echo "FAIL      $img (build)"; failed=1; continue; }
  else
    docker pull -q "$img" >/dev/null || { echo "FAIL      $img (pull)"; failed=1; continue; }
  fi

  # shellcheck disable=SC2046  # the tables are deliberately word-split
  docker run -d --name "$c" $(db_docker_flags "$img") $(db_env_args "$img") "$img" >/dev/null \
    || { echo "FAIL      $img (run)"; failed=1; continue; }

  result=""
  if ! db_wait_ready "$c" "$img"; then
    result="FAIL      $img (not ready)"
  else
    db_write "$c" "$img" >/dev/null 2>&1
    rc=$?
    if [[ $rc -eq $DB_NO_CLIENT ]]; then
      result="NO-CLIENT $img"
    elif [[ $rc -ne 0 ]]; then
      db_write "$c" "$img" 2>&1 | tail -5 | sed 's/^/    | /'
      result="FAIL      $img (write rc=$rc)"
    elif ! db_verify "$c" "$img"; then
      result="FAIL      $img (verify before restart)"
    else
      docker stop -t 60 "$c" >/dev/null && docker start "$c" >/dev/null
      if ! db_wait_ready "$c" "$img"; then
        result="FAIL      $img (not ready after restart)"
      elif ! db_verify "$c" "$img"; then
        result="FAIL      $img (verify after restart)"
      else
        result="PASS      $img"
      fi
    fi
  fi
  echo "$result"
  [[ "$result" == FAIL* ]] && { failed=1; docker logs --tail 15 "$c" 2>&1 | sed 's/^/    | /'; }
  docker rm -f "$c" >/dev/null 2>&1 || true
done
exit $failed
