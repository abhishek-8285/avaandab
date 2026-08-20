#!/bin/bash
set -e

echo "=== Security Scanner Suite ==="

./scripts/lint.sh
./scripts/check-vulns.sh
./scripts/check-npm-audit.sh

echo "✅ All security checks passed"
