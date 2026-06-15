#!/usr/bin/env bats
#
# Favicon validation.
#
# The dit.dev favicon set must exist and meet favicon standards. These tests
# fail if a delivered icon is the wrong format, not square, or the wrong size —
# so swapping in a bad image (e.g. a JPEG, a 1x1, a non-square crop) is caught.
#
# Two layers:
#   * source files  — the committed assets in dit-remote-server/services/web
#                      (deterministic; the "bad format" guard).
#   * served URLs    — dit.dev/<icon> (ENV=PROD) or localhost (ENV=DEV)
#                      (proves the favicon actually exists where browsers fetch it).
#
# Format/dimension detection uses `file`, which is available in CI without
# ImageMagick. ICO entry sizes are read straight from the ICONDIR header.

load '../../test_helper'
load 'env'

# --- helpers ---------------------------------------------------------------

# Echo "WIDTH HEIGHT" for a PNG, or nothing if not a PNG.
png_dims() {
  file -b "$1" | sed -n 's/^PNG image data, \([0-9]\{1,\}\) x \([0-9]\{1,\}\).*/\1 \2/p'
}

# Assert a file is a square PNG of the expected edge length.
assert_png_square() {
  local f="$1" expect="$2" dims w h
  [ -f "$f" ] || { echo "missing PNG: $f"; return 1; }
  dims="$(png_dims "$f")"
  [ -n "$dims" ] || { echo "not a PNG: $(file -b "$f")"; return 1; }
  w="${dims% *}"; h="${dims#* }"
  [ "$w" = "$h" ] || { echo "not square: ${w}x${h} ($f)"; return 1; }
  [ "$w" = "$expect" ] || { echo "expected ${expect}px, got ${w}px ($f)"; return 1; }
  [ "$(wc -c <"$f")" -gt 0 ] || { echo "empty file: $f"; return 1; }
}

# Echo the sorted, unique icon sizes embedded in an .ico (from its ICONDIR).
ico_sizes() {
  local f="$1" b0 b1 count i off w out=""
  read -r b0 b1 < <(od -An -tu1 -j4 -N2 "$f")
  count=$(( b0 + b1 * 256 ))
  for (( i = 0; i < count; i++ )); do
    off=$(( 6 + i * 16 ))
    w="$(od -An -tu1 -j"$off" -N1 "$f" | tr -d ' ')"
    [ "$w" -eq 0 ] && w=256          # 0 encodes 256 in the ICO format
    out="$out $w"
  done
  printf '%s\n' $out | sort -n | uniq | tr '\n' ' '
}

# Locate dit-remote-server/services/web (sibling of the dit repo).
web_dir() {
  local c
  for c in \
    "${BATS_TEST_DIRNAME}/../../../../../dit-remote-server/services/web" \
    "/c/dev/dit/dit-remote-server/services/web"; do
    if [ -d "$c/app" ]; then (cd "$c" && pwd); return 0; fi
  done
  return 1
}

# Fetch a URL to a temp file; echo the HTTP status.
fetch() {
  # curl here may be the native Windows (mingw) build, which mis-resolves an
  # MSYS-style /tmp/... path - it writes to C:\tmp, where the MSYS `file`/`[ -f`
  # checks below can't see the file (the served tests would then fail even
  # though the asset downloaded fine). Hand curl a Windows path when cygpath is
  # present; on Linux CI cygpath is absent and the path passes through unchanged.
  local dest="$2"
  command -v cygpath >/dev/null 2>&1 && dest="$(cygpath -w "$2")"
  curl -s -o "$dest" -w "%{http_code}" --max-time 15 "$1"
}

# ===========================================================================
# Source assets — the "bad format" guard (deterministic, no network)
# ===========================================================================

@test "favicon source: favicon.ico is a multi-resolution ICO with 16/32/48" {
  local web; web="$(web_dir)" || skip "dit-remote-server/services/web not found"
  local ico="$web/app/favicon.ico"
  [ -f "$ico" ] || { echo "missing $ico"; false; }
  run file -b "$ico"
  assert_output --partial "MS Windows icon resource"
  run ico_sizes "$ico"
  assert_output --partial "16"
  assert_output --partial "32"
  assert_output --partial "48"
}

@test "favicon source: app/icon.png is a 512x512 square PNG" {
  local web; web="$(web_dir)" || skip "dit-remote-server/services/web not found"
  run assert_png_square "$web/app/icon.png" 512
  assert_success
}

@test "favicon source: app/apple-icon.png is a 180x180 square PNG" {
  local web; web="$(web_dir)" || skip "dit-remote-server/services/web not found"
  run assert_png_square "$web/app/apple-icon.png" 180
  assert_success
}

@test "favicon source: public/icon-192.png is a 192x192 square PNG" {
  local web; web="$(web_dir)" || skip "dit-remote-server/services/web not found"
  run assert_png_square "$web/public/icon-192.png" 192
  assert_success
}

@test "favicon source: public/icon-512.png is a 512x512 square PNG" {
  local web; web="$(web_dir)" || skip "dit-remote-server/services/web not found"
  run assert_png_square "$web/public/icon-512.png" 512
  assert_success
}

@test "favicon source: manifest references the 192 and 512 PWA icons" {
  local web; web="$(web_dir)" || skip "dit-remote-server/services/web not found"
  local manifest="$web/app/manifest.ts"
  [ -f "$manifest" ] || { echo "missing $manifest"; false; }
  run cat "$manifest"
  assert_output --partial "/icon-192.png"
  assert_output --partial "/icon-512.png"
}

# ===========================================================================
# Served endpoints — proves the favicon exists where browsers fetch it
# ===========================================================================

@test "favicon served: /favicon.ico exists and is a valid ICO" {
  local out="$BATS_TMPDIR/served-favicon.ico"
  run fetch "${WEB_UI}/favicon.ico" "$out"
  assert_output "200"
  run file -b "$out"
  assert_output --partial "MS Windows icon resource"
}

@test "favicon served: /icon-512.png exists and is a 512x512 square PNG" {
  local out="$BATS_TMPDIR/served-icon-512.png"
  run fetch "${WEB_UI}/icon-512.png" "$out"
  assert_output "200"
  run assert_png_square "$out" 512
  assert_success
}

@test "favicon served: /icon-192.png exists and is a 192x192 square PNG" {
  local out="$BATS_TMPDIR/served-icon-192.png"
  run fetch "${WEB_UI}/icon-192.png" "$out"
  assert_output "200"
  run assert_png_square "$out" 192
  assert_success
}

@test "favicon served: /manifest.webmanifest lists the PWA icons" {
  run curl -sf --max-time 15 "${WEB_UI}/manifest.webmanifest"
  assert_success
  assert_output --partial "icon-192.png"
  assert_output --partial "icon-512.png"
}
