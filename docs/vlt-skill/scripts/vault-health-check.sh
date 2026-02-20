#!/usr/bin/env bash
# vault-health-check.sh -- Quick vault health validation
#
# Returns exit 0 if vault is healthy, exit 1 if issues found.
# Designed for CI/CD integration or pre-commit hooks.
#
# Usage:
#   ./vault-health-check.sh <vault-name> [--strict]
#
# Options:
#   --strict    Fail on orphan notes and inbox items (not just broken links)

set -euo pipefail

VAULT="${1:-}"
STRICT="${2:-}"

if [ -z "$VAULT" ]; then
  echo "Usage: vault-health-check.sh <vault-name> [--strict]" >&2
  exit 1
fi

issues=0
warnings=0

# ── Check: Vault accessible ──────────────────
if ! vlt vault="$VAULT" files ext=".md" >/dev/null 2>&1; then
  echo "FAIL: Cannot access vault '$VAULT'" >&2
  exit 1
fi

file_count=$(vlt vault="$VAULT" files ext=".md" 2>/dev/null | wc -l | tr -d ' ')
echo "Vault '$VAULT': $file_count markdown files"

# ── Check: Broken links ──────────────────────
broken_count=$(vlt vault="$VAULT" unresolved 2>/dev/null | wc -l | tr -d ' ')
if [ "$broken_count" -gt 0 ]; then
  echo "FAIL: $broken_count broken wikilinks found"
  vlt vault="$VAULT" unresolved 2>/dev/null | head -5 | while IFS=$'\t' read -r target source; do
    echo "  [[${target}]] in ${source}"
  done
  [ "$broken_count" -gt 5 ] && echo "  ... ($((broken_count - 5)) more)"
  issues=$((issues + broken_count))
fi

# ── Check: Orphan notes ──────────────────────
orphan_count=$(vlt vault="$VAULT" orphans 2>/dev/null | wc -l | tr -d ' ')
if [ "$orphan_count" -gt 0 ]; then
  if [ "$STRICT" = "--strict" ]; then
    echo "FAIL: $orphan_count orphan notes (no incoming links)"
    issues=$((issues + orphan_count))
  else
    echo "WARN: $orphan_count orphan notes (no incoming links)"
    warnings=$((warnings + orphan_count))
  fi
fi

# ── Check: Inbox items ───────────────────────
inbox_count=$(vlt vault="$VAULT" files folder="_inbox" 2>/dev/null | wc -l | tr -d ' ')
if [ "$inbox_count" -gt 0 ]; then
  if [ "$STRICT" = "--strict" ]; then
    echo "FAIL: $inbox_count notes in _inbox/ awaiting triage"
    issues=$((issues + inbox_count))
  else
    echo "WARN: $inbox_count notes in _inbox/ awaiting triage"
    warnings=$((warnings + inbox_count))
  fi
fi

# ── Summary ───────────────────────────────────
echo ""
if [ "$issues" -gt 0 ]; then
  echo "HEALTH: UNHEALTHY ($issues issues, $warnings warnings)"
  exit 1
elif [ "$warnings" -gt 0 ]; then
  echo "HEALTH: OK ($warnings warnings)"
  exit 0
else
  echo "HEALTH: HEALTHY"
  exit 0
fi
