#!/usr/bin/env bash
# Builds and smoke-tests the Flatpak locally without GitHub Actions minutes.
#
# It reuses the binary from the local AppImage, exactly like the release
# workflow does.
#
# Usage: ./build-local.sh [path/to/AppImage]
set -euo pipefail

command -v flatpak >/dev/null || { echo "flatpak is required" >&2; exit 1; }
if ! command -v flatpak-builder >/dev/null; then
  cat >&2 <<'MSG'
flatpak-builder is not installed. Install it with your package manager, e.g.
  Arch:   sudo pacman -S flatpak-builder
  Ubuntu: sudo apt install flatpak-builder
MSG
  exit 1
fi

repo_root="$(realpath "$(dirname "$0")/../../..")"
cd "$repo_root"

if [ "$#" -ge 1 ]; then
  app="$1"
else
  app="$(ls bin/*.AppImage 2>/dev/null | head -1 || true)"
  [ -n "$app" ] || { echo "no AppImage found in bin/; build it first or pass a path" >&2; exit 1; }
fi
app="$(realpath "$app")"

echo "== extracting binary from $app =="
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
mkdir -p bin
(
  cd "$work"
  "$app" --appimage-extract >/dev/null
  cp squashfs-root/usr/bin/jota "$repo_root/bin/jota"
)

echo "== building the flatpak =="
flatpak-builder --user --disable-rofiles-fuse --force-clean \
  --install-deps-from=flathub --repo=flatpak-repo flatpak-build \
  build/linux/flatpak/io.github.jota_music.jota.yml
flatpak build-bundle flatpak-repo jota.flatpak io.github.jota_music.jota

echo "== smoke-testing the flatpak =="
flatpak install --user -y ./jota.flatpak
out=$(timeout 30 dbus-run-session -- flatpak run io.github.jota_music.jota 2>&1 || true)
echo "$out"
echo "$out" | grep -q "cannot open display"

echo "flatpak ok: $repo_root/jota.flatpak"
