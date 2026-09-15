#!/usr/bin/env bash
# Builds a distro-agnostic AppImage.
#
# The binary is linked against the build machine's audio decoder sonames
# (libFLAC, libmpg123, libogg, libvorbis), so those are bundled. GTK and
# WebKit are intentionally NOT bundled: WebKitGTK resolves its helper
# processes (WebKitNetworkProcess, ...) via absolute paths baked in at build
# time, and mixing a bundled GTK/gdk-pixbuf with the system's WebKit breaks
# codecs. Letting the system provide GTK + WebKit keeps the AppImage portable
# across distributions (the host needs gtk3 and webkit2gtk-4.1 installed).
set -euo pipefail

APP_NAME="${APP_NAME:?APP_NAME is required}"
APP_BINARY="${APP_BINARY:?APP_BINARY is required}"
ICON="${ICON:?ICON is required}"
DESKTOP_FILE="${DESKTOP_FILE:?DESKTOP_FILE is required}"
OUTPUT_DIR="${OUTPUT_DIR:-.}"
BUNDLED_PREFIX="${BUNDLED_PREFIX:-libFLAC so:libmpg123 libogg libvorbis}"

case "$(uname -m)" in
  x86_64 | amd64) ARCH=x86_64 ;;
  aarch64 | arm64) ARCH=aarch64 ;;
  *)
    echo "unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

appdir="$work/${APP_NAME}.AppDir"
mkdir -p "$appdir/usr/bin" "$appdir/usr/lib"

cp "$APP_BINARY" "$appdir/usr/bin/${APP_NAME}"
chmod +x "$appdir/usr/bin/${APP_NAME}"
cp "$ICON" "$appdir/${APP_NAME}.png"
ln -sf "${APP_NAME}.png" "$appdir/.DirIcon"
cp "$DESKTOP_FILE" "$appdir/"

curl -fsSL -o "$appdir/AppRun" \
  "https://github.com/AppImage/AppImageKit/releases/download/continuous/AppRun-${ARCH}"
chmod +x "$appdir/AppRun"

# Bundle the audio decoder libraries the binary links; they are not part of
# the desktop platform and their sonames differ across distributions.
pattern='(libFLAC|libmpg123|libogg|libvorbis|libvorbisenc|libopus)[^ /]*\.so[^ /]*$'
while read -r lib; do
  [ -e "$lib" ] || continue
  cp -L "$lib" "$appdir/usr/lib/"
done < <(ldd "$APP_BINARY" | grep -oE "/[^ ]+\.so[^ ]*" | grep -E "$pattern" | sort -u)

curl -fsSL -o "$work/appimagetool" \
  "https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-${ARCH}.AppImage"
chmod +x "$work/appimagetool"

out="${OUTPUT_DIR%/}/${APP_NAME}-${ARCH}.AppImage"
ARCH="$ARCH" "$work/appimagetool" --appimage-extract-and-run "$appdir" "$out"
echo "AppImage created: $out"
