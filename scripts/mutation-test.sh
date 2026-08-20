#!/bin/bash
# Mutation testing runner for Go core business logic
# Targets service files that have dedicated _test.go coverage

set -euo pipefail

THRESHOLD=70

MUTESTING_BIN="$(go env GOPATH)/bin/go-mutesting"
if [ ! -x "$MUTESTING_BIN" ]; then
  if command -v go-mutesting > /dev/null 2>&1; then
    MUTESTING_BIN="$(command -v go-mutesting)"
  else
    echo "Installing go-mutesting..."
    go install github.com/avito-tech/go-mutesting/cmd/go-mutesting@latest
    MUTESTING_BIN="$(go env GOPATH)/bin/go-mutesting"
  fi
fi

# Target: files with dedicated test coverage (mutation is only meaningful here)
TARGET_FILES=(
  "internal/service/booking_service.go"
  "internal/service/scorecard_service.go"
  "internal/service/fuel_audit_service.go"
  "internal/service/compliance_service.go"
)

echo "=============================================================================="
echo "🧪 Running Go Mutation Testing Baseline"
echo "=============================================================================="
echo "Binary:    $MUTESTING_BIN"
echo "Targets:   ${#TARGET_FILES[@]} files"
echo "Threshold: ${THRESHOLD}%"
echo ""

TOTAL_MUTANTS=0
TOTAL_KILLED=0
TOTAL_SURVIVED=0

for filepath in "${TARGET_FILES[@]}"; do
  if [ ! -f "$filepath" ]; then
    echo "⚠️  Skipping non-existent file: $filepath"
    continue
  fi

  echo "--- Testing $filepath ---"

  # Run go-mutesting on a single file with generous timeouts
  # --exec-timeout: per-mutant test timeout (seconds)
  OUTPUT=$("$MUTESTING_BIN" \
    --exec-timeout=60 \
    "$filepath" 2>&1) || true

  # Parse summary: "The mutation score is X (N passed, M failed, ... total is T)"
  # go-mutesting: passed = survived (BAD), failed = killed (GOOD)
  SCORE_LINE=$(echo "$OUTPUT" | grep "The mutation score is" | tail -1)

  if [ -n "$SCORE_LINE" ]; then
    SURVIVED=$(echo "$SCORE_LINE" | grep -oP '\d+(?= passed)')
    KILLED=$(echo "$SCORE_LINE"   | grep -oP '\d+(?= failed)')
    TOTAL=$(echo "$SCORE_LINE"    | grep -oP '(?<=total is )\d+')
    SURVIVED=${SURVIVED:-0}
    KILLED=${KILLED:-0}
    TOTAL=${TOTAL:-0}
  else
    # Fallback: count output lines
    KILLED=$(echo "$OUTPUT" | grep -c "^FAIL " 2>/dev/null || echo 0)
    SURVIVED=$(echo "$OUTPUT" | grep -c "^PASS " 2>/dev/null || echo 0)
    TOTAL=$((KILLED + SURVIVED))
  fi

  TOTAL_MUTANTS=$((TOTAL_MUTANTS + TOTAL))
  TOTAL_KILLED=$((TOTAL_KILLED + KILLED))
  TOTAL_SURVIVED=$((TOTAL_SURVIVED + SURVIVED))

  if [ "$TOTAL" -gt 0 ]; then
    RATE=$(( (KILLED * 100) / TOTAL ))
    echo "  Mutants: $TOTAL | Killed: $KILLED | Survived: $SURVIVED | Kill Rate: ${RATE}%"
  else
    echo "  ⚠️  No mutants generated — check output:"
    echo "$OUTPUT" | tail -5 | sed 's/^/    /'
  fi
done

echo ""
echo "=============================================================================="
echo "📊 Mutation Testing Summary"
echo "=============================================================================="
echo "Total mutants generated: $TOTAL_MUTANTS"
echo "Total mutants killed:    $TOTAL_KILLED"
echo "Total mutants survived:  $TOTAL_SURVIVED"

if [ "$TOTAL_MUTANTS" -eq 0 ]; then
  echo "⚠️  No mutants generated."
  exit 0
fi

OVERALL=$(( (TOTAL_KILLED * 100) / TOTAL_MUTANTS ))
echo "Overall Mutation Score (Kill Rate): ${OVERALL}%"

if [ "$OVERALL" -lt "$THRESHOLD" ]; then
  echo "❌ Mutation score ${OVERALL}% is below required threshold ${THRESHOLD}%"
  exit 1
fi

echo "✅ Mutation score ${OVERALL}% meets threshold ${THRESHOLD}%"
exit 0
