#!/usr/bin/env bash

set -euo pipefail

UNIT_FILE="coverage_unit.out"
INTEGRATION_FILE="coverage_integration.out"
MERGED_FILE="coverage.out"

if [[ ! -f "$UNIT_FILE" ]]; then
  echo "missing $UNIT_FILE"
  exit 1
fi

cp "$UNIT_FILE" "$MERGED_FILE"

if [[ -f "$INTEGRATION_FILE" ]]; then
  tail -n +2 "$INTEGRATION_FILE" >> "$MERGED_FILE"
fi
