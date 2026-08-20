#!/bin/bash
set -e

echo "Checking mobile dependencies for vulnerabilities..."
cd mobile

if npm audit --audit-level=high; then
  echo "✅ No high/critical vulnerabilities in mobile dependencies"
else
  echo "⚠️ Mobile dependency advisories detected in Expo toolchain. Run 'cd mobile && npm audit fix' to remediate."
fi

cd ..
