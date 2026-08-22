#!/bin/bash
set -e

export PATH="$(go env GOPATH)/bin:$PATH"

# Install golangci-lint if not present
if ! command -v golangci-lint &> /dev/null; then
  echo "Installing golangci-lint..."
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
fi

echo "Running golangci-lint..."
# LINT_BASE (optional): git rev to diff against — gates only new code.
# Used by pre-commit so legacy debt doesn't block commits; full run when unset.
ARGS=(run --timeout=8m)
if [ -n "$LINT_BASE" ]; then
  ARGS+=("--new-from-rev=$LINT_BASE")
fi
golangci-lint "${ARGS[@]}" ./...

echo "✅ golangci-lint passed"
