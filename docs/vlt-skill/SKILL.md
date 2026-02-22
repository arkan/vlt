---
name: vlt - Obsidian Vault CLI
description: >-
  This skill should be used when the user asks to "read a vault note",
  "create a note", "search the vault", "manage frontmatter properties",
  "find backlinks", "check orphan notes", "work with daily notes",
  "apply a template", "manage bookmarks", "find broken links",
  "append to a note", "patch a note section", "list vault tags",
  "manage tasks in vault", "move or rename a note", "delete a note",
  "generate an obsidian URI", or mentions Obsidian vault operations,
  vlt CLI, or vault-backed knowledge management.
  Provides comprehensive guidance for using vlt in agentic AI workflows,
  CI/CD pipelines, and shell scripting.
version: 0.5.0
---

# vlt -- Obsidian Vault CLI for Coding Agents

vlt is a fast, zero-dependency CLI for Obsidian vault operations. It reads and writes
vault files directly on the filesystem without requiring the Obsidian desktop app,
Electron, Node.js, or any network calls. Purpose-built for agentic AI workflows,
CI/CD pipelines, and shell scripting.

## When to Use This Skill

- Reading, creating, editing, or searching notes in an Obsidian vault
- Managing YAML frontmatter properties on notes
- Navigating vault structure via links, backlinks, tags, and bookmarks
- Building knowledge management workflows for AI agent sessions
- Automating vault maintenance (orphans, broken links, unresolved references)
- Working with daily notes, templates, or tasks

## Core Concepts

### Vault Discovery

vlt locates vaults from Obsidian's config or via explicit parameters:

```bash
vlt vault="MyVault" read file="Note"        # By vault name
vlt vault="/absolute/path" read file="Note"  # By absolute path
vlt vault="~/path" read file="Note"          # By home-relative path
```

Environment variables `VLT_VAULT` and `VLT_VAULT_PATH` set defaults.
Run `vlt vaults` to list all discovered vaults.

### Note Resolution

vlt resolves note titles using a two-pass algorithm:
1. **Fast pass** -- exact filename match (`<title>.md`), no file I/O
2. **Alias pass** -- checks frontmatter `aliases` field (case-insensitive)

Reference notes by filename (without `.md`) or by any alias.

### Parameter Syntax

All commands use `key="value"` pairs. Boolean flags are bare words:

```bash
vlt vault="V" read file="Note" heading="## Section"
vlt vault="V" create name="Title" path="folder/Title.md" content="..." silent timestamps
```

### Output Formats

All listing commands support structured output:
`--json`, `--yaml`, `--csv`, `--tsv`, `--tree` (files only), or plain text (default).

## Command Quick Reference

### File Operations

| Command | Purpose | Key Parameters |
|---------|---------|----------------|
| `read` | Print note content | `file=`, `heading=` (optional section) |
| `create` | Create a new note | `name=`, `path=`, `content=`, `silent`, `timestamps` |
| `append` | Add content (end, section, or after line). **NOT for tasks -- use `tasks:add`** | `file=`, `content=`, `heading=`, `section=`, `line=` |
| `prepend` | Insert content (after FM, section, or before line) | `file=`, `content=`, `heading=`, `section=`, `line=` |
| `write` | Replace body, keep frontmatter | `file=`, `content=` (or stdin) |
| `patch` | Edit by heading or line | `file=`, `heading=`/`line=`, `content=`/`delete` |
| `delete` | Trash or hard-delete | `file=`, `permanent` (optional) |
| `move` | Rename with link repair | `path=`, `to=` |
| `daily` | Create/read daily note | `content=`, `template=` (optional) |
| `files` | List vault files | `folder=`, `ext=` (optional filters) |

### Properties

| Command | Purpose | Key Parameters |
|---------|---------|----------------|
| `properties` | Show frontmatter | `file=` |
| `property:set` | Set a property | `file=`, `name=`, `value=` |
| `property:remove` | Remove a property | `file=`, `name=` |

### Links and Navigation

| Command | Purpose | Key Parameters |
|---------|---------|----------------|
| `backlinks` | Notes linking to a note | `file=` |
| `links` | Outgoing links (marks broken) | `file=` |
| `orphans` | Notes with no incoming links | (none) |
| `unresolved` | Broken wikilinks vault-wide | (none) |

### Search

