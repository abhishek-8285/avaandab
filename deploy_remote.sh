#!/usr/bin/env bash
# FlyFleet (MVTMS) Fully Remote SSH Deployment Script
set -e

# Terminate on error formatting colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

SSH_HOST="${1:-tecno-wifi}"
REMOTE_PATH="/data/data/com.termux/files/home/app"
PORT="8090"

echo -e "${BLUE}=== FlyFleet Remote SSH Deployment Script ===${NC}"
echo -e "${CYAN}Target Host: ${SSH_HOST}${NC}"

# 1. Test SSH connection
echo -e "${CYAN}[1/5] Verifying SSH connection to Android device (${SSH_HOST})...${NC}"
if ! ssh -F ~/.ssh/config -q -o ConnectTimeout=5 "${SSH_HOST}" "echo hello" >/dev/null; then
    echo -e "${RED}Error: Cannot connect to Android device via SSH (${SSH_HOST}).${NC}"
    echo -e "Ensure the Android device is online and SSH daemon is running."
    exit 1
fi
echo -e "${GREEN}SSH connection established successfully.${NC}"

# 2. Build Server Binary locally
echo -e "${CYAN}[2/5] Compiling Go server locally for Android (ARM64)...${NC}"
mkdir -p bin
env GOOS=linux GOARCH=arm64 go build -o bin/mvtms-arm64 ./cmd/server/
echo -e "${GREEN}Compilation successful! Binary size: $(du -sh bin/mvtms-arm64 | cut -f1)${NC}"

# 3. Clean existing running instances on remote device
echo -e "${CYAN}[3/5] Terminating previous server process on TECNO phone...${NC}"
ssh -F ~/.ssh/config "${SSH_HOST}" "pkill -9 mvtms-arm64 2>/dev/null || true; pkill -9 server 2>/dev/null || true; mkdir -p ${REMOTE_PATH}/internal"

# 4. Copy binary, DB & assets via scp
echo -e "${CYAN}[4/5] Syncing updated binary, templates, static files & DB to remote device...${NC}"
echo "Pushing binary..."
scp -F ~/.ssh/config bin/mvtms-arm64 "${SSH_HOST}":"${REMOTE_PATH}/mvtms-arm64"
if [ -f "mvtms.db" ]; then
    echo "Pushing database (mvtms.db)..."
    scp -F ~/.ssh/config mvtms.db "${SSH_HOST}":"${REMOTE_PATH}/mvtms.db"
fi
echo "Pushing templates..."
scp -F ~/.ssh/config -r internal/templates "${SSH_HOST}":"${REMOTE_PATH}/internal/"
echo "Pushing static assets..."
scp -F ~/.ssh/config -r internal/static "${SSH_HOST}":"${REMOTE_PATH}/internal/"
echo -e "${GREEN}Asset transfer completed.${NC}"

# 5. Start the server daemon cleanly
echo -e "${CYAN}[5/5] Launching fresh server process (Port ${PORT})...${NC}"
ssh -F ~/.ssh/config "${SSH_HOST}" "chmod +x ${REMOTE_PATH}/mvtms-arm64 && cd ${REMOTE_PATH} && PORT=${PORT} DATABASE_URL='file:mvtms.db?cache=shared&mode=rwc' COOKIE_SECRET='dev-secret-32bytes-for-cookie-signing!' APP_DOMAIN='avandab.com' nohup ./mvtms-arm64 > server.log 2>&1 &"

sleep 3
echo -e "${YELLOW}Remote Server Boot Output Log:${NC}"
ssh -F ~/.ssh/config "${SSH_HOST}" "cat ${REMOTE_PATH}/server.log"

echo -e "\n${GREEN}=== Remote SSH Deployment Successful! ===${NC}"
echo -e "Access via local ADB forward: ${CYAN}http://localhost:${PORT}${NC}"
echo -e "Check live logs via SSH: ${YELLOW}ssh ${SSH_HOST} \"tail -f ${REMOTE_PATH}/server.log\"${NC}\n"


