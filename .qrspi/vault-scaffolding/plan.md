<plan_artifact feature="vault-scaffolding">

<objective>

## Objective

Implement the `agent-memory` CLI with `init` and `instructions` subcommands, producing a fully scaffolded vault directory with embedded seed templates. The binary is buildable via mise, tested with ginkgo/gomega BDD integration tests, and outputs JSON to stdout.

**Constrained by:**
- Design decisions in `.qrspi/vault-scaffolding/design.md`
- Structure outline in `.qrspi/vault-scaffolding/structure.md`

</objective>

<slices>

<slice name="slice-1-project-skeleton">

## Slice 1: Project Skeleton

**Goal**: Minimal buildable Go project with mise tasks. `go build` succeeds, binary runs.

<tasks>

<task>
**Name**: Task 1.1 — Initialize Go module
**Files**: `go.mod`
**Action**: Run `go mod init github.com/michaelin/agent-memory`. Module path follows standard GitHub convention.
**Verify**: `cat go.mod` shows module declaration
**Done**: `go.mod` exists with correct module path
</task>

<task>
**Name**: Task 1.2 — Create mise config
**Files**: `.mise.toml`
**Action**: Per design decision #8. Define Go version, golangci-lint, and tasks for build, test, lint, test-integration. Validate mise task syntax against mise docs — if `[tasks.X] run = "..."` doesn't work, use the correct syntax.
**Verify**: `mise run build` succeeds (after task 1.4)
**Done**: `.mise.toml` exists, `mise install` installs Go and golangci-lint
</task>

<task>
**Name**: Task 1.3 — Create root cobra command
**Files**: `internal/cli/root.go`
**Action**: Per design decision #2. Create `NewRootCmd()` returning a `*cobra.Command` with `Use: "agent-memory"`, a short description, and a `--version` flag. No subcommands yet.
**Verify**: `go build ./...` compiles
**Done**: `NewRootCmd()` returns a valid cobra command
</task>

<task>
**Name**: Task 1.4 — Create main entry point
**Files**: `cmd/agent-memory/main.go`
**Action**: Per design decision #1. Import `internal/cli`, call `cli.NewRootCmd().Execute()`. Handle exit code on error.
**Verify**: `mise run build && ./bin/agent-memory --version`
**Done**: Binary builds and prints version string
</task>

<task>
**Name**: Task 1.5 — Fetch dependencies
**Files**: `go.sum`
**Action**: Run `go mod tidy` to resolve cobra dependency.
**Verify**: `go build ./...` succeeds with no missing deps
**Done**: `go.sum` exists, build is clean
</task>

</tasks>

<checkpoint>
**Slice 1 Checkpoint**:
- [ ] `mise run build` succeeds
- [ ] `./bin/agent-memory --version` prints version
- [ ] `go vet ./...` passes
</checkpoint>

</slice>

<slice name="slice-2-vault-structure">

## Slice 2: Vault Structure Definition

**Goal**: Vault package defines the complete vault structure and embeds all seed templates.

<tasks>

<task>
**Name**: Task 2.1 — Define vault structure types and entries
**Files**: `internal/vault/structure.go`
**Action**: Per structure type definitions. Define `VaultEntry` struct. Define `VaultStructure()` returning the ordered list of entries: directories (`_meta/`, `_inbox/`, `_contested/`, `notes/`) and files (`_meta/writing-protocol.md`, `_meta/tag-taxonomy.md`, `_meta/constraints-summary.md`, `_meta/status-lifecycle.md`, `_meta/log.md`). Embed templates with `//go:embed templates/*`.
**Verify**: `go build ./internal/vault/`
**Done**: `VaultStructure()` compiles and returns all expected entries
</task>

