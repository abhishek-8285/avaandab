#!/usr/bin/env bash
# ==============================================================================
# Pre-Commit Automated Quality & Integrity Gate
# ==============================================================================
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo -e "${BLUE}==============================================================================${NC}"
echo -e "${BLUE}🔍 Running Pre-Commit Verification Suite...${NC}"
echo -e "${BLUE}==============================================================================${NC}"

# 1. Format Check (gofmt)
echo -e "\n${CYAN}[1/5] Checking Go code formatting (gofmt)...${NC}"
UNFORMATTED=$(gofmt -l -s . | grep -v '^dist/' || true)
if [ -n "$UNFORMATTED" ]; then
    echo -e "${RED}❌ Error: The following Go files are not formatted:${NC}"
    echo "$UNFORMATTED"
    echo -e "${YELLOW}Run 'gofmt -w -s .' to fix automatically.${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Formatting check passed.${NC}"

# 2. Go Vet Analysis
echo -e "\n${CYAN}[2/5] Running go vet static analysis...${NC}"
go vet ./...
echo -e "${GREEN}✅ Go vet passed.${NC}"

# 3. Linter (golangci-lint)
echo -e "\n${CYAN}[3/5] Running golangci-lint...${NC}"
go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run ./...
echo -e "${GREEN}✅ Linter check passed.${NC}"

# 4. SQLC Out-of-Date Check
echo -e "\n${CYAN}[4/5] Checking sqlc generated files integrity...${NC}"
if [ -f "/tmp/go/bin/sqlc" ]; then
    /tmp/go/bin/sqlc generate
else
    go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
fi

if [ -n "$(git status --porcelain db/generated/)" ]; then
    echo -e "${RED}❌ Error: sqlc generated files are out of date.${NC}"
    git status --porcelain db/generated/
    echo -e "${YELLOW}Run 'sqlc generate' and commit the updated files.${NC}"
    exit 1
fi
echo -e "${GREEN}✅ sqlc files up to date.${NC}"

# 5. Unit Tests
echo -e "\n${CYAN}[5/5] Executing Go unit tests...${NC}"
go test -v ./...
echo -e "${GREEN}✅ All unit tests passed.${NC}"

echo -e "\n${GREEN}==============================================================================${NC}"
echo -e "${GREEN}🎉 All pre-commit checks passed! Your commit is ready to push.${NC}"
echo -e "${BLUE}==============================================================================${NC}\n"
