#!/usr/bin/env bash
# Fails when the Android JS bundle (Hermes bytecode) grows past the budget.
# It stands in for the cold-start budget in CI, which has no device: a bigger
# bundle is slower to load. Raise BUDGET_BYTES on purpose, in a reviewed change.
set -euo pipefail

BUDGET_BYTES="${BUDGET_BYTES:-5242880}" # 5 MiB; the baseline on 2026-10-31 was 4.05 MB
out="$(mktemp -d)"
trap 'rm -rf "$out"' EXIT

cd "$(dirname "$0")/../apps/mobile"
pnpm exec expo export --platform android --output-dir "$out" >/dev/null

size=$(find "$out/_expo/static/js/android" -name '*.hbc' -exec stat -c %s {} + | awk '{s+=$1} END {print s+0}')
if [ "$size" -eq 0 ]; then
  echo "No Hermes bundle found; the export layout changed." >&2
  exit 1
fi
echo "Android bundle: $size bytes (budget $BUDGET_BYTES)"
if [ "$size" -gt "$BUDGET_BYTES" ]; then
  echo "Bundle is over budget. See docs/performance.md." >&2
  exit 1
fi