| Command | Purpose | Key Parameters |
|---------|---------|----------------|
| `search` | Find by title, content, properties | `query=`, `regex=`, `context=` |
| `tags` | List all tags | `counts`, `sort="count"` |
| `tag` | Notes with a tag (hierarchical) | `tag=` |
| `tasks` | List checkboxes | `file=`/`path=`, `done`/`pending` |
| `tasks:add` | Add a task to a note | `file=`, `content=`, `heading=`, `section=`, `line=`, `due=`, `priority=`, `--emoji` |
| `tasks:edit` | Edit task text/metadata/status | `file=`, `{id=\|line=\|match=}`, `content=`, `due=`, `priority=`, `status=` |
| `tasks:remove` | Remove a task line | `file=`, `{id=\|line=\|match=}` |
| `tasks:done` | Mark task as completed | `file=`, `{id=\|line=\|match=}` |
| `tasks:toggle` | Toggle done/pending | `file=`, `{id=\|line=\|match=}` |

### Templates, Bookmarks, URI

| Command | Purpose | Key Parameters |
|---------|---------|----------------|
| `templates` | List available templates | (none) |
| `templates:apply` | Create note from template | `template=`, `name=`, `path=` |
| `bookmarks` | List bookmarks | (none) |
| `bookmarks:add` | Bookmark a note | `file=` |
| `bookmarks:remove` | Remove bookmark | `file=` |
| `uri` | Generate `obsidian://` URI | `file=`, `heading=`, `block=` |

## Agentic Session Workflow

### Session Start -- Load Context

```bash
# Discover what the vault knows about the current project
vlt vault="Claude" search query="<project-name>"
vlt vault="Claude" search query="[type:decision] [project:<name>]"
vlt vault="Claude" search query="[type:pattern] [status:active]"
```

### During Work -- Capture Knowledge

```bash
# Capture a decision
vlt vault="Claude" create name="Use WebSockets over SSE" \
  path="decisions/Use WebSockets over SSE.md" \
  content="---
type: decision
project: my-app
status: active
created: 2026-02-19
---
# Use WebSockets over SSE
## Context
Real-time updates needed for dashboard.
## Decision
WebSockets chosen for bidirectional communication.
## Alternatives
SSE -- simpler but one-directional." silent timestamps

# Capture a debug insight
vlt vault="Claude" create name="PostgreSQL JSONB index gotcha" \
  path="_inbox/PostgreSQL JSONB index gotcha.md" \
  content="---
type: debug
status: active
created: 2026-02-19
---
# PostgreSQL JSONB index gotcha
## Symptoms
Slow queries on JSONB column despite GIN index.
## Root Cause
GIN index does not support ordering; query plan fell back to seq scan.
## Fix
Add a B-tree index on the extracted scalar value." silent
```

### Session End -- Update Project Index

```bash
vlt vault="Claude" append file="projects/my-app" \
  content="## Session 2026-02-19
- Implemented WebSocket transport
- Discovered JSONB index limitation
- [[Use WebSockets over SSE]]"
```

## Search Patterns

### Text Search
```bash
vlt vault="V" search query="authentication"
```

### Property-Filtered Search
```bash
vlt vault="V" search query="[status:active] [type:decision]"
vlt vault="V" search query="caching [project:my-app]"
```

### Regex Search with Context
```bash
vlt vault="V" search regex="TODO|FIXME|HACK" context="2"
```

## Content Manipulation

### Replace a Section
```bash
vlt vault="V" patch file="Note" heading="## Status" content="Completed 2026-02-19."
```

### Edit by Line Number
```bash
vlt vault="V" patch file="Note" line="5" content="Updated line."
vlt vault="V" patch file="Note" line="10-15" content="Replaced block."
```

### Delete a Section
```bash
vlt vault="V" patch file="Note" heading="## Deprecated" delete
```

### Append/Prepend with Positioning
```bash
# Append at end of a section
vlt vault="V" append file="Note" heading="## Log" content="New entry"

# Append at start of a section
vlt vault="V" append file="Note" heading="## Log" section="start" content="Top entry"

# Append after a specific line
vlt vault="V" append file="Note" line="5" content="After line 5"

# Prepend at start of a section (after heading)
vlt vault="V" prepend file="Note" heading="## TODO" content="- [ ] Urgent"

# Prepend at end of a section
vlt vault="V" prepend file="Note" heading="## TODO" section="end" content="Last item"

# Prepend before a specific line
vlt vault="V" prepend file="Note" line="10" content="Before line 10"
```

### Replace Entire Body (Keep Frontmatter)
```bash
vlt vault="V" write file="Note" content="New body content."
```

## Task Management

> **IMPORTANT**: ALWAYS use `tasks:add`, `tasks:edit`, `tasks:done`, `tasks:remove`, or `tasks:toggle` for task operations. NEVER use `append` or `prepend` to add tasks -- they skip metadata (id, created date) and bypass task formatting.

