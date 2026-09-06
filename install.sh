#!/bin/bash
set -e

BINARY="./build/bin/jota"
ICON="./build/appicon.png"
DESKTOP_FILE="jota.desktop"
INSTALL_DIR="$HOME/.local/bin"
APPS_DIR="$HOME/.local/share/applications"
ICONS_DIR="$HOME/.local/share/icons/hicolor/512x512/apps"

echo "Instalando Jota..."

# Crear directorios
mkdir -p "$INSTALL_DIR" "$APPS_DIR" "$ICONS_DIR"

# Copiar binario
cp "$BINARY" "$INSTALL_DIR/jota"
chmod +x "$INSTALL_DIR/jota"

# Copiar icono
cp "$ICON" "$ICONS_DIR/jota.png"

# Crear .desktop
cat > "$APPS_DIR/$DESKTOP_FILE" << EOF
[Desktop Entry]
Name=Jota
Comment=Self-hosted music streaming desktop app
Exec=$INSTALL_DIR/jota
Icon=jota
Type=Application
Categories=AudioVideo;Audio;Player;
StartupNotify=true
Terminal=false
EOF

# Actualizar base de datos de iconos y aplicaciones
update-desktop-database "$APPS_DIR" 2>/dev/null || true
gtk-update-icon-cache "$HOME/.local/share/icons/hicolor" 2>/dev/null || true

echo "✓ Jota instalado en $INSTALL_DIR/jota"
echo "✓ Acceso directo creado en menú de aplicaciones"
echo ""
echo "Para ejecutar: jota (si ~/.local/bin está en PATH) o $INSTALL_DIR/jota"