#!/usr/bin/env bash
# knowledge-capture.sh -- Patterns for capturing different types of knowledge
#
# Each function demonstrates the capture pattern for a specific note type.
# Adapt VAULT variable to your environment.

set -euo pipefail

VAULT="Claude"
TODAY=$(date +%Y-%m-%d)

# ─────────────────────────────────────────────
# Capture a Decision
# ─────────────────────────────────────────────
capture_decision() {
  local title="$1"
  local project="$2"
  local context="$3"
  local decision="$4"
  local alternatives="$5"

  vlt vault="$VAULT" create name="$title" \
    path="decisions/$title.md" \
    content="---
type: decision
project: $project
status: active
confidence: medium
created: $TODAY
---
# $title

## Context
$context

## Decision
$decision

## Alternatives Considered
$alternatives

## Consequences
(To be filled as consequences become apparent)" silent timestamps
}

# Example usage:
# capture_decision \
#   "Use SQLite for local storage" \
#   "my-cli" \
#   "Need persistent storage for CLI tool without external dependencies." \
#   "SQLite via modernc.org/sqlite (pure Go, no CGO required)." \
#   "- BoltDB -- simpler but no SQL query support\n- JSON files -- no concurrent access safety"


# ─────────────────────────────────────────────
# Capture a Debug Insight
# ─────────────────────────────────────────────
capture_debug() {
  local title="$1"
  local project="$2"
  local symptoms="$3"
  local root_cause="$4"
  local fix="$5"

  vlt vault="$VAULT" create name="$title" \
    path="debug/$title.md" \
    content="---
type: debug
project: $project
status: active
severity: major
created: $TODAY
---
# $title

## Symptoms
$symptoms

## Root Cause
$root_cause

## Fix
$fix

## Prevention
(How to avoid this in the future)" silent timestamps
}


# ─────────────────────────────────────────────
# Capture a Reusable Pattern
# ─────────────────────────────────────────────
capture_pattern() {
  local title="$1"
  local stack="$2"
  local problem="$3"
  local solution="$4"
  local when_to_use="$5"

  vlt vault="$VAULT" create name="$title" \
    path="patterns/$title.md" \
    content="---
type: pattern
stack: [$stack]
status: active
created: $TODAY
---
# $title

## Problem
$problem

## Solution
$solution

## When to Use
$when_to_use

## Trade-offs
(Document trade-offs as they become apparent)" silent timestamps
}


# ─────────────────────────────────────────────
# Capture a Convention
# ─────────────────────────────────────────────
capture_convention() {
  local title="$1"
  local project="$2"
  local rule="$3"
  local rationale="$4"

  vlt vault="$VAULT" create name="$title" \
    path="conventions/$title.md" \
    content="---
type: convention
project: $project
status: active
created: $TODAY
---
# $title

## Rule
$rule

## Rationale
$rationale

## Exceptions
(Document any known exceptions)" silent timestamps
}


# ─────────────────────────────────────────────
# Capture a Session Handoff
# ─────────────────────────────────────────────
capture_handoff() {
  local project="$1"
  local state="$2"
  local in_progress="$3"
  local blockers="$4"
  local next_steps="$5"

  vlt vault="$VAULT" create name="$project Handoff $TODAY" \
    path="_inbox/$project Handoff $TODAY.md" \
    content="---
type: handoff
project: $project
status: active
created: $TODAY
---
# Handoff: $project ($TODAY)

## Current State
$state

## In Progress
$in_progress

## Blockers
$blockers

## Next Steps
$next_steps" silent timestamps
}


# ─────────────────────────────────────────────
# Quick Capture (Inbox)
# ─────────────────────────────────────────────
quick_capture() {
  local title="$1"
  local content="$2"

  vlt vault="$VAULT" create name="$title" \
    path="_inbox/$title.md" \
    content="---
status: active
created: $TODAY
---
# $title

$content" silent timestamps
}

# Example: quick_capture "Interesting API design idea" "Consider using HATEOAS for the REST API..."


# ─────────────────────────────────────────────
# Triage an Inbox Note
# ─────────────────────────────────────────────
triage_note() {
  local file="$1"
  local type="$2"
  local target_folder="$3"
  local project="${4:-}"

  # Set type property
  vlt vault="$VAULT" property:set file="$file" name="type" value="$type"

  # Set project if provided
  if [ -n "$project" ]; then
    vlt vault="$VAULT" property:set file="$file" name="project" value="$project"
  fi

  # Move to target folder
  vlt vault="$VAULT" move path="_inbox/$file.md" to="$target_folder/$file.md"

  echo "Triaged '$file' -> $target_folder/ (type: $type)"
}

# Example: triage_note "Interesting API design idea" "pattern" "patterns" "my-app"
