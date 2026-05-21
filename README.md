# agent-memory

Persistent knowledge storage for AI agents. A single Go binary that manages a
local vault of structured notes — agents write observations, the Librarian
promotes them, and everything stays in plain Markdown files you can browse in
Obsidian.

## Install

```
go install github.com/michaelin/agent-memory/cmd/agent-memory@latest
```

Or build from source:

```
git clone https://github.com/michaelin/agent-memory.git
cd agent-memory
go build -o agent-memory ./cmd/agent-memory
```

## Quick start

### 1. Create a vault

```
agent-memory init
```

This creates `.agent-memory/` in the current directory with the full vault
structure: `_inbox/`, `notes/`, `_deprecated/`, `_meta/`, and deployment
templates for the Librarian subagent.

### 2. Write a note

Notes need a body file with Markdown sections. Create one:

```
cat > /tmp/body.md << 'EOF'
# Go errors should wrap context at domain boundaries

## Evidence
Observed in pkg/auth/handler.go lines 45-60: raw SQL errors leak to HTTP layer.

## Implications
Callers cannot distinguish auth failures from database failures without unwrapping.

## Related
- None yet
EOF
```

Then write it to the vault:

```
agent-memory write-note \
  --type observation \
  --title "Go errors should wrap context at domain boundaries" \
  --domain golang,error-handling \
  --scope project \
  --project my-project \
  --source-artifact "pkg/auth/handler.go" \
  --source-agent my-agent \
  --confidence high \
  /tmp/body.md
```

The note lands in `_inbox/` with `status: inbox`. It is not yet trusted knowledge.

### 3. Validate a note

```
agent-memory lint-note .agent-memory/_inbox/2026-05-21-go-errors-should-wrap-context-at-domain-boundaries.md
```

Lint checks required fields, enum values, date formats, body sections, and
placeholder detection. Pass `--json` for machine-readable output.

### 4. Promote a note

Move a validated note from `_inbox/` to `notes/`:

```
agent-memory promote --slug=go-errors-should-wrap-context-at-domain-boundaries
```

This runs lint and wikilink resolution checks before promoting. Constraint and
decision notes require `--confirmed` (the Librarian asks the human first).

### 5. Deprecate a note

When a note is replaced by a newer one:

```
agent-memory deprecate \
  --slug=go-errors-should-wrap-context-at-domain-boundaries \
  --superseded-by=go-error-wrapping-policy
```

The old note moves to `_deprecated/` with a forward link. Wikilinks to it
still resolve.

### 6. Set up your agent

Pipe the instructions blurb into your agent's configuration:

```
agent-memory instructions >> .claude/AGENTS.md
```

This tells agents how to use the vault, including the Librarian workflow.

## Agent integration

### JSON output

All subcommands support `--json` for machine-readable output. Agents should
always use this flag.

```
agent-memory --json write-note --type observation --title "..." ... body.md
agent-memory --json promote --slug=my-note
agent-memory --json deprecate --slug=old-note --superseded-by=new-note
agent-memory lint-note --json path/to/note.md
```

### Librarian subagent

The Librarian is a dedicated agent that handles the promotion pipeline. It
scans `_inbox/`, lints notes, promotes valid ones, asks the human about
constraint/decision notes, and deprecates superseded notes.

Deployment templates are installed at:
- `.agent-memory/_meta/templates/librarian-agent.md` — agent definition
- `.agent-memory/_meta/templates/librarian-skill.md` — workflow instructions

Copy these to your agent harness configuration (e.g., OpenCode's
`~/.config/opencode/agent/` directory).

### Vault discovery

Subcommands find the vault automatically:
1. `--vault` flag (highest priority)
2. `AGENT_MEMORY_VAULT` environment variable
3. Walk up directories looking for `.agent-memory/`

## Vault structure

```
.agent-memory/
├── _meta/
│   ├── writing-protocol.md      # Note format spec and rules
│   ├── tag-taxonomy.md           # Tag definitions (future)
│   ├── constraints-summary.md    # Active constraints (future)
│   ├── status-lifecycle.md       # Status transitions
│   ├── log.md                    # Append-only audit log
│   └── templates/
│       ├── librarian-agent.md    # Librarian agent definition
│       └── librarian-skill.md    # Librarian workflow
├── _inbox/                       # New notes land here
├── _contested/                   # Contradicted notes (future)
├── _deprecated/                  # Superseded notes with forward links
└── notes/                        # Promoted, verified knowledge
```

## Note format

Every note is a Markdown file with YAML frontmatter:

