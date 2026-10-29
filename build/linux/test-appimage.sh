#!/usr/bin/env bash
# Mirrors the release workflow's AppImage smoke tests locally through a container
# runtime, so iterating does not spend GitHub Actions minutes.
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

# The window has to come up with the network blackholed (every outbound TCP
# packet dropped). Startup used to wait on Spotify's session restore before any
# window existed, so a stalled connection left the process alive, mute and
# blocking each later launch. TestStalledAccesspointHonoursDeadline pins the
# credential path; this is the end-to-end promise: no network, window anyway.
#
# The launch is budgeted: `window ready in <ms>` is what a regression on the
# launch path looks like (opening the store or the sync folder before the window
# once added hundreds of ms), and 5s in a container under xvfb leaves room for a
# slow runner while still failing anything that waits on the network.
blackholed() {
  local out
  out=$("$runtime" run --rm --cap-add NET_ADMIN -v "$app:/app.AppImage:ro" ubuntu:24.04 bash -lc '
    set -e
    apt-get update -qq
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq libgtk-3-0 libwebkit2gtk-4.1-0 xvfb iptables >/dev/null
    mkdir -p /run/user/0 && chmod 700 /run/user/0
    export XDG_RUNTIME_DIR=/run/user/0
    iptables -A OUTPUT -p tcp -j DROP
    timeout 30 xvfb-run -a /app.AppImage --appimage-extract-and-run >/dev/null 2>&1 || true
    grep "window ready" /root/.config/jota/jota.log || echo "no window ready line within 30s"
    ms=$(grep -o "window ready in [0-9]*" /root/.config/jota/jota.log | grep -o "[0-9]*$" || true)
    echo "launch: ${ms:-none}ms"
    [ -n "$ms" ] && [ "$ms" -lt 3000 ]
  ' 2>&1)
  echo "$out" | tail -3
  echo "$out" | grep -q "window ready"
  echo "$out" | grep -q "launch: "
}
check "blackholed network: window opens within budget" blackholed

# Closing the window has to end the process. A blocking call on the GTK thread
# (the clipboard poll was one, gtk_clipboard_wait_for_text, which froze the
# window and once crashed in it) leaves a windowless process holding the store
# lock, and every later launch then runs without persistence. This clicks the
# app's own close button, which is the path a user takes.
closes() {
  local out
  out=$("$runtime" run --rm -v "$app:/app.AppImage:ro" ubuntu:24.04 bash -lc '
    set -e
    apt-get update -qq
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq libgtk-3-0 libwebkit2gtk-4.1-0 xvfb xdotool >/dev/null
    mkdir -p /run/user/0 && chmod 700 /run/user/0
    export XDG_RUNTIME_DIR=/run/user/0
    Xvfb :99 -screen 0 1920x1080x24 >/dev/null 2>&1 &
    export DISPLAY=:99
    sleep 1
    /app.AppImage --appimage-extract-and-run >/dev/null 2>&1 &
    pid=$!
    for i in $(seq 600); do
      grep -q "window ready" /root/.config/jota/jota.log 2>/dev/null && break
      sleep 0.1
    done
    sleep 1
    wid=$(xdotool search --name "^Jota$" | while read -r w; do
      g=$(xdotool getwindowgeometry "$w" | awk "/Geometry:/{print \$2}")
      [ "${g%x*}" -ge 500 ] && echo "$w"
    done | tail -1)
    if [ -z "$wid" ]; then echo "close: no window"; kill -9 $pid; exit 1; fi
    width=$(xdotool getwindowgeometry "$wid" | awk "/Geometry:/{split(\$2,a,\"x\");print a[1]}")
    xdotool mousemove --window "$wid" $((width - 20)) 20 click 1
    for i in $(seq 200); do kill -0 $pid 2>/dev/null || break; sleep 0.05; done
    if kill -0 $pid 2>/dev/null; then
      echo "close: still alive after 10s"
      kill -9 $pid
      exit 1
    fi
    echo "close: exited after $((i * 50))ms"
    grep -q "jota shutting down" /root/.config/jota/jota.log || { echo "close: exited without the shutdown path"; exit 1; }
    echo "close: clean"
  ' 2>&1)
  echo "$out" | tail -3
  echo "$out" | grep -q "close: clean"
}
check "closing the window ends the process" closes

if [ "$failed" -eq 0 ]; then
  echo "all checks passed"
else
  echo "some checks failed (see above)"
  exit 1
fi