<task>
**Name**: Task 2.2 — Create seed template files
**Files**: `internal/vault/templates/writing-protocol.md`, `internal/vault/templates/tag-taxonomy.md`, `internal/vault/templates/constraints-summary.md`, `internal/vault/templates/status-lifecycle.md`, `internal/vault/templates/log.md`
**Action**: Per ROADMAP.md: `writing-protocol.md` gets minimal v1 content (vault purpose, directory meanings, basic rules — useful even before writing is enabled). `tag-taxonomy.md` and `constraints-summary.md` are stubs with a header and placeholder text. `status-lifecycle.md` gets the lifecycle stages table from AGENT_MEMORY_DESIGN.md §5.6. `log.md` is an empty file (header only — entries are appended by init).
**Verify**: `go test ./internal/vault/ -run TestTemplatesReadable`
**Done**: All template files exist and are readable from embedded FS
</task>

<task>
**Name**: Task 2.3 — Unit tests for vault structure
**Files**: `internal/vault/structure_test.go`
**Action**: Test that `VaultStructure()` returns the expected number of entries, all directories come before their child files, and all template files referenced by file entries are readable from the embedded FS.
**Verify**: `go test ./internal/vault/`
**Done**: All unit tests pass
</task>

</tasks>

<checkpoint>
**Slice 2 Checkpoint**:
- [ ] `go test ./internal/vault/` passes
- [ ] All 5 template files embedded and readable
- [ ] `VaultStructure()` returns correct entries
</checkpoint>

</slice>

<slice name="slice-3-init-core">

## Slice 3: Init Core

**Goal**: `agent-memory init [path]` creates a fresh vault with all directories and seed files. JSON output to stdout.

<tasks>

<task>
**Name**: Task 3.1 — Implement vault.Init for fresh creation
**Files**: `internal/vault/init.go`
**Action**: Per design decisions #3, #4, #6, #7. Define `InitOptions`, `InitResult`, `InitError` types. Implement `Init(vaultPath string, opts InitOptions) (*InitResult, error)`: resolve to absolute path, create vault root with `os.MkdirAll`, iterate `VaultStructure()` creating dirs and writing files from embedded templates, append log entry to `_meta/log.md` per decision #7 format, return `InitResult{Status: "created", Vault: absPath}`. For now, only handle the fresh-creation path (no existing vault).
**Verify**: Unit test calling `Init()` on a temp dir
**Done**: `Init()` creates complete vault structure, returns "created" status
</task>

<task>
**Name**: Task 3.2 — Wire init subcommand to CLI
**Files**: `internal/cli/init.go`
**Action**: Per design decisions #1, #6. Create `newInitCmd()` returning a cobra command for `init`. Accept optional positional arg (vault path, default `.agent-memory/` in cwd). Call `vault.Init()`, marshal result to JSON, print to stdout. On error, marshal `InitError` to JSON, print to stdout, exit 1.
**Verify**: `mise run build && ./bin/agent-memory init /tmp/test-vault-$RANDOM`
**Done**: Binary creates vault at specified path, outputs JSON with status "created"
</task>

<task>
**Name**: Task 3.3 — Unit tests for init core
**Files**: `internal/vault/init_test.go`
**Action**: Test fresh init: all dirs exist, all files exist with correct content, log entry written, result status is "created". Test default path (no arg → `.agent-memory/` in cwd).
**Verify**: `go test ./internal/vault/`
**Done**: All unit tests pass
</task>

</tasks>

<checkpoint>
**Slice 3 Checkpoint**:
- [ ] `mise run build` succeeds
- [ ] `./bin/agent-memory init /tmp/test-vault` outputs `{"status":"created","vault":"/tmp/test-vault"}`
- [ ] Vault contains `_meta/`, `_inbox/`, `_contested/`, `notes/`, and all seed files
- [ ] `_meta/log.md` contains init log entry
- [ ] `go test ./internal/vault/` passes
</checkpoint>

</slice>

<slice name="slice-4-idempotency-and-repair">

## Slice 4: Idempotency and Repair

**Goal**: Double init is a no-op. Missing pieces are repaired. Incompatible paths produce errors.

<tasks>

