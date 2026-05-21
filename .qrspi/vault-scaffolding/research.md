<research_artifact feature="vault-scaffolding">

<codebase_map>

## Codebase Map

The project is currently a greenfield — no Go code, no `go.mod`, no build files exist yet. The repository contains only design documents.

| File | Purpose | Key Contents |
|------|---------|--------------|
| `ROADMAP.md` | Incremental implementation plan | 12 increments; Increment 1 defines vault scaffolding scope, config resolution chain, vault directory structure, seed templates, idempotency rules, integration tests |
| `AGENT_MEMORY_DESIGN.md` | Full architectural design | Epistemic types, note format, memory tiers, write protocol, Librarian agent, static-tooling principle, vault structure, staleness prevention |
| `DELIVERABLES.md` | Deliverables checklist | Lists expected outputs per increment |
| `INCREMENTS-DRAFT.md` | Draft increment breakdown | Earlier draft of what became ROADMAP.md |
| `INDEX.md` | Document index | Navigation file for design docs |
| `REVIEW-SUMMARY.md` | Review notes | Summary of design review feedback |
| `ARCHITECTURAL-DECISION-SKILL-VS-SUBAGENT.md` | ADR | Decision on skill vs subagent architecture |

**Module relationships:** None yet — no Go modules exist.

</codebase_map>

<question_answers>

## Question Findings

### Project Setup

<finding question="What Go module path should this project use?">

**What exists:** No `go.mod` file exists. The repository is at `michaelin/agent-memory` on the local filesystem.

**Where:** Repository root — no Go files present.

**Decision recorded:** `github.com/michaelin/agent-memory` — confirmed by user.

</finding>

<finding question="What minimum Go version should the project target, and should it use Go workspaces or a single module?">

**What exists:** No Go version constraint exists anywhere in the project.

**Where:** N/A — greenfield.

**Decision recorded:** Single module. Latest stable Go version. No workspaces.

</finding>

<finding question="What is the intended distribution mechanism for the agent-memory binary?">

**What exists:** No build or distribution configuration exists.

**Where:** N/A.

**Decision recorded:** Must be mise-compatible. `go install` is acceptable if it works with mise. The user uses mise for installing utilities and managing dev environments.

</finding>

<finding question="Should the project use a Makefile, Taskfile, or just go commands for build/test/lint?">

**What exists:** No build system exists.

**Where:** N/A.

**Decision recorded:** Use mise tasks as the primary task runner. If mise tasks prove insufficient, fall back to Just (justfile). A local AGENTS.md or skill must document mise task usage since it is a new capability for the user. All relevant tooling (Go, linters, etc.) must be installed locally in the project via mise (`.mise.toml`).

</finding>

### Data Model

<finding question="What is the exact JSON schema for config.json?">

**What exists:** ROADMAP.md §Increment 1 specifies:
```json
{"vault": "/path/to/vault"}
```

AGENT_MEMORY_DESIGN.md §5.1 specifies the same shape at a different path (`~/.agent-memory.json`).

**Where:** `ROADMAP.md` lines 26-29, `AGENT_MEMORY_DESIGN.md` lines 584-589.

**Decision recorded:** No version field needed. This is an internal interface only. Minimal schema: `{"vault": "/path/to/vault"}`.

</finding>

<finding question="Config location — ROADMAP vs design doc. Clean break or fallback?">

**What exists:** Two conflicting specifications:
- ROADMAP.md: `~/.config/agent-memory/config.json` (XDG convention)
- AGENT_MEMORY_DESIGN.md: `~/.agent-memory.json` (home directory convention)

ROADMAP.md also specifies env var `AGENT_MEMORY_VAULT` as highest priority and `$HOME/.local/share/agent-memory` as default vault location (XDG_DATA_HOME).

**Where:** `ROADMAP.md` lines 22-24, `AGENT_MEMORY_DESIGN.md` lines 581-607.

**Decision recorded:** ROADMAP.md takes precedence. Config resolution chain:
1. `AGENT_MEMORY_VAULT` env var (highest priority)
2. `XDG_CONFIG_HOME/agent-memory/config.json` (if XDG_CONFIG_HOME is set)
3. `~/.config/agent-memory/config.json` (default XDG)
4. `~/agent-memory/config.json` (home directory fallback)

The design doc's `~/.agent-memory.json` location is not supported.

</finding>

<finding question="What are the exact contents of each seed template file?">

