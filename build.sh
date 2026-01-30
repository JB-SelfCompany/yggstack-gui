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

# Detect current platform (Linux/Windows only)
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    case "$os" in
        linux*)   GOOS="linux" ;;
        mingw*|msys*|cygwin*) GOOS="windows" ;;
        *)
            echo -e "${RED}Error: Unsupported platform: $os${NC}"
            echo -e "${YELLOW}Only Linux and Windows are supported${NC}"
            exit 1
            ;;
    esac

    case "$arch" in
        x86_64|amd64) GOARCH="amd64" ;;
        *)
            echo -e "${RED}Error: Unsupported architecture: $arch${NC}"
            echo -e "${YELLOW}Only amd64 (x86_64) is supported${NC}"
            exit 1
            ;;
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

# Clean and create output directories
echo -e "${YELLOW}Cleaning output directories...${NC}"
rm -rf "${BIN_DIR}"
rm -rf "${DIST_DIR}"
mkdir -p "${BIN_DIR}"
mkdir -p "${DIST_DIR}"
echo -e "${GREEN}✓ Output directories cleaned${NC}"
echo ""

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
else
    # Linux: Try CEF 109 first (has liblcl support), then 136, then ~/.energy
    for dir in "energy/CEF-109_LINUX_64" "energy/CEF-136_LINUX_64" "$HOME/.energy/cef/CEF-109_LINUX_64" "$HOME/.energy/CEF109_LINUX64"; do
        if [ -d "$dir" ]; then
            CEF_DIR="$dir"
            break
        fi
    done
    [ -z "$CEF_DIR" ] && CEF_DIR="energy/CEF-109_LINUX_64"
fi

