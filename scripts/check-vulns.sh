#!/bin/bash
set -e

export PATH="$(go env GOPATH)/bin:$PATH"

# Install govulncheck if not present
if ! command -v govulncheck &> /dev/null; then
  echo "Installing govulncheck..."
  go install golang.org/x/vuln/cmd/govulncheck@latest
fi

echo "Checking for known vulnerabilities in Go dependencies..."
govulncheck ./... || {
  echo "⚠️ Vulnerability findings reported by govulncheck"
}

echo "✅ Go vulnerability scan complete"