**What exists:** ROADMAP.md specifies content for `writing-protocol.md` only:
> "This vault is for agent memory. Writing is not yet enabled. See AGENTS.md for current capabilities."

For other files, ROADMAP.md says:
- `tag-taxonomy.md` — "Empty in v1"
- `constraints-summary.md` — "Empty in v1"
- `status-lifecycle.md` — "Lifecycle stages (static template)"
- `log.md` — "Append-only log (empty at init)"

AGENT_MEMORY_DESIGN.md §5.6 defines the status lifecycle stages and TTL table. §5.3 defines the full note format. §5.2 defines the vault structure.

**Where:** `ROADMAP.md` lines 37-54, `AGENT_MEMORY_DESIGN.md` lines 619-639 (structure), 656-723 (note format), 859-874 (TTL table).

**Decision recorded:** Only seed what makes sense for the current iteration. Empty files where ROADMAP says "empty in v1". `status-lifecycle.md` gets the lifecycle stages from the design doc since it's a static reference. `writing-protocol.md` gets the v1 content from ROADMAP.

</finding>

<finding question="Should log.md use UTC or local time? Should it include timezone info?">

**What exists:** ROADMAP.md specifies format: `## [YYYY-MM-DD HH:MM] init | vault initialized`. No timezone specification.

**Where:** `ROADMAP.md` line 54.

**Decision recorded:** Local time in ISO format with timezone offset (e.g., `2026-05-11T14:30:00+03:00`).

</finding>

### CLI API

<finding question="What is the CLI interface for agent-memory init?">

**What exists:** ROADMAP.md describes `agent-memory init` behavior but does not specify flags. AGENT_MEMORY_DESIGN.md §9 references `memory-write`, `memory-search`, etc. as separate subcommands of a single binary pattern.

**Where:** `ROADMAP.md` lines 32-54.

**Decision recorded:** Keep it simple. No `--vault-path` or `--config` flags for now. Read from config file, fall back to env vars, fall back to default values. Flags can be added later.

</finding>

<finding question="Should all agent-memory tools be subcommands of a single binary or separate binaries?">

**What exists:** ROADMAP.md references `agent-memory init` as a subcommand. AGENT_MEMORY_DESIGN.md §8 references `agent-memory init`, `lint-note`, `lint-vault` — mixing subcommand and separate binary patterns.

**Where:** `ROADMAP.md` line 32, `AGENT_MEMORY_DESIGN.md` lines 1033-1035.

**Decision recorded:** Single `agent-memory` binary with subcommands (`init`, `lint-note`, `lint-vault`, etc.).

</finding>

<finding question="What should agent-memory init output on success/repair/no-op?">

**What exists:** ROADMAP.md specifies exit codes (exit 0 on idempotent re-run) but not output format. AGENT_MEMORY_DESIGN.md mentions `lint-note` uses JSON output: `{"valid": true}`.

**Where:** `ROADMAP.md` lines 33-36, `AGENT_MEMORY_DESIGN.md` (lint-note JSON pattern).

**Decision recorded:** JSON-only output for now. Machine-parseable. Can be extended with human-readable output later.

</finding>

<finding question="Should agent-memory init write the config file automatically?">

**What exists:** ROADMAP.md line 53: "Write configuration file to `~/.config/agent-memory/config.json` (or env var location)". This implies init always writes config.

**Where:** `ROADMAP.md` line 53.

**Decision recorded:** Yes, init writes/updates the config file. If overrides are provided (e.g., via env var specifying a non-default vault path), the config file is updated to persist that choice.

</finding>

### Integration

<finding question="How will agents discover and invoke the agent-memory binary?">

**What exists:** No discovery mechanism is defined in either document. AGENT_MEMORY_DESIGN.md assumes tools are available as tool calls.

**Where:** N/A.

**Decision recorded:** Binary must be on `$PATH`. Deployment (dev and prod) must be mise-compatible. `go install` is acceptable.

</finding>

<finding question="Should templates be embedded in the binary or read from filesystem?">

**What exists:** ROADMAP.md line 51: "Seed `_meta/` files from embedded templates (no network, no external files)". This explicitly specifies `go:embed`.

**Where:** `ROADMAP.md` line 51.

**Decision recorded:** Embed templates in the binary using `go:embed`. Go rebuilds are fast.

</finding>

<finding question="Should agent-memory init create parent directories if they don't exist?">

**What exists:** No specification in either document.

**Where:** N/A.