if [ -d "$CEF_DIR" ]; then
    if [ "$GOOS" = "linux" ]; then
        # Linux: CEF 109 files are in Release/ and Resources/ subdirectories
        # Copy libraries from Release/
        if [ -d "${CEF_DIR}/Release" ]; then
            cp "${CEF_DIR}/Release"/libcef.so "${BIN_DIR}/" 2>/dev/null || true
            cp "${CEF_DIR}/Release"/libEGL.so "${BIN_DIR}/" 2>/dev/null || true
            cp "${CEF_DIR}/Release"/libGLESv2.so "${BIN_DIR}/" 2>/dev/null || true
            cp "${CEF_DIR}/Release"/libvk_swiftshader.so "${BIN_DIR}/" 2>/dev/null || true
            cp "${CEF_DIR}/Release"/libvulkan.so.1 "${BIN_DIR}/" 2>/dev/null || true
            cp "${CEF_DIR}/Release"/snapshot_blob.bin "${BIN_DIR}/" 2>/dev/null || true
            cp "${CEF_DIR}/Release"/v8_context_snapshot.bin "${BIN_DIR}/" 2>/dev/null || true
            cp "${CEF_DIR}/Release"/vk_swiftshader_icd.json "${BIN_DIR}/" 2>/dev/null || true
        fi
        # Copy resources from Resources/
        if [ -d "${CEF_DIR}/Resources" ]; then
            cp "${CEF_DIR}/Resources"/icudtl.dat "${BIN_DIR}/" 2>/dev/null || true
            cp "${CEF_DIR}/Resources"/resources.pak "${BIN_DIR}/" 2>/dev/null || true
            cp "${CEF_DIR}/Resources"/chrome_100_percent.pak "${BIN_DIR}/" 2>/dev/null || true
            cp "${CEF_DIR}/Resources"/chrome_200_percent.pak "${BIN_DIR}/" 2>/dev/null || true
            # CEF 109 requires all locale files - copy real ones and create stubs
            mkdir -p "${BIN_DIR}/locales"
            cp "${CEF_DIR}/Resources/locales/en-US.pak" "${BIN_DIR}/locales/" 2>/dev/null || true
            cp "${CEF_DIR}/Resources/locales/ru.pak" "${BIN_DIR}/locales/" 2>/dev/null || true
            # Create stub locales (copy en-US.pak as placeholder for other locales)
            for locale in am ar bg bn ca cs da de el en-GB es es-419 et fa fi fil fr gu he hi hr hu id it ja kn ko lt lv ml mr ms nb nl pl pt-BR pt-PT ro sk sl sr sv sw ta te th tr uk vi zh-CN zh-TW; do
                cp "${BIN_DIR}/locales/en-US.pak" "${BIN_DIR}/locales/${locale}.pak" 2>/dev/null || true
            done
        fi
        # Copy liblcl.so from CEF_DIR root or ~/.energy
        if [ -f "${CEF_DIR}/liblcl.so" ]; then
            cp "${CEF_DIR}/liblcl.so" "${BIN_DIR}/" 2>/dev/null || true
        elif [ -f "$HOME/.energy/liblcl.so" ]; then
            cp "$HOME/.energy/liblcl.so" "${BIN_DIR}/" 2>/dev/null || true
        fi
    else
        # Windows: all files in root directory
        cp -r "${CEF_DIR}"/*.dll "${BIN_DIR}/" 2>/dev/null || true
        cp -r "${CEF_DIR}"/*.pak "${BIN_DIR}/" 2>/dev/null || true
        cp -r "${CEF_DIR}"/*.dat "${BIN_DIR}/" 2>/dev/null || true
        cp -r "${CEF_DIR}"/*.bin "${BIN_DIR}/" 2>/dev/null || true
        cp -r "${CEF_DIR}"/*.json "${BIN_DIR}/" 2>/dev/null || true
        # Copy only en and ru locales for Windows
        mkdir -p "${BIN_DIR}/locales"
        cp "${CEF_DIR}/locales/en-US.pak" "${BIN_DIR}/locales/" 2>/dev/null || true
        cp "${CEF_DIR}/locales/ru.pak" "${BIN_DIR}/locales/" 2>/dev/null || true
    fi
    # Verify critical files were copied
    if [ "$GOOS" = "linux" ]; then
        CRITICAL_FILES="libcef.so libEGL.so libGLESv2.so liblcl.so resources.pak icudtl.dat"
    else
        CRITICAL_FILES="libcef.dll liblcl.dll resources.pak icudtl.dat"
    fi

    MISSING=""
    for f in $CRITICAL_FILES; do
        if [ ! -f "${BIN_DIR}/$f" ]; then
            MISSING="$MISSING $f"
        fi
    done

    if [ -n "$MISSING" ]; then
        echo -e "${RED}Warning: Missing critical CEF files:${MISSING}${NC}"
        echo -e "${YELLOW}CEF_DIR was: ${CEF_DIR}${NC}"
        ls -la "${CEF_DIR}/" 2>/dev/null | head -20
    else
        echo -e "${GREEN}✓ CEF framework copied to ${BIN_DIR}/${NC}"
        LIB_COUNT=$(ls ${BIN_DIR}/*.so ${BIN_DIR}/*.dll 2>/dev/null | wc -l)
        PAK_COUNT=$(ls ${BIN_DIR}/*.pak 2>/dev/null | wc -l)
        LOCALE_COUNT=$(ls ${BIN_DIR}/locales/*.pak 2>/dev/null | wc -l)
        echo -e "${GREEN}  Libraries: ${LIB_COUNT}, Resources: ${PAK_COUNT}, Locales: ${LOCALE_COUNT}${NC}"
    fi
else
    echo -e "${RED}Error: CEF framework not found${NC}"
    echo -e "${RED}Searched: ${CEF_DIR} and ~/.energy/cef/${NC}"
    echo -e "${YELLOW}Install CEF using:${NC}"
    echo -e "${YELLOW}  go install github.com/energye/energy/v2/cmd/energy@latest${NC}"
    echo -e "${YELLOW}  energy install${NC}"
    exit 1
fi

# Strip debug symbols from binaries (Linux) - MAXIMUM stripping
if [ "$GOOS" = "linux" ]; then
    echo ""
    echo -e "${YELLOW}Stripping debug symbols (maximum)...${NC}"
    if command -v strip &> /dev/null; then
        # Strip all .so files with maximum stripping (-s removes all symbols)
        for lib in "${BIN_DIR}"/*.so "${BIN_DIR}"/*.so.*; do
            [ -f "$lib" ] && strip -s "$lib" 2>/dev/null || true
        done
        # Strip the main binary
        strip -s "${BIN_DIR}/${OUTPUT_NAME}" 2>/dev/null || true
        echo -e "${GREEN}✓ Debug symbols stripped${NC}"
        BIN_SIZE_AFTER=$(du -sh "${BIN_DIR}" | cut -f1)
        echo -e "${GREEN}  bin/ size after strip: ${BIN_SIZE_AFTER}${NC}"
    else
        echo -e "${YELLOW}⚠ strip not found, skipping${NC}"
    fi
fi

# Compress binaries with UPX (if available)
echo ""
echo -e "${YELLOW}Compressing binaries with UPX...${NC}"
if [ -f "energy/upx/upx.exe" ] && [ "$GOOS" = "windows" ]; then
    # Windows: use bundled UPX
    energy/upx/upx.exe -9 --best --lzma "${BIN_DIR}/${OUTPUT_NAME}" 2>/dev/null || true
    energy/upx/upx.exe -9 --best --lzma "${BIN_DIR}/liblcl.dll" 2>/dev/null || true
    echo -e "${GREEN}✓ Binaries compressed${NC}"
elif [ "$GOOS" = "linux" ]; then
    # Linux: download UPX if not available
    UPX_BIN=""
    if command -v upx &> /dev/null; then
        UPX_BIN="upx"
    elif [ -f "energy/upx/upx" ]; then
        UPX_BIN="energy/upx/upx"
    else
        # Download UPX for Linux
        echo -e "${YELLOW}Downloading UPX for Linux...${NC}"
        mkdir -p energy/upx
        UPX_VERSION="4.2.4"
        curl -sL "https://github.com/upx/upx/releases/download/v${UPX_VERSION}/upx-${UPX_VERSION}-amd64_linux.tar.xz" | tar -xJf - -C energy/upx --strip-components=1 2>/dev/null
        if [ -f "energy/upx/upx" ]; then
            chmod +x energy/upx/upx
            UPX_BIN="energy/upx/upx"
            echo -e "${GREEN}✓ UPX downloaded${NC}"
        fi
    fi

    if [ -n "$UPX_BIN" ]; then
        # Compress ALL binaries for minimum unpacked size
        echo -e "${YELLOW}Compressing all binaries (this may take a while)...${NC}"

        # Compress main binary
        echo -n "  ${OUTPUT_NAME}... "
        if $UPX_BIN -9 --best --lzma "${BIN_DIR}/${OUTPUT_NAME}" 2>/dev/null; then
            echo -e "${GREEN}OK${NC}"
        else
            echo -e "${YELLOW}skipped${NC}"
        fi

        # Compress all .so files
        for lib in "${BIN_DIR}"/*.so; do
            if [ -f "$lib" ]; then
                libname=$(basename "$lib")
                echo -n "  ${libname}... "
                if $UPX_BIN -9 --best --lzma "$lib" 2>/dev/null; then
                    echo -e "${GREEN}OK${NC}"
                else
                    echo -e "${YELLOW}skipped${NC}"
                fi
            fi
        done

        # Show final size
        BIN_SIZE_FINAL=$(du -sh "${BIN_DIR}" | cut -f1)
        echo -e "${GREEN}✓ Compression complete. bin/ size: ${BIN_SIZE_FINAL}${NC}"
    else
        echo -e "${YELLOW}⚠ UPX not available, skipping compression${NC}"
    fi
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
    # Linux: use tar with xz compression for better ratio
    if command -v xz &> /dev/null; then
        tar -cJf "${DIST_DIR}/${ARCHIVE_BASE}.tar.xz" -C "${BIN_DIR}" \
            --exclude='*.tar*' \
            --exclude='*.7z' \
            --exclude='*.zip' \
            --exclude='data' \
            --exclude='cache' \
            --exclude='GPUCache' \
            --exclude='blob_storage' \
            .
        ARCHIVE_SIZE=$(du -h "${DIST_DIR}/${ARCHIVE_BASE}.tar.xz" | cut -f1)
        echo -e "${GREEN}✓ Created ${ARCHIVE_BASE}.tar.xz (${ARCHIVE_SIZE})${NC}"
    else
        # Fallback to gzip
        tar -czf "${DIST_DIR}/${ARCHIVE_BASE}.tar.gz" -C "${BIN_DIR}" \
            --exclude='*.tar*' \
            --exclude='*.7z' \
            --exclude='*.zip' \
            --exclude='data' \
            --exclude='cache' \
            --exclude='GPUCache' \
            --exclude='blob_storage' \
            .
        ARCHIVE_SIZE=$(du -h "${DIST_DIR}/${ARCHIVE_BASE}.tar.gz" | cut -f1)
        echo -e "${GREEN}✓ Created ${ARCHIVE_BASE}.tar.gz (${ARCHIVE_SIZE})${NC}"
    fi
fi

# Show bin directory size
BIN_SIZE=$(du -sh "${BIN_DIR}" | cut -f1)
echo -e "${GREEN}  Total bin/ size: ${BIN_SIZE}${NC}"

echo ""
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo -e "${GREEN}Build completed successfully!${NC}"
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo ""
echo -e "Binary:  ${YELLOW}${BIN_DIR}/${OUTPUT_NAME}${NC}"
echo -e "Archive: ${YELLOW}${DIST_DIR}/${ARCHIVE_BASE}.*${NC}"
echo ""
