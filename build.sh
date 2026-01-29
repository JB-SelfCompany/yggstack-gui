#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Version from internal/version/version.go (single source of truth)
VERSION_FILE="internal/version/version.go"
if [ -f "$VERSION_FILE" ]; then
    VERSION=$(grep -oP 'var Version = "\K[^"]+' "$VERSION_FILE" 2>/dev/null || echo "dev")
else
    VERSION="dev"
fi
BIN_DIR="bin"
FRONTEND_DIR="frontend"
DIST_DIR="dist"

# Parse semantic version (v1.2.3 -> MAJOR=1, MINOR=2, PATCH=3)
parse_version() {
    if [[ "$VERSION" =~ ^v?([0-9]+)\.([0-9]+)\.([0-9]+) ]]; then
        VERSION_MAJOR="${BASH_REMATCH[1]}"
        VERSION_MINOR="${BASH_REMATCH[2]}"
        VERSION_PATCH="${BASH_REMATCH[3]}"
    else
        VERSION_MAJOR="0"
        VERSION_MINOR="0"
        VERSION_PATCH="0"
    fi
}

# Generate versioninfo.json with current version (for Windows resource embedding)
generate_versioninfo() {
    local versioninfo="cmd/yggstack-gui/versioninfo.json"
    echo -e "${YELLOW}Generating version info...${NC}"

    # Always create/overwrite versioninfo.json with current version
    cat > "$versioninfo" << EOF
{
  "FixedFileInfo": {
    "FileVersion": {"Major": ${VERSION_MAJOR}, "Minor": ${VERSION_MINOR}, "Patch": ${VERSION_PATCH}, "Build": 0},
    "ProductVersion": {"Major": ${VERSION_MAJOR}, "Minor": ${VERSION_MINOR}, "Patch": ${VERSION_PATCH}, "Build": 0},
    "FileFlagsMask": "3f",
    "FileFlags": "00",
    "FileOS": "040004",
    "FileType": "01",
    "FileSubType": "00"
  },
  "StringFileInfo": {
    "FileDescription": "Yggdrasil Network Manager",
    "FileVersion": "${VERSION}",
    "ProductName": "Yggstack-GUI",
    "ProductVersion": "${VERSION}",
    "CompanyName": "JB-SelfCompany",
    "LegalCopyright": "© 2025 JB-SelfCompany"
  },
  "VarFileInfo": {
    "Translation": {"LangID": "0409", "CharsetID": "04B0"}
  }
}
EOF
    echo -e "${GREEN}✓ Version info generated (${VERSION})${NC}"
}

# Detect current platform
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    case "$os" in
        linux*)   GOOS="linux" ;;
        darwin*)  GOOS="darwin" ;;
        mingw*|msys*|cygwin*) GOOS="windows" ;;
        *)        GOOS="$os" ;;
    esac

    case "$arch" in
        x86_64|amd64) GOARCH="amd64" ;;
        aarch64|arm64) GOARCH="arm64" ;;
        *)            GOARCH="$arch" ;;
    esac
}

detect_platform
parse_version

echo -e "${GREEN}Yggstack-GUI Build Script${NC}"
echo -e "${GREEN}Version: ${VERSION}${NC}"
echo -e "${GREEN}Platform: ${GOOS}/${GOARCH}${NC}"
echo ""

# Check dependencies
echo -e "${YELLOW}Checking dependencies...${NC}"
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    exit 1
fi

if ! command -v npm &> /dev/null; then
    echo -e "${RED}Error: npm is not installed${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Dependencies OK${NC}"
echo ""

# Download Go dependencies
echo -e "${YELLOW}Downloading Go dependencies...${NC}"
go mod download
go mod verify
echo -e "${GREEN}✓ Go dependencies downloaded${NC}"
echo ""

# Build frontend
echo -e "${YELLOW}Building frontend...${NC}"
cd "${FRONTEND_DIR}"
npm install
npm run build
cd ..
echo -e "${GREEN}✓ Frontend built${NC}"
echo ""

