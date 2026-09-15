#!/usr/bin/env bash
# Mirrors the release workflow's AppImage smoke tests locally through a container
# runtime, so iterating does not spend GitHub Actions minutes.
#
# Note: the AppImage bundles the audio codecs from the build host, so one built on
# a newer distro can legitimately fail on older ones. CI builds on Ubuntu 24.04
# (glibc 2.39), which is the baseline these checks assume.
#
# Usage: ./test-appimage.sh [path/to/AppImage]
#        CONTAINER_RUNTIME=docker ./test-appimage.sh
set -uo pipefail

runtime="${CONTAINER_RUNTIME:-podman}"
command -v "$runtime" >/dev/null || { echo "container runtime '$runtime' not found" >&2; exit 1; }

if [ "$#" -ge 1 ]; then
  app="$1"
else
  app="$(ls bin/*.AppImage 2>/dev/null | head -1 || true)"
fi
[ -n "$app" ] && [ -f "$app" ] || { echo "no AppImage found; build it first or pass a path" >&2; exit 1; }
app="$(realpath "$app")"
echo "testing: $app"

failed=0
check() {
  local name="$1"
  shift
  if "$@"; then echo "PASS  $name"; else echo "FAIL  $name"; failed=1; fi
}

static_runtime() { file "$app" | grep -q "static-pie linked"; }
check "static-pie runtime (no libfuse2)" static_runtime

for image in debian:12 ubuntu:24.04 fedora:41; do
  distro() {
    "$runtime" run --rm -v "$app:/app.AppImage:ro" "$image" bash -lc '
      set -e
      if command -v apt-get >/dev/null 2>&1; then
        apt-get update -qq
        DEBIAN_FRONTEND=noninteractive apt-get install -y -qq libgtk-3-0 libwebkit2gtk-4.1-0 >/dev/null
      else
        dnf install -y -q gtk3 webkit2gtk4.1 >/dev/null
      fi
      out=$(timeout 20 /app.AppImage --appimage-extract-and-run 2>&1 || true)
      echo "$out" | tail -3
      echo "$out" | grep -q "cannot open display"
    '
  }
  check "$image" distro
done

missing_message() {
  local out
  out=$("$runtime" run --rm -v "$app:/app.AppImage:ro" ubuntu:24.04 bash -lc \
    'timeout 20 /app.AppImage --appimage-extract-and-run 2>&1 || true')
  echo "$out" | tail -3
  echo "$out" | grep -q "missing system libraries"
}
check "negative: missing WebKitGTK message" missing_message

if [ "$failed" -eq 0 ]; then
  echo "all checks passed"
else
  echo "some checks failed (see above)"
  exit 1
fi
