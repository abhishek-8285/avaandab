#!/bin/bash
set -e

export PATH="$(go env GOPATH)/bin:$PATH"

# Install golangci-lint if not present
if ! command -v golangci-lint &> /dev/null; then
  echo "Installing golangci-lint..."
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
fi

echo "Running golangci-lint..."
golangci-lint run --timeout=5m ./...

echo "✅ golangci-lint passed"