vlt supports full CRUD on Obsidian Tasks checkboxes with metadata in both **Dataview** (`[key:: value]`) and **Emoji** (📅, ⏫) formats.

### Task Metadata Fields

| Field | Dataview | Emoji | CLI param |
|-------|----------|-------|-----------|
| Due | `[due:: DATE]` | 📅 | `due=` |
| Scheduled | `[scheduled:: DATE]` | ⏳ | `scheduled=` |
| Start | `[start:: DATE]` | 🛫 | `start=` |
| Created | `[created:: DATE]` | ➕ | `created=` (auto-filled) |
| Completion | `[completion:: DATE]` | ✅ | `completion=` |
| Priority | `[priority:: LEVEL]` | ⏬🔽🔼⏫🔺 | `priority=` |
| Recurrence | `[repeat:: RULE]` | 🔁 | `repeat=` |
| ID | `[id:: ID]` | 🆔 | `id=` |
| Depends On | `[dependsOn:: IDS]` | ⛔ | `dependsOn=` |

Priority levels: `lowest`, `low`, `medium`, `high`, `highest`.

### Task Resolution

Tasks are identified by (in priority order):
1. **ID**: `id="abc123"` -- finds `[id:: abc123]` or 🆔 abc123
2. **Line number**: `line="5"` -- task at line 5
3. **Text match**: `match="groceries"` -- first task containing text (case-insensitive)

### Add a Task

```bash
# Add at end of file (Dataview format, auto-fills created date)
vlt vault="V" tasks:add file="Note" content="Buy groceries" due="2024-01-15" priority="high"

# Add after a heading (at section start or end)
vlt vault="V" tasks:add file="Note" content="Review PR" heading="## TODO" section="end"

# Add at a specific line
vlt vault="V" tasks:add file="Note" content="Call dentist" line="10"

# Add in emoji format
vlt vault="V" tasks:add file="Note" content="Ship feature" due="2024-06-01" --emoji
```

### Edit a Task

```bash
# Change text
vlt vault="V" tasks:edit file="Note" line="5" content="Updated text"

# Change metadata by ID
vlt vault="V" tasks:edit file="Note" id="abc" due="2024-02-01"

# Clear a field with "-"
vlt vault="V" tasks:edit file="Note" match="groceries" due="-"

# Change status
vlt vault="V" tasks:edit file="Note" line="5" status="done"
```

### Remove, Done, Toggle

```bash
# Remove a task line entirely
vlt vault="V" tasks:remove file="Note" line="5"

# Mark as done (sets [x] + completion date)
vlt vault="V" tasks:done file="Note" match="groceries"

# Toggle done/pending
vlt vault="V" tasks:toggle file="Note" id="abc"
```

## Stdin Support

Commands accepting `content=` also accept stdin when `content=` is omitted:

```bash
date | vlt vault="V" append file="Daily Log"
echo "New content" | vlt vault="V" write file="Note"
cat data.md | vlt vault="V" create name="Import" path="_inbox/Import.md"
```

## Important Behaviors

- **Exit codes**: 0 on success, 1 on error. Empty results exit 0 silently (Unix convention).
- **Error output**: Errors go to stderr with `vlt:` prefix.
- **Link repair on move**: `move` updates all wikilinks and markdown links vault-wide.
- **Inert zones**: Links, tags, and references inside code blocks, comments, and math are ignored.
- **Timestamps**: Opt-in via `timestamps` flag or `VLT_TIMESTAMPS=1` env var.
- **Case-insensitive**: Tag matching and alias resolution are case-insensitive.

## Additional Resources

### Reference Files

For detailed documentation beyond this overview, consult:
- **`references/command-reference.md`** -- Complete command reference with all parameters, flags, and edge cases
- **`references/agentic-patterns.md`** -- Proven patterns for AI agent knowledge management workflows
- **`references/advanced-techniques.md`** -- Advanced features: inert zones, link repair, property search, templates
- **`references/vault-architecture.md`** -- Vault design principles, frontmatter conventions, folder structure, linking strategies

### Example Files

Working examples in `examples/`:
- **`examples/session-workflow.sh`** -- Complete AI agent session lifecycle (start, work, end)
- **`examples/knowledge-capture.sh`** -- Patterns for capturing decisions, debug insights, and patterns
- **`examples/vault-maintenance.sh`** -- Vault hygiene: orphans, broken links, tag audits

### Scripts

Utility scripts in `scripts/`:
- **`scripts/vault-health-check.sh`** -- Validate vault health, find issues, report statistics
