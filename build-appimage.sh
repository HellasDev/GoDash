#!/bin/bash
set -e

VERSION=${1:-1.1.0}
APP_NAME="GoDash"
APP_DIR="${APP_NAME}.AppDir"

echo "🚀 Building AppImage for ${APP_NAME} v${VERSION}..."

# Clean previous build
rm -rf "${APP_DIR}"
rm -f "${APP_NAME}-${VERSION}-x86_64.AppImage"

# Create AppDir structure
mkdir -p "${APP_DIR}/usr/bin"
mkdir -p "${APP_DIR}/usr/share/applications"
mkdir -p "${APP_DIR}/usr/share/pixmaps"

# Build the Go binary
echo "📦 Building Go binary..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "${APP_DIR}/usr/bin/godash" .

# Copy desktop file and icon
echo "🎨 Adding desktop integration..."
cp godash.desktop "${APP_DIR}/usr/share/applications/"
cp logo.png "${APP_DIR}/usr/share/pixmaps/godash.png"

# Create AppRun script
cat > "${APP_DIR}/AppRun" << 'EOF'
#!/bin/bash
HERE="$(dirname "$(readlink -f "${0}")")"

# Set up environment
export PATH="${HERE}/usr/bin:${PATH}"

# Launch the application with proper terminal handling
if [ -t 0 ]; then
    # Already in a terminal
    exec "${HERE}/usr/bin/godash" "$@"
else
    # Need to launch in a terminal
    if command -v x-terminal-emulator >/dev/null 2>&1; then
        exec x-terminal-emulator -e "${HERE}/usr/bin/godash" "$@"
    elif command -v gnome-terminal >/dev/null 2>&1; then
        exec gnome-terminal -- "${HERE}/usr/bin/godash" "$@"
    elif command -v konsole >/dev/null 2>&1; then
        exec konsole -e "${HERE}/usr/bin/godash" "$@"
    elif command -v xfce4-terminal >/dev/null 2>&1; then
        exec xfce4-terminal -e "${HERE}/usr/bin/godash" "$@"
    elif command -v alacritty >/dev/null 2>&1; then
        exec alacritty -e "${HERE}/usr/bin/godash" "$@"
    elif command -v xterm >/dev/null 2>&1; then
        exec xterm -e "${HERE}/usr/bin/godash" "$@"
    else
        # Fallback - try to run anyway
        exec "${HERE}/usr/bin/godash" "$@"
    fi
fi
EOF

chmod +x "${APP_DIR}/AppRun"

# Copy desktop file to root for AppImage
cp godash.desktop "${APP_DIR}/"
cp logo.png "${APP_DIR}/godash.png"

# Download appimagetool if not present
if [ ! -f "appimagetool-x86_64.AppImage" ]; then
    echo "📥 Downloading appimagetool..."
    wget -q "https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage"
    chmod +x appimagetool-x86_64.AppImage
fi

# Build AppImage
echo "🔨 Creating AppImage..."
ARCH=x86_64 ./appimagetool-x86_64.AppImage "${APP_DIR}" "${APP_NAME}-${VERSION}-x86_64.AppImage"

# Make executable
chmod +x "${APP_NAME}-${VERSION}-x86_64.AppImage"

echo "✅ AppImage created: ${APP_NAME}-${VERSION}-x86_64.AppImage"
echo "📊 Size: $(du -h "${APP_NAME}-${VERSION}-x86_64.AppImage" | cut -f1)"
echo "🧪 Test with: ./${APP_NAME}-${VERSION}-x86_64.AppImage"