<task>
**Name**: Task 4.1 — Add idempotency and repair logic to Init
**Files**: `internal/vault/init.go`
**Action**: Per design decision #4. Extend `Init()`: if vault path exists as a file → error. If vault path exists as a directory → check each `VaultEntry`: if dir entry exists as file → error (incompatible). If file entry exists → skip (idempotent). If entry missing → create it and track in `repaired` list. If nothing was created → status "ok". If something was repaired → status "repaired" with list. Follow symlinks at vault root only (per decision #4 step 3).
**Verify**: Unit tests for all three paths
**Done**: Double init returns "ok", missing file returns "repaired", file-as-dir returns error
</task>

<task>
**Name**: Task 4.2 — Unit tests for idempotency and repair
**Files**: `internal/vault/init_test.go`
**Action**: Add tests: (1) init twice → second returns "ok", no files changed. (2) Delete a seed file, re-init → "repaired" with file in list. (3) Delete a directory, re-init → "repaired". (4) Create file at `_meta` path → error. (5) Vault path is a file → error.
**Verify**: `go test ./internal/vault/`
**Done**: All idempotency/repair tests pass
</task>

</tasks>

<checkpoint>
**Slice 4 Checkpoint**:
- [ ] Double init returns `{"status":"ok"}`
- [ ] Repair returns `{"status":"repaired","repaired":[...]}`
- [ ] Incompatible path returns `{"error":"..."}`
- [ ] `go test ./internal/vault/` passes
</checkpoint>

</slice>

<slice name="slice-5-force-and-clean">

## Slice 5: Force and Clean Flags

**Goal**: `--force` overwrites conflicts. `--clean --force` resets vault. `--clean` alone errors.

<tasks>

<task>
**Name**: Task 5.1 — Add force and clean logic to Init
**Files**: `internal/vault/init.go`
**Action**: Per design decision #4. If `opts.Clean && !opts.Force` → error ("--clean requires --force"). If `opts.Clean && opts.Force` → `os.RemoveAll(vaultPath)` then proceed with fresh init. If `opts.Force` (without clean) → when encountering a conflict (file where dir expected, or dir where file expected), remove the conflicting entry and create the correct one.
**Verify**: Unit tests
**Done**: Force and clean logic works for all combinations
</task>

<task>
**Name**: Task 5.2 — Wire flags to CLI
**Files**: `internal/cli/init.go`
**Action**: Add `--force` and `--clean` bool flags to the init command. Pass through to `vault.InitOptions`.
**Verify**: `./bin/agent-memory init --help` shows both flags
**Done**: Flags appear in help and are passed to `vault.Init()`
</task>

<task>
**Name**: Task 5.3 — Unit tests for force and clean
**Files**: `internal/vault/init_test.go`
**Action**: Add tests: (1) `--force` with file-as-dir conflict → overwrites, succeeds. (2) `--clean --force` on existing vault → vault deleted and recreated, status "created". (3) `--clean` without `--force` → error.
**Verify**: `go test ./internal/vault/`
**Done**: All force/clean tests pass
</task>

</tasks>

<checkpoint>
**Slice 5 Checkpoint**:
- [ ] `--force` resolves conflicts
- [ ] `--clean --force` resets vault
- [ ] `--clean` alone errors
- [ ] `go test ./internal/vault/` passes
</checkpoint>

</slice>

<slice name="slice-6-instructions">

## Slice 6: Instructions Subcommand

**Goal**: `agent-memory instructions` outputs agent configuration blurb to stdout.

<tasks>

<task>
**Name**: Task 6.1 — Implement instructions subcommand
**Files**: `internal/cli/instructions.go`
**Action**: Per design decision #5. Create `newInstructionsCmd()` returning a cobra command. The command prints the agent blurb (hardcoded string matching the exact content from design decision #5) to stdout. No flags, no args.
**Verify**: `mise run build && ./bin/agent-memory instructions`
**Done**: Output matches the blurb from design decision #5 exactly
</task>

</tasks>