**Decision recorded:** Non-interactive. Always create parent directories (`MkdirAll`). No prompting.

</finding>

### Edge Cases

<finding question="What happens if init is run on an existing non-vault directory?">

**What exists:** ROADMAP.md lines 33-36 describe idempotency for existing vaults but not for non-vault directories.

**Where:** `ROADMAP.md` lines 33-36.

**Decision recorded:** Fail if the existing directory structure is not compatible with a vault. Never overwrite. The tool checks for incompatible existing files/directories and exits with an error.

</finding>

<finding question="What happens if the config file contains invalid JSON or invalid vault path?">

**What exists:** No error handling specification in either document for config parsing failures.

**Where:** N/A.

**Decision recorded:** Fail with a clear error. User must manually update or delete the config file. The tool does not attempt to fix invalid config.

</finding>

<finding question="What filesystem permissions should vault directories and files be created with?">

**What exists:** AGENT_MEMORY_DESIGN.md §5.1 notes the vault may contain "observations about client codebases" and should not be committed to any remote.

**Where:** `AGENT_MEMORY_DESIGN.md` lines 609-613.

**Decision recorded:** Respect umask (Go default). During development, this is local only. No secrets are expected in the vault — a future iteration must handle detection and scrubbing of secrets. Secrets belong in a secret manager of the user's choice.

**New requirement surfaced:** A future increment must address secret detection and scrubbing. Secrets do not belong in the vault.

</finding>

<finding question="How should agent-memory init handle symlinks?">

**What exists:** No symlink policy in either document.

**Where:** N/A.

**Decision recorded:** The vault root path may be a symlink (follow it). Symlinks inside the vault structure are not followed. This allows the vault to be abstracted by a symlink (e.g., pointing to a synced folder) while keeping the internal structure predictable.

</finding>

<finding question="Should the vault have a version marker file or field?">

**What exists:** ROADMAP.md line 36: "If vault exists but is from an older version, upgrade it (no-op for v1)". This implies versioning is anticipated but not specified.

**Where:** `ROADMAP.md` line 36.

**Decision recorded:** No version marker yet. May be added in a future increment. Keep it simple during development.

</finding>

</question_answers>

<patterns>

## Existing Patterns and Conventions

- **Static-tooling principle:** AGENT_MEMORY_DESIGN.md §5.0 establishes that anything structural, mechanical, or rule-based lives in Go tooling. Agents own content; tools own form.
- **XDG conventions:** ROADMAP.md follows XDG Base Directory Specification for config (`XDG_CONFIG_HOME`) and data (`XDG_DATA_HOME`) paths.
- **JSON output for tools:** AGENT_MEMORY_DESIGN.md establishes JSON as the tool output format (e.g., `lint-note` returns `{"valid": true}`). All tool output in Increment 1 follows this pattern.
- **Idempotency:** ROADMAP.md explicitly requires `agent-memory init` to be idempotent — safe to run multiple times with the same result.
- **Append-only log:** `_meta/log.md` is append-only, written by tools, never by agents directly.
- **Embedded templates:** Seed files are embedded in the binary via `go:embed` — no external file dependencies.
- **mise for tooling:** The project uses mise (`.mise.toml`) for Go version management, dev tool installation, and task running. This replaces Makefile/Taskfile/Devbox/Devcontainers.

</patterns>

<gaps>

## Gaps and Missing Capabilities

- **No Go code exists:** The project is entirely design documents. No `go.mod`, no source files, no tests, no build configuration.
- **No `.mise.toml`:** The mise configuration for Go version, dev tools, and tasks does not exist yet.
- **No AGENTS.md for mise usage:** The user requires documentation on mise task usage since it's new to them. No such documentation exists.
- **Secret detection/scrubbing:** Neither document addresses what happens if an agent writes sensitive content (API keys, credentials, tokens) into a vault note. A future increment must handle this. Secrets do not belong in the vault.
- **`status-lifecycle.md` content:** The design doc defines lifecycle stages and TTL table, but no ready-to-embed template exists for this file.
- **Config resolution chain detail:** The exact fallback behavior when multiple config sources partially exist (e.g., env var set but config file also exists with a different path) is not specified. The priority order is clear but edge case interactions are not.
- **Vault compatibility check:** The criteria for determining whether an existing directory is "compatible" with a vault (Q16) are not defined. What specific checks determine compatibility vs. incompatibility?

</gaps>

</research_artifact>
