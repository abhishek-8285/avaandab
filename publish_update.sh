#!/usr/bin/env bash
# ==============================================================================
# Remote Web Publisher for Tecno Pova 2 Auto-Updater Agent
# ==============================================================================
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
YELLOW='\033[0;33m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Config / Target Directory (Can be local web directory or public server directory)
DIST_DIR="${DIST_DIR:-$SCRIPT_DIR/dist}"
BASE_URL="${BASE_URL:-http://avandab.com:8092/dist}"

VERSION="$(date +%Y%m%d%H%M%S)"

echo -e "${BLUE}==============================================================================${NC}"
echo -e "${BLUE}🚀 Packaging Remote Release Version: ${VERSION}${NC}"
echo -e "${BLUE}==============================================================================${NC}"

mkdir -p "$DIST_DIR" bin tmp_pkg/internal

# 1. Compile Server Binary
echo -e "\n${CYAN}[1/4] Compiling Go server binary for ARM64...${NC}"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o tmp_pkg/server ./cmd/server/main.go

# 2. Minify / Copy Assets
echo -e "${CYAN}[2/4] Packaging CSS and templates...${NC}"
npx @tailwindcss/cli -i src/input.css -o internal/static/css/tailwind.css --minify 2>/dev/null || true
cp -r internal/templates tmp_pkg/internal/
cp -r internal/static tmp_pkg/internal/

# 3. Create Tarball Payload
echo -e "${CYAN}[3/4] Creating payload tarball (update.tar.gz)...${NC}"
TAR_NAME="update_${VERSION}.tar.gz"
tar -czf "$DIST_DIR/$TAR_NAME" -C tmp_pkg .
rm -rf tmp_pkg

SHA256=$(sha256sum "$DIST_DIR/$TAR_NAME" | cut -d' ' -f1)

# 4. Generate Manifest JSON
echo -e "${CYAN}[4/4] Writing release manifest...${NC}"
cat << MANIFESTEOF > "$DIST_DIR/manifest.json"
{
  "version": "${VERSION}",
  "download_url": "${BASE_URL}/${TAR_NAME}",
  "sha256": "${SHA256}"
}
MANIFESTEOF

echo -e "\n${GREEN}==============================================================================${NC}"
echo -e "${GREEN}🎉 Version ${VERSION} Published Successfully!${NC}"
echo -e "Manifest URL: ${CYAN}${BASE_URL}/manifest.json${NC}"
echo -e "Payload SHA256: ${YELLOW}${SHA256}${NC}"
echo -e "${BLUE}==============================================================================${NC}\n"
