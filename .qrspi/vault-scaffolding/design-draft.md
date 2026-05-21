# Design Draft: vault-scaffolding

## Current State

- **Codebase**: Greenfield. No Go code, no `go.mod`, no build files. Only design documents exist.
- **Tooling**: Nothing installed. No mise config, no linters, no test infrastructure.
- **Design docs**: `docs/ROADMAP.md` and `docs/AGENT_MEMORY_DESIGN.md` provide detailed specs. ROADMAP takes precedence where they conflict.

## Desired End State

After this increment:
- `agent-memory init` is a working Go binary that scaffolds a vault
- `agent-memory instructions` outputs an agent configuration blurb to stdout
- Running `init` produces a complete vault directory structure with seeded `_meta/` files
- Running `init` again on the same vault is a safe no-op (idempotent)
- Running `init` on a vault with missing pieces repairs them
- All tool output is JSON to stdout
- The project is set up with mise for Go version management, dev tools, and task running
- BDD-style integration tests verify all behaviors and serve as a live spec

## Design Decisions

### Decision 1: Project layout

Standard Go CLI project layout:

```
agent-memory/
├── .mise.toml              # Go version, tools, tasks
├── go.mod
├── go.sum
├── cmd/
│   └── agent-memory/
│       └── main.go         # Entry point
├── internal/
│   ├── vault/
│   │   ├── init.go         # Vault scaffolding, repair, idempotency
│   │   ├── structure.go    # Vault structure definition (dirs, files)
│   │   └── templates/      # Embedded template files
│   │       ├── writing-protocol.md
│   │       ├── tag-taxonomy.md
│   │       ├── constraints-summary.md
│   │       └── status-lifecycle.md
│   └── cli/
│       ├── root.go         # Root command setup (cobra)
│       ├── init.go         # `init` subcommand handler
│       └── instructions.go # `instructions` subcommand handler
├── internal/testutil/      # Shared test helpers (temp dirs, etc.)
└── test/
    └── integration/
        └── init_test.go    # BDD integration tests
```

**Rationale**: `cmd/` for the binary entry point, `internal/` for non-exported packages. Separating `vault` and `cli` keeps concerns clean. Templates live alongside the vault package that embeds them.

### Decision 2: CLI framework

Cobra for CLI subcommand routing.

**Rationale**: De facto standard for Go CLIs with subcommands. The project will grow to have many subcommands (`init`, `instructions`, `lint-note`, `lint-vault`, `memory-write`, `memory-search`, etc.). Cobra handles help text, flag parsing, and subcommand dispatch.

**Alternative considered**: `urfave/cli` — also solid, but cobra is more common in the Go ecosystem. Bare `flag` package — too minimal for a multi-subcommand binary.

### Decision 3: Vault location (git-init model)

`agent-memory init` works like `git init`:

```
agent-memory init                  # creates .agent-memory/ in current directory
agent-memory init /path/to/vault   # creates vault at specified path
```

No config file. No resolution chain. The vault is self-describing — its presence at a path *is* the configuration.

**Vault discovery** (for future subcommands that need to find an existing vault):

1. `AGENT_MEMORY_VAULT` env var → path to vault directory
2. Walk up from current directory looking for `.agent-memory/` (like `git` finds `.git/`)
3. `~/.local/share/agent-memory` → global fallback (XDG default)

This gives you:
- **Project-specific vaults**: `agent-memory init` in a repo root → `.agent-memory/` lives in the repo
- **Global vaults**: `agent-memory init ~/.local/share/agent-memory`
- **Env var override**: for CI or non-standard setups

**Note**: Walk-up discovery (step 2) may be deferred past v1 since Increment 1 only has `init` and `instructions`. Discovery matters more when `memory-write`, `memory-search`, etc. arrive. For v1, vault path is always explicit (positional arg or default `.agent-memory/` in cwd).

### Decision 4: Vault scaffolding logic

The `init` command does this in order:

1. Determine vault path: positional argument, or `.agent-memory/` in current directory
2. Check if vault path exists:
   - **Doesn't exist**: Create it with `os.MkdirAll` (always create parents)
   - **Exists, is a file**: Fail with error
   - **Exists, is a directory**: Check compatibility (see rule below)
