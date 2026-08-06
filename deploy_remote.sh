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

SSH_HOST="flyfleet"
REMOTE_PATH="/data/local/tmp"

echo -e "${BLUE}=== FlyFleet Remote SSH Deployment Script ===${NC}"

# 1. Test SSH connection
echo -e "${CYAN}[1/5] Verifying SSH connection to Android VPS (${SSH_HOST})...${NC}"
if ! ssh -F ~/.ssh/config -q -o ConnectTimeout=5 "${SSH_HOST}" "echo hello" >/dev/null; then
    echo -e "${RED}Error: Cannot connect to Android VPS via SSH (${SSH_HOST}).${NC}"
    echo -e "Ensure the Android device is online and SSH daemon is running."
    exit 1
fi
echo -e "${GREEN}SSH connection established successfully.${NC}"

# 2. Build Server Binary locally
echo -e "${CYAN}[2/5] Compiling Go server locally for Android (ARM64)...${NC}"
mkdir -p bin
env GOOS=linux GOARCH=arm64 go build -o bin/mvtms-arm64 ./cmd/server/
echo -e "${GREEN}Compilation successful! Binary size: $(du -sh bin/mvtms-arm64 | cut -f1)${NC}"

# 3. Prepare target directory and clean old binary
echo -e "${CYAN}[3/5] Cleaning old binary and preparing remote directory...${NC}"
ssh -F ~/.ssh/config "${SSH_HOST}" "su -c 'pkill mvtms || true && rm -f ${REMOTE_PATH}/mvtms && mkdir -p ${REMOTE_PATH}/internal'"

# 4. Copy files via scp
echo -e "${CYAN}[4/5] Copying binary and assets to remote VPS...${NC}"
echo "Pushing binary..."
scp -F ~/.ssh/config bin/mvtms-arm64 "${SSH_HOST}":"${REMOTE_PATH}/mvtms"
echo "Pushing templates..."
scp -F ~/.ssh/config -r internal/templates "${SSH_HOST}":"${REMOTE_PATH}/internal/"
echo "Pushing static assets..."
scp -F ~/.ssh/config -r internal/static "${SSH_HOST}":"${REMOTE_PATH}/internal/"
echo -e "${GREEN}Asset transfer completed.${NC}"

# 5. Correct permissions and start the server
echo -e "${CYAN}[5/5] Restarting server process on remote Android VPS...${NC}"
ssh -F ~/.ssh/config "${SSH_HOST}" "su -c 'chown -R shell:shell ${REMOTE_PATH}/internal ${REMOTE_PATH}/mvtms && chmod 755 ${REMOTE_PATH}/mvtms && chmod -R 777 ${REMOTE_PATH}/internal && pkill mvtms || true && cd ${REMOTE_PATH} && nohup ./mvtms >${REMOTE_PATH}/mvtms.log 2>&1 &'"

sleep 2
echo -e "${YELLOW}Remote Server Output Log:${NC}"
ssh -F ~/.ssh/config "${SSH_HOST}" "su -c 'cat ${REMOTE_PATH}/mvtms.log'"

echo -e "\n${GREEN}=== Remote SSH Deployment Successful! ===${NC}"
echo -e "Access domain: ${CYAN}https://flag-find-erik-liberty.trycloudflare.com${NC} / ${CYAN}http://flyfleet.duckdns.org${NC}"
echo -e "Check live logs via SSH: ${YELLOW}ssh ${SSH_HOST} \"su -c 'tail -f ${REMOTE_PATH}/mvtms.log'\"${NC}\n"
