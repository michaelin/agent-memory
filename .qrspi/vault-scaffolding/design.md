<design_artifact feature="vault-scaffolding">

<current_state>

## Current State

- **Codebase**: Greenfield. No Go code, no `go.mod`, no build files. Only design documents exist.
- **Tooling**: Nothing installed. No mise config, no linters, no test infrastructure.
- **Design docs**: `docs/ROADMAP.md` and `docs/AGENT_MEMORY_DESIGN.md` provide detailed specs. ROADMAP takes precedence where they conflict.

</current_state>

<desired_state>

## Desired End State

- **User-facing**: `agent-memory init [path]` scaffolds a vault. `agent-memory instructions` outputs an agent configuration blurb to stdout for piping into repo instruction files.
- **Technical**: A complete vault directory structure with seeded `_meta/` files. Idempotent re-runs. Repair of missing pieces. JSON output on stdout.
- **Success criteria**: All 10 integration test cases pass. `mise run build`, `mise run test`, `mise run test-integration` all succeed.

</desired_state>

<design_decisions>

## Design Decisions

<decision id="1">

### Project Layout

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

</decision>

<decision id="2">

### CLI Framework: Cobra

De facto standard for Go CLIs with subcommands. The project will grow to many subcommands (`init`, `instructions`, `lint-note`, `lint-vault`, `memory-write`, `memory-search`, etc.). Cobra handles help text, flag parsing, and subcommand dispatch.

**Alternatives rejected**: `urfave/cli` (less common in Go ecosystem), bare `flag` package (too minimal for multi-subcommand binary).

</decision>

<decision id="3">

### Vault Location: git-init Model

```
agent-memory init                  # creates .agent-memory/ in current directory
agent-memory init /path/to/vault   # creates vault at specified path
```

No config file. No resolution chain. The vault is self-describing — its presence at a path *is* the configuration.

**Vault discovery** (for future subcommands, deferred past v1):

1. `AGENT_MEMORY_VAULT` env var → path to vault directory
2. Walk up from current directory looking for `.agent-memory/` (like `git` finds `.git/`)
3. `~/.local/share/agent-memory` → global fallback (XDG default)

For v1, vault path is always explicit: positional arg or default `.agent-memory/` in cwd.

</decision>

<decision id="4">

### Vault Scaffolding Logic

The `init` command executes in order:

1. Determine vault path: positional argument, or `.agent-memory/` in current directory
2. Check if vault path exists:
   - **Doesn't exist**: Create with `os.MkdirAll` (always create parents)
   - **Exists, is a file**: Fail with error
   - **Exists, is a directory**: Check compatibility
3. If vault root is a symlink, follow it. Symlinks inside the vault are not followed.
4. Create/verify required directories: `_meta/`, `_inbox/`, `_contested/`, `notes/`
5. Create/verify required files: all `_meta/` seed files
   - Existing files are left untouched (idempotent — never overwrite user edits)
   - Missing files are created from embedded templates
6. Write log entry to `_meta/log.md`
7. Output JSON result

**Compatibility rule**: A directory is incompatible only if a required vault path (`_meta`, `_inbox`, `_contested`, `notes`) exists as a regular file instead of a directory. Everything else is compatible — scaffold around existing content additively.

**Flags**:
- `--force`: Overwrite conflicting files or folders instead of failing.
- `--clean`: Delete the target vault entirely before initializing. Requires `--force`.

</decision>

<decision id="5">

### Agent Instructions Subcommand

`agent-memory instructions` echoes a configuration blurb to stdout for piping into a repo's agent instructions file:

```
agent-memory instructions >> .claude/AGENTS.md
```

v1 blurb content:
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

The `init` command does not touch the user's repo. Any repo-level agent instruction file is the user's responsibility and depends on which agent harness they use.

</decision>

<decision id="6">

### JSON Output Format

All tool output is JSON to stdout. Stderr is reserved for fatal errors only.

```json
{"status": "created", "vault": "/path/to/vault"}
{"status": "ok", "vault": "/path/to/vault"}
{"status": "repaired", "vault": "/path/to/vault", "repaired": ["_meta/tag-taxonomy.md", "_inbox/"]}
{"error": "vault path /foo/bar exists as a file, not a directory"}
```

Exit codes: 0 on success (created/ok/repaired), 1 on error.

</decision>

<decision id="7">

### Embedded Templates

`go:embed` for all seed templates:

```go
//go:embed templates/*
var templateFS embed.FS
```

Templates are plain markdown files read from the embedded FS and written to disk during init. No network access, no external files.

**Log entry format**: `## [2026-05-11T14:30:00+03:00] init | vault initialized`
Full ISO 8601 with timezone offset.

</decision>

<decision id="8">

### mise Setup

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

mise task syntax needs validation against docs before implementation. Fallback: Just (justfile).

</decision>

<decision id="9">

### Testing Strategy

Two levels:
- **Unit tests**: `_test.go` files in each package. Test vault structure validation, template embedding, compatibility checks.
- **Integration tests**: `test/integration/` directory. Build tag `integration`. BDD-style using ginkgo/gomega — test output serves as a live spec with `Describe/Context/It` blocks that read like UML use cases.

Integration tests use `t.TempDir()` for isolated vault paths.

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

</decision>

</design_decisions>

<data_flow>

## Data Flow

```
User runs `agent-memory init [path]`
  → CLI parses subcommand (cobra)
  → Determine vault path (positional arg or .agent-memory/ in cwd)
  → vault.Init(vaultPath, opts) scaffolds directories and files
  → JSON result to stdout
```

</data_flow>

<open_questions>

## Open Questions

Resolved during design:
- **Config file model**: Eliminated in favor of git-init model. No config file, no resolution chain.
- **AGENTS.md in vault**: Removed. AGENTS.md is repo-level, not vault-level. Added `instructions` subcommand instead.
- **Vault compatibility boundary**: Incompatible only if required path exists as file instead of directory.
- **Config fallback chain**: Replaced with positional arg + default `.agent-memory/` in cwd.

Remaining uncertainties:
- **mise task syntax**: Needs validation. Fallback to Just if needed.
- **writing-protocol.md v1 content**: Needs practical daily-use instructions for agents. Exact content to be defined during Structure/Plan phase.
- **status-lifecycle.md v1 content**: Full table from design doc or simplified version. To be defined during Structure/Plan phase.
- **BDD framework**: ginkgo/gomega preferred, needs confirmation it produces desired output style. goconvey is the alternative.

</open_questions>

</design_artifact>