```yaml
---
title: "Concise factual claim"
created: 2026-05-21
updated: 2026-05-21
status: inbox                  # inbox | verified | deprecated | contested | superseded
confidence: high               # low | medium | high
type: observation              # observation | pattern | constraint | decision | assumption | synthesis
scope: project                 # project | cross-project
project: my-project
domain: [golang, error-handling]
source-agent: my-agent
source-artifact: "pkg/auth/handler.go"
review-by: 2026-08-19
requires-human-review: false
superseded-by: ""
tags: []
---
# Title

## Evidence
...

## Implications
...

## Related
- [[other-note]] — description
```

Synthesis notes use `## Synthesis` and `## Contributing notes` instead of
`## Evidence` and `## Implications`.

## What works today

The core write-promote-deprecate lifecycle is complete. An agent can write
notes, a Librarian can promote or deprecate them, and the vault maintains a
clean audit trail. Specifically:

- **Vault init** with idempotent repair and embedded templates
- **Note writing** with frontmatter assembly, lint validation, similarity
  deduplication, and wikilink extraction
- **Note promotion** with lint gate, wikilink resolution gate, human
  confirmation gate for constraint/decision notes, atomic writes with rollback
- **Note deprecation** with forward links, atomic moves, audit logging
- **Lint** with 7 rules (NF001–NF007): required fields, date formats, enum
  values, conditional fields, body sections, placeholder detection, domain list
- **Wikilink resolution** across `_inbox/`, `notes/`, and `_deprecated/`
  using Obsidian-style filename matching
- **Backward compatibility** for notes using the old `epistemic-type` or
  `update-type` YAML keys
- **Dual output**: human-readable by default, JSON with `--json`

## What doesn't exist yet

These are planned but not implemented:

- **Search** — no `search` subcommand. Agents must read note files directly
  or scan `notes/` themselves. This was deferred because it's unclear whether
  a search command is the right interface for small vaults vs. direct file
  access.
- **Session context** — no `context` subcommand. Agents don't get an
  automatic briefing of relevant notes at session start. They have to read
  `_meta/` and scan directories manually.
- **Index maintenance** — no `reindex` subcommand. There are no auto-generated
  index files per domain. The vault has no table of contents beyond browsing
  the directory.
- **Tagging** — no `tag` subcommand. The Librarian can't auto-tag notes.
  Tags stay empty unless manually set.
- **Pattern detection** — the Librarian doesn't scan for recurring themes
  across observations to suggest pattern notes.
- **Synthesis** — the Librarian doesn't proactively create synthesis notes
  that consolidate related knowledge.
- **Contested notes** — no `contest` subcommand. The `_contested/` directory
  exists but nothing writes to it. There's no workflow for handling
  contradictions between notes.
- **Update/correction** — no `--update` or `--contest` mode on `write-note`.
  To replace a note, you write a new one and manually deprecate the old one.
  There's no staged correction workflow.
- **Backup** — no `backup` subcommand. The vault is plain files, so
  `cp -r` or git works, but there's no built-in mechanism.
- **Semantic search** — no embedding-based retrieval. Search (when built)
  will be frontmatter filtering, not semantic similarity.
- **Staleness scanning** — `review-by` dates are written but nothing checks
  them. No alerts when notes are overdue for review.
- **Constraints summary** — `_meta/constraints-summary.md` exists but is
  never regenerated from constraint notes.

## What works but has known gaps

- **Wikilink resolution** scans directories with `os.ReadDir` per link —
  fine for small vaults, will need indexing for large ones.
- **Similarity check** uses Jaccard on title tokens — catches obvious
  duplicates but misses semantic similarity.
- **Promote/deprecate paths** are returned as absolute paths in JSON output;
  the design doc specifies relative paths.
- **`FindBySlug`** uses string suffix matching for inbox files — works
  because slugs are validated, but could be more precise with date-prefix
  parsing.
- **No concurrent access protection** beyond `O_EXCL` on writes. Two agents
  promoting the same note simultaneously will have one fail cleanly, but
  there's no locking.

## Development

```
# Run unit tests
go test ./internal/... -count=1

# Run integration tests
go test ./test/integration/... -tags=integration -count=1

# Run everything
go test ./... -count=1 && go test ./test/integration/... -tags=integration -count=1

# Vet
go vet ./...
```

Tests use [Ginkgo](https://onsi.github.io/ginkgo/) and
[Gomega](https://onsi.github.io/gomega/) in BDD style.

## Architecture

Single binary with subcommands ([ADR-0001](docs/adr/adr-0001-single-binary-with-subcommands.md)).
Tool owns form, agent owns content. All deterministic work lives in Go
subcommands; LLM judgment stays in the agent layer.

See [AGENT_MEMORY_DESIGN.md](docs/AGENT_MEMORY_DESIGN.md) for the full design
and [ROADMAP.md](docs/ROADMAP.md) for the implementation plan.

## License

TBD
