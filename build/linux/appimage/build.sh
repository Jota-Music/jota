#!/usr/bin/env bash
# Builds a distro-agnostic AppImage.
#
# WebKitGTK is intentionally NOT bundled. Release builds of webkit2gtk-4.1
# locate their helper processes (WebKitWebProcess, WebKitNetworkProcess,
# WebKitGPUProcess) through the absolute compile-time PKGLIBEXECDIR (for
# example /usr/lib/x86_64-linux-gnu/webkit2gtk-4.1). The runtime override
# (WEBKIT_EXEC_PATH) and the "look next to the executable" fallback are compiled
# out unless WebKit is built with ENABLE_DEVELOPER_MODE, which no distribution
# package is. So a bundled WebKit can never find its own helpers inside a
# portable AppImage, and the host must provide webkit2gtk-4.1 (which pulls in
# GTK3). AppRun below checks for it and prints install hints when it is missing.
#
# The AppImage uses a statically linked runtime, so it does not require libfuse2
# on the target system.
set -euo pipefail

APP_NAME="${APP_NAME:?APP_NAME is required}"
APP_BINARY="${APP_BINARY:?APP_BINARY is required}"
ICON="${ICON:?ICON is required}"
DESKTOP_FILE="${DESKTOP_FILE:?DESKTOP_FILE is required}"
OUTPUT_DIR="${OUTPUT_DIR:-.}"
APPSTREAM="${APPSTREAM:-}"

# Pinned AppImage runtime (immutable dated release). Update deliberately.
RUNTIME_TAG="20251108"
RUNTIME_URL="https://github.com/AppImage/type2-runtime/releases/download/${RUNTIME_TAG}/runtime"
# Static appimagetool; runs without FUSE and embeds the runtime passed via
# --runtime-file. Build-time tool only, not shipped in the AppImage.
APPIMAGETOOL_URL="https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool"

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
mkdir -p \
  "$appdir/usr/bin" \
  "$appdir/usr/share/applications" \
  "$appdir/usr/share/icons/hicolor/512x512/apps"

cp "$APP_BINARY" "$appdir/usr/bin/${APP_NAME}"
chmod +x "$appdir/usr/bin/${APP_NAME}"
cp "$ICON" "$appdir/usr/share/icons/hicolor/512x512/apps/${APP_NAME}.png"
ln -sf "usr/share/icons/hicolor/512x512/apps/${APP_NAME}.png" "$appdir/${APP_NAME}.png"
ln -sf "${APP_NAME}.png" "$appdir/.DirIcon"
cp "$DESKTOP_FILE" "$appdir/usr/share/applications/${APP_NAME}.desktop"
ln -sf "usr/share/applications/${APP_NAME}.desktop" "$appdir/${APP_NAME}.desktop"

if [ -n "$APPSTREAM" ] && [ -f "$APPSTREAM" ]; then
  mkdir -p "$appdir/usr/share/metainfo"
  cp "$APPSTREAM" "$appdir/usr/share/metainfo/"
fi

# AppRun: fail early with a readable message when the host is missing the
# platform libraries instead of letting the dynamic loader print a bare
# "cannot open shared object file".
cat > "$appdir/AppRun" <<APPRUN
#!/bin/sh
HERE="\$(dirname "\$(readlink -f "\$0")")"
APP="\$HERE/usr/bin/${APP_NAME}"

if command -v ldd >/dev/null 2>&1; then
  missing="\$(ldd "\$APP" 2>/dev/null | grep 'not found' | sed 's/^[[:space:]]*//; s/[[:space:]].*//' | sort -u)"
  if [ -n "\$missing" ]; then
    printf '%s\n' "${APP_NAME}: missing system libraries:" \$missing >&2
    printf '%s\n' "${APP_NAME}: this AppImage needs WebKitGTK 4.1 and GTK3 on the host. Install with:" >&2
    printf '%s\n' "  Debian/Ubuntu:  sudo apt install libwebkit2gtk-4.1-0" >&2
    printf '%s\n' "  Fedora:         sudo dnf install webkit2gtk4.1" >&2
    printf '%s\n' "  Arch:           sudo pacman -S webkit2gtk-4.1" >&2
    printf '%s\n' "  openSUSE:       sudo zypper install libwebkit2gtk-4_1-0" >&2
    exit 1
  fi
fi

exec "\$APP" "\$@"
APPRUN
chmod +x "$appdir/AppRun"

curl -fsSL -o "$work/runtime" "${RUNTIME_URL}-${ARCH}"
curl -fsSL -o "$work/appimagetool" "${APPIMAGETOOL_URL}-${ARCH}.AppImage"
chmod +x "$work/runtime" "$work/appimagetool"

out="${OUTPUT_DIR%/}/${APP_NAME}-${ARCH}.AppImage"
mkdir -p "${OUTPUT_DIR}"
ARCH="$ARCH" "$work/appimagetool" --runtime-file "$work/runtime" "$appdir" "$out"
echo "AppImage created: $out"
