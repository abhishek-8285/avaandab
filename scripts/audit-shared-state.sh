#!/bin/bash
set -e

echo "=== Shared State Audit ==="
echo ""

# 1. Package-level var declarations in test files
echo "--- Package-level vars in _test.go files ---"
grep -rn "^var " --include="*_test.go" internal/ test/ | grep -v "^.*// " | head -20 || echo "(none found)"
echo ""

# 2. Package-level caches/registries
echo "--- Global caches/registries (map/slice) in non-test code ---"
grep -rn "^var.*= map\[" internal/ --include="*.go" | head -20 || echo "(none found)"
grep -rn "^var.*= make(map" internal/ --include="*.go" | head -10 || echo "(none found)"
echo ""

# 3. sync.Mutex at package level (potential shared locks)
echo "--- Package-level mutexes ---"
grep -rn "^var.*sync\.Mutex" internal/ --include="*.go" | head -10 || echo "(none found)"
echo ""

# 4. init() functions that modify global state
echo "--- init() functions ---"
grep -rn "^func init()" internal/ --include="*.go" | head -10 || echo "(none found)"
echo ""

echo "=== Audit complete ==="