<checkpoint>
**Slice 6 Checkpoint**:
- [ ] `./bin/agent-memory instructions` outputs the agent blurb
- [ ] Output starts with `# Agent Memory`
- [ ] Output ends with the "Writing to the vault is not yet enabled" line
</checkpoint>

</slice>

<slice name="slice-7-integration-tests">

## Slice 7: Integration Tests

**Goal**: BDD integration test suite covering all 10 test cases from design. Ginkgo/gomega with build tag `integration`.

<tasks>

<task>
**Name**: Task 7.1 — Set up ginkgo test suite
**Files**: `test/integration/integration_suite_test.go`
**Action**: Per design decision #9. Bootstrap ginkgo suite with `//go:build integration` build tag. Suite builds the `agent-memory` binary to a temp location in `BeforeSuite` and cleans up in `AfterSuite`.
**Verify**: `go test -tags=integration ./test/integration/... -list '.*'` lists tests
**Done**: Suite bootstraps without error
</task>

<task>
**Name**: Task 7.2 — Create test helpers
**Files**: `internal/testutil/helpers.go`
**Action**: Shared helpers: `BuildBinary(t) string` (builds binary, returns path), `RunBinary(binPath string, args ...string) (stdout, stderr string, exitCode int)` (runs binary, captures output), `TempDir(t) string` (creates temp dir).
**Verify**: `go build ./internal/testutil/`
**Done**: Helpers compile
</task>

<task>
**Name**: Task 7.3 — Write all 10 integration test cases
**Files**: `test/integration/init_test.go`
**Action**: Per design decision #9. BDD-style Describe/Context/It blocks:
1. `Describe("agent-memory init")` / `Context("with no existing vault")` / `It("creates vault in cwd")` — run init with no args, verify `.agent-memory/` created with all dirs/files
2. `It("creates vault at specified path")` — run init with path arg
3. `Context("with existing vault")` / `It("is idempotent")` — double init, verify "ok" status
4. `It("repairs missing directories")` — delete dir, re-init, verify "repaired"
5. `Context("with incompatible path")` / `It("errors on file conflict")` — create file at dir path, verify error
6. `Context("with --force")` / `It("overwrites conflicts")` — force with conflict, verify success
7. `Context("with --clean --force")` / `It("resets vault")` — clean+force, verify fresh vault
8. `Context("with --clean without --force")` / `It("errors")` — verify error
9. `Describe("log entry")` / `It("writes init log with ISO 8601 timestamp")` — verify `_meta/log.md` content
10. `Describe("agent-memory instructions")` / `It("outputs agent blurb")` — verify stdout content
**Verify**: `mise run test-integration`
**Done**: All 10 test cases pass
</task>

</tasks>

<checkpoint>
**Slice 7 Checkpoint**:
- [ ] `mise run test-integration` passes all 10 tests
- [ ] Test output reads like a spec (Describe/Context/It structure)
- [ ] No test pollution between cases (each uses isolated temp dir)
</checkpoint>

</slice>

</slices>

<verification>

## Final Verification

Before declaring implementation complete:
- [ ] `mise run build` succeeds
- [ ] `mise run test` passes (unit tests)
- [ ] `mise run test-integration` passes (all 10 integration tests)
- [ ] `mise run lint` passes
- [ ] `./bin/agent-memory init` creates a complete vault
- [ ] `./bin/agent-memory instructions` outputs the agent blurb
- [ ] JSON output is valid and matches design decision #6 format

</verification>

<success_criteria>

## Success Criteria

- [ ] `agent-memory init [path]` creates a complete vault with all directories and seed files
- [ ] `agent-memory init` is idempotent and repairs missing pieces
- [ ] `--force` and `--clean` flags work as specified
- [ ] `agent-memory instructions` outputs the agent blurb
- [ ] All output is JSON to stdout, stderr for fatal errors only
- [ ] All 10 integration tests pass
- [ ] All unit tests pass
- [ ] Lint passes

</success_criteria>

</plan_artifact>
