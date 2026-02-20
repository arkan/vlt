#!/usr/bin/env bash
# vault-maintenance.sh -- Vault hygiene operations
#
# Run periodically to keep the vault healthy and discoverable.
# Adapt VAULT variable to your environment.

set -euo pipefail

VAULT="Claude"

echo "=== Vault Maintenance Report ==="
echo ""

# ─────────────────────────────────────────────
# 1. Orphan Notes (No Incoming Links)
# ─────────────────────────────────────────────
echo "--- Orphan Notes ---"
orphans=$(vlt vault="$VAULT" orphans 2>/dev/null || true)
orphan_count=$(echo "$orphans" | grep -c . 2>/dev/null || echo "0")

if [ "$orphan_count" -gt 0 ]; then
  echo "Found $orphan_count orphan notes (no incoming links):"
  echo "$orphans" | while read -r note; do
    echo "  - $note"
  done
  echo ""
  echo "  Action: Add [[wikilinks]] to these notes from related notes,"
  echo "  or archive them if they are no longer relevant."
else
  echo "  No orphan notes found."
fi
echo ""

# ─────────────────────────────────────────────
# 2. Broken Links (Unresolved Wikilinks)
# ─────────────────────────────────────────────
echo "--- Broken Links ---"
broken=$(vlt vault="$VAULT" unresolved 2>/dev/null || true)
broken_count=$(echo "$broken" | grep -c . 2>/dev/null || echo "0")

if [ "$broken_count" -gt 0 ]; then
  echo "Found $broken_count broken wikilinks:"
  echo "$broken" | head -20 | while IFS=$'\t' read -r target source; do
    echo "  [[${target}]] in ${source}"
  done
  if [ "$broken_count" -gt 20 ]; then
    echo "  ... and $((broken_count - 20)) more"
  fi
  echo ""
  echo "  Action: Create the missing notes or fix the link targets."
else
  echo "  No broken links found."
fi
echo ""

# ─────────────────────────────────────────────
# 3. Inbox Triage Status
# ─────────────────────────────────────────────
echo "--- Inbox Status ---"
inbox=$(vlt vault="$VAULT" files folder="_inbox" 2>/dev/null || true)
inbox_count=$(echo "$inbox" | grep -c . 2>/dev/null || echo "0")

if [ "$inbox_count" -gt 0 ]; then
  echo "Found $inbox_count notes in _inbox/ awaiting triage:"
  echo "$inbox" | while read -r note; do
    echo "  - $note"
  done
  echo ""
  echo "  Action: Classify, add frontmatter, add links, move to proper folder."
else
  echo "  Inbox is clean."
fi
echo ""

# ─────────────────────────────────────────────
# 4. Tag Audit
# ─────────────────────────────────────────────
echo "--- Tag Summary ---"
vlt vault="$VAULT" tags counts sort="count" 2>/dev/null || echo "  No tags found."
echo ""

# ─────────────────────────────────────────────
# 5. Notes Missing Required Frontmatter
# ─────────────────────────────────────────────
echo "--- Frontmatter Audit ---"
missing_type=0
missing_status=0

for f in $(vlt vault="$VAULT" files ext=".md" 2>/dev/null | grep -v "^_templates/"); do
  # Extract just the filename without path for the file= parameter
  basename=$(basename "$f" .md)
  props=$(vlt vault="$VAULT" properties file="$basename" 2>/dev/null || true)

  if [ -n "$props" ]; then
    if ! echo "$props" | grep -q "^type:"; then
      echo "  Missing 'type': $f"
      missing_type=$((missing_type + 1))
    fi
    if ! echo "$props" | grep -q "^status:"; then
      echo "  Missing 'status': $f"
      missing_status=$((missing_status + 1))
    fi
  fi
done

if [ "$missing_type" -eq 0 ] && [ "$missing_status" -eq 0 ]; then
  echo "  All notes have required frontmatter properties."
fi
echo ""

# ─────────────────────────────────────────────
# 6. Pending Tasks
# ─────────────────────────────────────────────
echo "--- Pending Tasks ---"
tasks=$(vlt vault="$VAULT" tasks pending 2>/dev/null || true)
task_count=$(echo "$tasks" | grep -c . 2>/dev/null || echo "0")

if [ "$task_count" -gt 0 ]; then
  echo "Found $task_count pending tasks across the vault:"
  echo "$tasks" | head -15
  if [ "$task_count" -gt 15 ]; then
    echo "  ... and $((task_count - 15)) more"
  fi
else
  echo "  No pending tasks found."
fi
echo ""

# ─────────────────────────────────────────────
# Summary
# ─────────────────────────────────────────────
echo "=== Summary ==="
echo "  Orphan notes:     $orphan_count"
echo "  Broken links:     $broken_count"
echo "  Inbox items:      $inbox_count"
echo "  Missing 'type':   $missing_type"
echo "  Missing 'status': $missing_status"
echo "  Pending tasks:    $task_count"
echo ""

total_issues=$((orphan_count + broken_count + inbox_count + missing_type + missing_status))
if [ "$total_issues" -eq 0 ]; then
  echo "Vault is healthy."
else
  echo "Total issues found: $total_issues"
fi
