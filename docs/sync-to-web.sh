#!/usr/bin/env bash
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Vendor the hand-written docs from docs/src/ into the dit-remote-server web app
# (services/web/content/docs/), which is what renders at https://dit.dev/docs.
# After running, commit the result in BOTH repos (dit and dit-remote-server);
# the next release rebuilds and deploys the web image with the new content.
#
# cli/cmd/ is intentionally skipped: it's generated from the Cobra command tree
# (`make gen-docs`) and link-transformed into content/docs by dit-remote-server's
# scripts/sync-cli-docs.sh — a raw copy here would clobber that transform.
#
# Usage:
#   bash docs/sync-to-web.sh [path-to-dit-remote-server]   # default: ../../dit-remote-server
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"      # <dit-repo>/docs
SRC="$SCRIPT_DIR/src"
REMOTE="${1:-$SCRIPT_DIR/../../dit-remote-server}"
DST="$REMOTE/services/web/content/docs"

[ -d "$SRC" ] || { echo "source docs not found: $SRC" >&2; exit 1; }
[ -d "$DST" ] || { echo "dest not found: $DST (pass the dit-remote-server path as arg 1)" >&2; exit 1; }

count=0
while IFS= read -r rel; do
  rel="${rel#./}"
  mkdir -p "$DST/$(dirname "$rel")"
  cp "$SRC/$rel" "$DST/$rel"
  count=$((count + 1))
done < <(cd "$SRC" && find . -path './cli/cmd' -prune -o -type f -print)

echo "Vendored $count file(s): docs/src/ -> $DST (cli/cmd skipped)."
echo "Next: commit in both repos — dit (docs/src/) and dit-remote-server (services/web/content/docs/)."
