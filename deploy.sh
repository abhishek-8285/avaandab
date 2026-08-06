#!/usr/bin/env bash
# FlyFleet (MVTMS) Automated Android VPS Deployment Script
set -e

# Terminate on error formatting colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== FlyFleet Android VPS Deployment Script ===${NC}"

# 1. Verify ADB Connection
echo -e "${CYAN}[1/6] Verifying ADB connection...${NC}"
DEVICE=$(adb devices | grep -E "device$" | head -n 1 | cut -f1 || true)
if [ -z "$DEVICE" ]; then
    echo -e "${RED}Error: No Android device attached via ADB.${NC}"
    exit 1
fi
echo -e "${GREEN}Detected device: $DEVICE${NC}"

# 2. Build Server Binary (Linux ARM64)
echo -e "${CYAN}[2/6] Compiling Go server for Android (ARM64)...${NC}"
mkdir -p bin
env GOOS=linux GOARCH=arm64 go build -o bin/mvtms-arm64 ./cmd/server/
echo -e "${GREEN}Compilation successful! Binary size: $(du -sh bin/mvtms-arm64 | cut -f1)${NC}"

# 3. Prepare Target Directories on Android
echo -e "${CYAN}[3/6] Setting up directory structures on Android VPS...${NC}"
adb shell "su -c 'mkdir -p /data/local/tmp/internal'"

# 4. Push Files via ADB
echo -e "${CYAN}[4/6] Pushing application assets to device...${NC}"
echo "Pushing binary..."
adb push bin/mvtms-arm64 /data/local/tmp/mvtms
echo "Pushing templates..."
adb push internal/templates /data/local/tmp/internal/
echo "Pushing static assets..."
adb push internal/static /data/local/tmp/internal/
echo -e "${GREEN}Asset transfer completed.${NC}"

# 5. Fix Permissions & Ownership
echo -e "${CYAN}[5/6] Correcting file ownership and permissions...${NC}"
adb shell "su -c 'chown -R shell:shell /data/local/tmp/internal /data/local/tmp/mvtms && chmod 755 /data/local/tmp/mvtms && chmod -R 777 /data/local/tmp/internal'"

# 6. Restart Server on Android
echo -e "${CYAN}[6/6] Restarting MVTMS server on Android VPS...${NC}"
# Kill existing instance if running
adb shell "su -c 'pkill -f mvtms || true'"
# Launch server in background
adb shell "su -c 'cd /data/local/tmp && nohup ./mvtms >/data/local/tmp/mvtms.log 2>&1 &'"

# Wait for server to boot and read log status
sleep 2
echo -e "${YELLOW}Server Output Log:${NC}"
adb shell "su -c 'cat /data/local/tmp/mvtms.log'"

# Fetch device IP address details
WLAN_IP=$(adb shell ip addr show wlan0 | grep "inet " | awk '{print $2}' | cut -d/ -f1 || true)

echo -e "\n${GREEN}=== Deployment Successful! ===${NC}"
if [ -n "$WLAN_IP" ]; then
    echo -e "Local Address: ${CYAN}http://$WLAN_IP:8080/${NC} (Direct)"
    echo -e "Caddy proxy:   ${CYAN}http://$WLAN_IP/${NC} (Reverse proxied)"
fi
echo -e "Cloudflare Tunnel URL: ${CYAN}https://flag-find-erik-liberty.trycloudflare.com${NC}"
echo -e "Check live output logs anytime with: ${YELLOW}adb shell \"su -c 'tail -f /data/local/tmp/mvtms.log'\"${NC}\n"
