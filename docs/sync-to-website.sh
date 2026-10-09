#!/usr/bin/env bash
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Vendor docs/src/ into the ditdotdev.github.io Jekyll site (docs/), which is
# what renders at https://ditdotdev.github.io/docs. Normally run by
# .github/workflows/docs-publish.yml on every release; run it by hand to
# preview or to seed the site.
#
# The copy is byte-for-byte except cli/cmd/*.md: gen-docs emits clean-slug
# links ("[dit clone](dit_clone)") for the dit.dev renderer, but Jekyll's
# relative-links plugin only rewrites links that point at a real ".md" file,
# so those get the ".md" suffix put back — the same transform
# dit-remote-server/scripts/sync-cli-docs.sh applies for its copy.
#
# Files under <site>/docs/ that no longer exist in docs/src/ are deleted, so
# the site never serves a page that was removed upstream.
#
# Usage:
#   bash docs/sync-to-website.sh [--check] [path-to-ditdotdev.github.io]
#     default path: ../../ditdotdev.github.io (or $WEBSITE_REPO)
#     --check: compare only; exit 1 if the site's docs/ differs from docs/src/
#     DOCS_SRC=<dir> publishes that docs/src tree instead of this checkout's
#     (the workflow uses it to publish a release tag's docs with the current
#     script, since older tags predate this script).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"      # <dit-repo>/docs
SRC="${DOCS_SRC:-$SCRIPT_DIR/src}"
WEB="${WEBSITE_REPO:-$SCRIPT_DIR/../../ditdotdev.github.io}"
CHECK=false
for arg in "$@"; do
  case "$arg" in
    --check) CHECK=true ;;
    *) WEB="$arg" ;;
  esac
done
DST="$WEB/docs"

[ -d "$SRC" ] || { echo "source docs not found: $SRC" >&2; exit 1; }
[ -f "$WEB/_config.yml" ] || { echo "not a Jekyll site (no _config.yml): $WEB (pass the ditdotdev.github.io path as arg 1)" >&2; exit 1; }

# Stage the transformed tree in a temp dir so the destination is only ever
# compared against, or replaced by, a complete and consistent copy.
STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

count=0
while IFS= read -r rel; do
  rel="${rel#./}"
  mkdir -p "$STAGE/$(dirname "$rel")"
  case "$rel" in
    cli/cmd/*.md) sed -E 's/\]\((dit[A-Za-z0-9_-]*)\)/](\1.md)/g' "$SRC/$rel" > "$STAGE/$rel" ;;
    *) cp "$SRC/$rel" "$STAGE/$rel" ;;
  esac
  count=$((count + 1))
done < <(cd "$SRC" && find . -type f -print | LC_ALL=C sort)

if $CHECK; then
  if [ ! -d "$DST" ]; then
    echo "website docs/ does not exist: $DST" >&2
    exit 1
  fi
  if ! diff -rq "$STAGE" "$DST"; then
    echo "website docs/ is out of sync with docs/src ($count source files)." >&2
    echo "Run: bash docs/sync-to-website.sh $WEB" >&2
    exit 1
  fi
  echo "website docs/ is in sync with docs/src ($count files)."
  exit 0
fi

mkdir -p "$DST"

# Remove anything the source no longer has.
removed=0
while IFS= read -r rel; do
  rel="${rel#./}"
  if [ ! -f "$STAGE/$rel" ]; then
    rm -f "$DST/$rel"
    removed=$((removed + 1))
  fi
done < <(cd "$DST" && find . -type f -print)
find "$DST" -type d -empty -delete

# Copy the staged tree in.
while IFS= read -r rel; do
  rel="${rel#./}"
  mkdir -p "$DST/$(dirname "$rel")"
  cp "$STAGE/$rel" "$DST/$rel"
done < <(cd "$STAGE" && find . -type f -print)

diff -rq "$STAGE" "$DST" >/dev/null
echo "Synced $count file(s): docs/src -> $DST (cli/cmd links rewritten to .md, $removed stale file(s) removed)."
echo "Next: commit in ditdotdev.github.io (docs/)."