# Copy frontend dist to internal/web for embedding
echo -e "${YELLOW}Copying frontend for embedding...${NC}"
rm -rf internal/web/dist
mkdir -p internal/web/dist
cp -r frontend/dist/* internal/web/dist/
echo -e "${GREEN}✓ Frontend copied to internal/web/dist${NC}"
echo ""

# Prepare app icons for embedding
echo -e "${YELLOW}Preparing app icons...${NC}"
mkdir -p cmd/yggstack-gui/resources

if [ -f "assets/build/appicon.png" ]; then
    cp assets/build/appicon.png cmd/yggstack-gui/resources/appicon.png
    echo -e "${GREEN}✓ PNG icon copied${NC}"
fi

if [ -f "assets/build/windows/icon.ico" ]; then
    cp assets/build/windows/icon.ico cmd/yggstack-gui/resources/appicon.ico
    echo -e "${GREEN}✓ ICO icon copied${NC}"
fi
echo ""

# Generate versioninfo.json with current version
generate_versioninfo
echo ""

# Generate Windows resources (icon embedded in exe)
if [ "$GOOS" = "windows" ]; then
    echo -e "${YELLOW}Generating Windows resources...${NC}"
    go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
    cd cmd/yggstack-gui
    goversioninfo -64 -icon=../../assets/build/windows/icon.ico -o resource_windows.syso
    cd ../..
    echo -e "${GREEN}✓ Windows resources generated${NC}"
    echo ""
fi

# Create output directories
mkdir -p "${BIN_DIR}"
mkdir -p "${DIST_DIR}"

# Determine output filename
VERSION_PKG="github.com/JB-SelfCompany/yggstack-gui/internal/version"
if [ "$GOOS" = "windows" ]; then
    OUTPUT_NAME="yggstack-gui-${GOOS}-${GOARCH}.exe"
    LDFLAGS="-X ${VERSION_PKG}.Version=${VERSION} -s -w -H windowsgui"
else
    OUTPUT_NAME="yggstack-gui-${GOOS}-${GOARCH}"
    LDFLAGS="-X ${VERSION_PKG}.Version=${VERSION} -s -w"
fi

# Build
echo -e "${YELLOW}Building ${GOOS}/${GOARCH}...${NC}"
go build -tags prod -trimpath -ldflags "${LDFLAGS}" -o "${BIN_DIR}/${OUTPUT_NAME}" ./cmd/yggstack-gui
echo -e "${GREEN}✓ Built ${OUTPUT_NAME}${NC}"

# Copy CEF framework files to bin directory
echo ""
echo -e "${YELLOW}Copying CEF framework files...${NC}"
# Find CEF directory (local first, then ~/.energy)
# Note: Linux uses CEF 109 (liblcl-109 available), Windows uses CEF 136
if [ "$GOOS" = "windows" ]; then
    CEF_DIR="energy/CEF-136_WINDOWS_64"
    [ ! -d "$CEF_DIR" ] && CEF_DIR="$HOME/.energy/cef/CEF-136_WINDOWS_64"
elif [ "$GOOS" = "linux" ]; then
    # Try CEF 109 first (has liblcl support), then 136, then ~/.energy
    for dir in "energy/CEF-109_LINUX_64" "energy/CEF-136_LINUX_64" "$HOME/.energy/cef/CEF-109_LINUX_64" "$HOME/.energy/CEF109_LINUX64"; do
        if [ -d "$dir" ]; then
            CEF_DIR="$dir"
            break
        fi
    done
    [ -z "$CEF_DIR" ] && CEF_DIR="energy/CEF-109_LINUX_64"
elif [ "$GOOS" = "darwin" ]; then
    CEF_DIR="energy/CEF-109_MACOSX_64"
    [ ! -d "$CEF_DIR" ] && CEF_DIR="$HOME/.energy/cef/CEF-109_MACOSX_64"
fi

if [ -d "$CEF_DIR" ]; then
    # Copy all CEF files (dll/so/dylib, pak, dat, bin, json) and locales directory
    cp -r "${CEF_DIR}"/*.dll "${BIN_DIR}/" 2>/dev/null || true
    cp -r "${CEF_DIR}"/*.so "${BIN_DIR}/" 2>/dev/null || true
    cp -r "${CEF_DIR}"/*.so.* "${BIN_DIR}/" 2>/dev/null || true
    cp -r "${CEF_DIR}"/*.dylib "${BIN_DIR}/" 2>/dev/null || true
    cp -r "${CEF_DIR}"/*.pak "${BIN_DIR}/" 2>/dev/null || true
    cp -r "${CEF_DIR}"/*.dat "${BIN_DIR}/" 2>/dev/null || true
    cp -r "${CEF_DIR}"/*.bin "${BIN_DIR}/" 2>/dev/null || true
    cp -r "${CEF_DIR}"/*.json "${BIN_DIR}/" 2>/dev/null || true
    cp -r "${CEF_DIR}"/locales "${BIN_DIR}/" 2>/dev/null || true
    # Linux: copy swiftshader and additional libs
    cp -r "${CEF_DIR}"/swiftshader "${BIN_DIR}/" 2>/dev/null || true
    cp -r "${CEF_DIR}"/libEGL.so "${BIN_DIR}/" 2>/dev/null || true
    cp -r "${CEF_DIR}"/libGLESv2.so "${BIN_DIR}/" 2>/dev/null || true
    # Copy liblcl from ~/.energy if not in CEF_DIR
    if [ "$GOOS" = "linux" ] && [ ! -f "${BIN_DIR}/liblcl.so" ]; then
        cp "$HOME/.energy/liblcl.so" "${BIN_DIR}/" 2>/dev/null || true
    elif [ "$GOOS" = "darwin" ] && [ ! -f "${BIN_DIR}/liblcl.dylib" ]; then
        cp "$HOME/.energy/liblcl.dylib" "${BIN_DIR}/" 2>/dev/null || true
    fi
    echo -e "${GREEN}✓ CEF framework copied to ${BIN_DIR}/${NC}"
else
    echo -e "${RED}Error: CEF framework not found${NC}"
    echo -e "${RED}Searched: ${CEF_DIR} and ~/.energy/cef/${NC}"
    echo -e "${YELLOW}Install CEF using:${NC}"
    echo -e "${YELLOW}  go install github.com/energye/energy/v2/cmd/energy@latest${NC}"
    echo -e "${YELLOW}  energy install${NC}"
    exit 1
fi

# Compress binaries with UPX (if available)
echo ""
echo -e "${YELLOW}Compressing binaries with UPX...${NC}"
if [ -f "energy/upx/upx.exe" ] && [ "$GOOS" = "windows" ]; then
    # Windows: use bundled UPX
    energy/upx/upx.exe -9 --best --lzma "${BIN_DIR}/${OUTPUT_NAME}" 2>/dev/null || true
    energy/upx/upx.exe -9 --best --lzma "${BIN_DIR}/liblcl.dll" 2>/dev/null || true
    echo -e "${GREEN}✓ Binaries compressed${NC}"
elif command -v upx &> /dev/null && [ "$GOOS" = "linux" ]; then
    # Linux: use system UPX
    upx -9 --best --lzma "${BIN_DIR}/${OUTPUT_NAME}" 2>/dev/null || true
    upx -9 --best --lzma "${BIN_DIR}/liblcl.so" 2>/dev/null || true
    echo -e "${GREEN}✓ Binaries compressed${NC}"
else
    echo -e "${YELLOW}⚠ UPX not found, skipping compression${NC}"
fi

# Clean up runtime data before archiving
echo ""
echo -e "${YELLOW}Cleaning runtime data...${NC}"
rm -rf "${BIN_DIR}/data" 2>/dev/null || true
rm -rf "${BIN_DIR}/cache" 2>/dev/null || true
rm -rf "${BIN_DIR}/GPUCache" 2>/dev/null || true
rm -rf "${BIN_DIR}/blob_storage" 2>/dev/null || true
echo -e "${GREEN}✓ Runtime data cleaned${NC}"

# Create archive
echo ""
echo -e "${YELLOW}Creating distribution archive...${NC}"
ARCHIVE_BASE="yggstack-gui-${VERSION}-${GOOS}-${GOARCH}"

if [ "$GOOS" = "windows" ]; then
    # Use 7z format for better compression
    if [ -f "energy/7z/7za.exe" ]; then
        (cd "${BIN_DIR}" && ../energy/7z/7za.exe a -t7z -mx=5 -mmt=on -xr!data -xr!cache -xr!GPUCache -xr!blob_storage "../${DIST_DIR}/${ARCHIVE_BASE}.7z" . > /dev/null)
        echo -e "${GREEN}✓ Created ${ARCHIVE_BASE}.7z${NC}"
    elif command -v zip &> /dev/null; then
        (cd "${BIN_DIR}" && zip -rq "../${DIST_DIR}/${ARCHIVE_BASE}.zip" .)
        echo -e "${GREEN}✓ Created ${ARCHIVE_BASE}.zip${NC}"
    else
        # PowerShell fallback
        powershell -Command "Compress-Archive -Path '${BIN_DIR}/*' -DestinationPath '${DIST_DIR}/${ARCHIVE_BASE}.zip' -Force"
        echo -e "${GREEN}✓ Created ${ARCHIVE_BASE}.zip (PowerShell)${NC}"
    fi
else
    tar -czf "${DIST_DIR}/${ARCHIVE_BASE}.tar.gz" -C "${BIN_DIR}" .
    echo -e "${GREEN}✓ Created ${ARCHIVE_BASE}.tar.gz${NC}"
fi

echo ""
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo -e "${GREEN}Build completed successfully!${NC}"
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo ""
echo -e "Binary:  ${YELLOW}${BIN_DIR}/${OUTPUT_NAME}${NC}"
echo -e "Archive: ${YELLOW}${DIST_DIR}/${ARCHIVE_BASE}.*${NC}"
echo ""
