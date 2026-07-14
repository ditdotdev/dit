#!/usr/bin/env bash
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1
# Local coverage that matches the CI repo-health gate.
#
# The CI workflow (ditdotdev/.github/.github/workflows/repo-health.yml)
# filters generated and non-library files out of the coverage profile
# before computing the scored %. `go test -cover ./...` does not, so the
# raw local number is usually 15-25 points lower than what the gate sees.
#
# This script reproduces the gate's filter so contributors can see the
# number the gate is going to score them against — no surprises in PR CI.
#
# Run from the repo root:
#   ./scripts/local-coverage.sh
#   make coverage
#
# Exits non-zero only if go test itself fails — coverage % is informational.

set -euo pipefail

mkdir -p .health
profile=.health/coverage.out

echo "Running go test -coverprofile=$profile ./..."
# CI uses -race -covermode=atomic, which requires cgo. Skip the race
# detector locally if cgo is unavailable (Windows without a C toolchain)
# so the script still produces a coverage number. Race issues are caught
# in CI anyway.
covermode=atomic
race=-race
if ! command -v gcc >/dev/null 2>&1 && [ "${CGO_ENABLED:-1}" != "1" ]; then
  race=""
  covermode=set
fi
go test $race -coverprofile="$profile" -covermode=$covermode ./... >.health/test.log 2>&1 || true

if [ ! -s "$profile" ]; then
  echo "No coverage profile produced; check .health/test.log"
  exit 1
fi

# Raw coverage — what `go test -cover ./...` would print.
raw_pct=$(go tool cover -func="$profile" | tail -1 | awk '{print $3}')

# Scored coverage — same exclude regex as the CI gate. Keep in sync with
# ditdotdev/.github/.github/workflows/repo-health.yml: any change to the
# exclude regex there must be mirrored here.
exclude_re='(\.pb\.go|_gen\.go|_generated\.go|mock_[^/]+\.go|/cmd/[^/]+/main\.go):'

filtered=.health/coverage-filtered.out
{
  head -1 "$profile"
  tail -n +2 "$profile" | grep -vE "$exclude_re" || true
} >"$filtered"

if [ "$(wc -l <"$filtered")" -gt 1 ]; then
  scored_pct=$(go tool cover -func="$filtered" | tail -1 | awk '{print $3}')
else
  scored_pct="$raw_pct"
fi

echo
echo "Coverage:"
echo "  raw    (all .go files):                       $raw_pct"
echo "  scored (excludes generated + cmd/*/main.go):  $scored_pct  <-- this is what the CI gate sees"
