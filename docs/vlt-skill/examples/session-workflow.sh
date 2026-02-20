#!/usr/bin/env bash
# session-workflow.sh -- Complete AI agent session lifecycle using vlt
#
# Demonstrates the three-phase pattern: Load Context, Work, Persist.
# Adapt VAULT and PROJECT variables to your environment.

set -euo pipefail

VAULT="Claude"
PROJECT="my-app"
TODAY=$(date +%Y-%m-%d)

# ─────────────────────────────────────────────
# Phase 1: Load Context (Session Start)
# ─────────────────────────────────────────────

echo "=== Phase 1: Loading context for $PROJECT ==="

# Read the project index note
echo "--- Project Index ---"
vlt vault="$VAULT" read file="projects/$PROJECT" 2>/dev/null || echo "(no project index found)"

# Load active decisions
echo "--- Active Decisions ---"
vlt vault="$VAULT" search query="[type:decision] [project:$PROJECT] [status:active]" 2>/dev/null || echo "(none)"

# Load known patterns
echo "--- Active Patterns ---"
vlt vault="$VAULT" search query="[type:pattern] [project:$PROJECT] [status:active]" 2>/dev/null || echo "(none)"

# Check for known bugs and gotchas
echo "--- Debug Notes ---"
vlt vault="$VAULT" search query="[type:debug] [project:$PROJECT]" 2>/dev/null || echo "(none)"

# Check for recent handoff notes
echo "--- Recent Handoffs ---"
vlt vault="$VAULT" search query="[type:handoff] [project:$PROJECT] [status:active]" 2>/dev/null || echo "(none)"


# ─────────────────────────────────────────────
# Phase 2: Work (Capture Knowledge As It Emerges)
# ─────────────────────────────────────────────

echo ""
echo "=== Phase 2: Working... ==="

# Example: Capture a decision made during work
vlt vault="$VAULT" create name="Use connection pooling for DB" \
  path="decisions/Use connection pooling for DB.md" \
  content="---
type: decision
project: $PROJECT
stack: [go, postgresql]
status: active
confidence: high
created: $TODAY
---
# Use connection pooling for DB

## Context
Direct connections exhaust PostgreSQL's default 100-connection limit under load.

## Decision
Use pgxpool (part of pgx v5) for connection pooling within the Go application.

## Alternatives Considered
- PgBouncer -- adds operational complexity for a single-service app
- database/sql pool -- less control over pool behavior, no pgx-specific features

## Consequences
- Must configure pool size based on expected concurrency
- Connection lifecycle managed by pgxpool, not manually" silent timestamps 2>/dev/null \
  && echo "Decision captured: Use connection pooling for DB" \
  || echo "Decision already exists, skipping"

# Example: Capture a debug insight discovered during work
vlt vault="$VAULT" create name="Go context cancellation in HTTP handlers" \
  path="debug/Go context cancellation in HTTP handlers.md" \
  content="---
type: debug
project: $PROJECT
stack: [go]
status: active
severity: major
created: $TODAY
---
# Go context cancellation in HTTP handlers

## Symptoms
Background goroutines spawned from HTTP handlers were being cancelled unexpectedly
when the client disconnected.

## Root Cause
\`r.Context()\` is tied to the HTTP request lifecycle. When the client closes the
connection, the context is cancelled, killing any goroutines using that context.

## Fix
Create a detached context for background work:
\`\`\`go
bgCtx := context.WithoutCancel(r.Context())
\`\`\`

## Prevention
Never pass request contexts to long-running goroutines. Always detach first." silent timestamps 2>/dev/null \
  && echo "Debug note captured: Go context cancellation" \
  || echo "Debug note already exists, skipping"


# ─────────────────────────────────────────────
# Phase 3: Persist (Session End)
# ─────────────────────────────────────────────

echo ""
echo "=== Phase 3: Persisting session results ==="

# Update the project index with session summary
vlt vault="$VAULT" append file="projects/$PROJECT" \
  content="
## Session $TODAY
- Implemented connection pooling with pgxpool
- Discovered context cancellation gotcha in HTTP handlers
- [[Use connection pooling for DB]]
- [[Go context cancellation in HTTP handlers]]" 2>/dev/null \
  && echo "Project index updated" \
  || echo "Project index not found; consider creating it"

# Archive any consumed handoff notes
for handoff in $(vlt vault="$VAULT" search query="[type:handoff] [project:$PROJECT] [status:active]" 2>/dev/null); do
  vlt vault="$VAULT" property:set file="$handoff" name="status" value="archived"
  echo "Archived handoff: $handoff"
done

echo ""
echo "=== Session complete ==="
