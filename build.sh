#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Version from git tag or default to dev
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
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

# Update versioninfo.json with current version
update_versioninfo() {
    local versioninfo="cmd/yggstack-gui/versioninfo.json"
    if [ -f "$versioninfo" ]; then
        echo -e "${YELLOW}Updating version info...${NC}"

        # Create updated versioninfo.json
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
        echo -e "${GREEN}✓ Version info updated (${VERSION})${NC}"
    fi
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

# Update versioninfo.json with current version
update_versioninfo
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
go build -trimpath -ldflags "${LDFLAGS}" -o "${BIN_DIR}/${OUTPUT_NAME}" ./cmd/yggstack-gui
echo -e "${GREEN}✓ Built ${OUTPUT_NAME}${NC}"

# Create archive
echo ""
echo -e "${YELLOW}Creating distribution archive...${NC}"
ARCHIVE_BASE="yggstack-gui-${VERSION}-${GOOS}-${GOARCH}"

if [ "$GOOS" = "windows" ]; then
    if command -v zip &> /dev/null; then
        (cd "${BIN_DIR}" && zip -q "../${DIST_DIR}/${ARCHIVE_BASE}.zip" "${OUTPUT_NAME}")
        echo -e "${GREEN}✓ Created ${ARCHIVE_BASE}.zip${NC}"
    fi
else
    tar -czf "${DIST_DIR}/${ARCHIVE_BASE}.tar.gz" -C "${BIN_DIR}" "${OUTPUT_NAME}"
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
