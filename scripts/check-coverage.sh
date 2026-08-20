#!/bin/bash
set -e

# Backend coverage check
echo "Checking backend coverage..."
go test -coverprofile=coverage.out -covermode=atomic ./test

# Apply exclusions if .covignore exists
if [ -f .covignore ]; then
  grep -v -f .covignore coverage.out > coverage.filtered.out || true
  coverage=$(go tool cover -func=coverage.filtered.out | grep total | awk '{print $3}' | sed 's/%//')
else
  coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
fi

echo "Backend coverage: ${coverage}%"

# Enforce 80% threshold
threshold=80
if (( $(echo "$coverage < $threshold" | bc -l) )); then
  echo "❌ Coverage ${coverage}% is below threshold ${threshold}%"
  echo "Run 'go test -coverprofile=coverage.out && go tool cover -html=coverage.out' to inspect"
  exit 1
fi

echo "✅ Backend coverage ${coverage}% meets threshold ${threshold}%"

# Mobile coverage check
echo "Checking mobile coverage..."
cd mobile
npm run coverage || {
  echo "❌ Mobile coverage below threshold"
  exit 1
}
cd ..

echo "✅ All coverage checks passed"