3. If vault root is a symlink, follow it. Symlinks inside the vault are not followed.
4. Create/verify required directories: `_meta/`, `_inbox/`, `_contested/`, `notes/`
5. Create/verify required files: all `_meta/` seed files
   - If a file already exists, leave it untouched (idempotent — don't overwrite user edits)
   - If a file is missing, create it from embedded template
6. Write log entry to `_meta/log.md`
7. Output JSON result

**Vault compatibility rule**: A directory is incompatible only if a required vault path (e.g., `_meta`, `_inbox`, `notes`) exists as a regular file instead of a directory. Everything else is compatible — we scaffold around existing content additively.

**`--force` flag**: Overwrite any existing conflicting files or folders instead of failing.

**`--clean` flag**: Delete the target vault directory entirely before initializing. Requires `--force` (safety guard). Useful for resetting a vault to a clean state.

### Decision 5: Agent instructions subcommand

`AGENTS.md` is a repo-level instruction file for the agent harness (Claude Code, Cursor, etc.) — it belongs in the user's project repo, not in the vault. The `init` command scaffolds the vault only and does not touch the user's repo.

`agent-memory instructions` echoes a configuration blurb to stdout that can be piped into a repo's agent instructions file (like shell completion commands):

```
agent-memory instructions >> .claude/AGENTS.md
```

The blurb content for v1:
```
# Agent Memory

This project uses agent-memory for persistent knowledge storage.

## Vault Location

The vault is at `.agent-memory/` in this repository (or set `AGENT_MEMORY_VAULT`
to point elsewhere).

## Usage

Before starting work, check the vault for relevant context:
1. Read `_meta/writing-protocol.md` in the vault for rules and conventions
2. Check `_inbox/` for unprocessed notes
3. Check `notes/` for existing knowledge

Writing to the vault is not yet enabled in this version.
```

### Decision 6: JSON output format

All tool output is JSON to stdout. Stderr is reserved for fatal errors only.

```json
// Success (fresh init)
{"status": "created", "vault": "/path/to/vault"}

// Success (idempotent, no changes needed)
{"status": "ok", "vault": "/path/to/vault"}

// Success (repaired missing structure)
{"status": "repaired", "vault": "/path/to/vault", "repaired": ["_meta/tag-taxonomy.md", "_inbox/"]}

// Error
{"error": "vault path /foo/bar exists as a file, not a directory"}
```

Exit codes: 0 on success (created/ok/repaired), 1 on error.

### Decision 7: Embedded templates

Using `go:embed` to embed all seed templates:

```go
//go:embed templates/*
var templateFS embed.FS
```

Templates are plain markdown files. The vault package reads them from the embedded FS and writes them to disk during init. No network access, no external files.

**Log entry format**: `## [2026-05-11T14:30:00+03:00] init | vault initialized`

Full ISO 8601 with timezone offset as decided in research.

### Decision 8: mise setup

`.mise.toml` at project root:

```toml
[tools]
go = "latest"
golangci-lint = "latest"

[tasks.build]
run = "go build -o bin/agent-memory ./cmd/agent-memory"

[tasks.test]
run = "go test ./..."

[tasks.lint]
run = "golangci-lint run"

[tasks.test-integration]
run = "go test -tags=integration ./test/integration/..."
```

**Uncertainty**: mise task runner syntax needs validation against actual mise docs before implementation. If mise tasks don't work well, fall back to Just (justfile).

### Decision 9: Testing strategy

Two levels:
- **Unit tests**: `_test.go` files in each package. Test vault structure validation, template embedding, compatibility checks.
- **Integration tests**: `test/integration/` directory. Build tag `integration`. Test the full `agent-memory init` binary end-to-end.

**Style**: BDD-style integration tests that read like UML use cases — test output serves as a live spec. Using **ginkgo/gomega** for expressive `Describe/Context/It` blocks.

Integration tests use `t.TempDir()` for isolated vault paths. Environment variables are set/unset per test where needed.

Test cases:
1. Fresh init (no args) → `.agent-memory/` created in cwd, all dirs/files present
2. Fresh init with path → vault created at specified path
3. Double init → idempotent, no errors, no changes
4. Missing directory in existing vault → repaired
5. Existing non-vault directory with conflicting file → error
6. `--force` with conflicting file → overwrites, succeeds
7. `--clean --force` → deletes vault, reinitializes
8. `--clean` without `--force` → error
9. Log entry → written with correct format and timestamp
10. `agent-memory instructions` → outputs agent blurb to stdout

## Data Flow

```
User runs `agent-memory init [path]`
  → CLI parses subcommand (cobra)
  → Determine vault path (positional arg or .agent-memory/ in cwd)
  → vault.Init(vaultPath, opts) scaffolds directories and files
  → JSON result to stdout
```

## Open Questions

1. **mise task syntax**: Needs validation against actual mise documentation. Fallback: Just.
2. **writing-protocol.md content**: Primary agent-facing file in the vault. Needs practical daily-use instructions: what the vault is, how to read from it, what the directories mean, what the rules are. Should be useful even before writing is enabled. Exact v1 content TBD.
3. **status-lifecycle.md content**: The design doc defines lifecycle stages and TTL table. Should the v1 seed contain the full table from AGENT_MEMORY_DESIGN.md §5.6, or a simplified version?
4. **Walk-up vault discovery**: Include in v1 or defer? Only `init` and `instructions` exist in Increment 1 — discovery matters more for later subcommands.
5. **BDD framework**: Leaning ginkgo/gomega, but need to confirm it produces the UML-use-case-style output desired. goconvey is an alternative.
