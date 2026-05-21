<structure_artifact feature="vault-scaffolding">

<type_definitions>

## Type Definitions

```go
// InitOptions — controls vault initialization behavior
type InitOptions struct {
    Force bool // overwrite conflicting files/folders
    Clean bool // delete vault before init (requires Force)
}

// InitResult — returned by vault.Init, serialized to JSON for CLI output
type InitResult struct {
    Status   string   `json:"status"`            // "created" | "ok" | "repaired"
    Vault    string   `json:"vault"`             // absolute path to vault
    Repaired []string `json:"repaired,omitempty"` // paths that were repaired
}

// InitError — error result, serialized to JSON for CLI output
type InitError struct {
    Error string `json:"error"`
}

// VaultEntry — defines a required path in the vault structure
type VaultEntry struct {
    Path     string    // relative path within vault (e.g., "_meta/writing-protocol.md")
    IsDir    bool      // true for directories, false for files
    Template string    // template filename in embedded FS (empty for dirs)
}
```

</type_definitions>

<signatures>

## Function and API Signatures

```go
// package vault (internal/vault/)

// Init scaffolds or repairs a vault at the given path.
// Returns InitResult on success, error on failure.
func Init(vaultPath string, opts InitOptions) (*InitResult, error)

// VaultStructure returns the ordered list of entries that define a complete vault.
func VaultStructure() []VaultEntry

// package cli (internal/cli/)

// NewRootCmd creates the root cobra command with all subcommands registered.
func NewRootCmd() *cobra.Command

// newInitCmd creates the `init` subcommand.
func newInitCmd() *cobra.Command

// newInstructionsCmd creates the `instructions` subcommand.
func newInstructionsCmd() *cobra.Command
```

No API endpoints — this is a CLI tool.

</signatures>

<vertical_slices>

## Vertical Slices

<slice name="slice-1-project-skeleton" order="1">

**Scope**: Minimal buildable Go project. `go build` succeeds, binary prints version, mise tasks work.

**Files to create/modify**:
- `.mise.toml` — Go version, golangci-lint, build/test/lint tasks
- `go.mod` — module declaration
- `cmd/agent-memory/main.go` — entry point, calls `cli.NewRootCmd().Execute()`
- `internal/cli/root.go` — root cobra command (no subcommands yet), version flag

**Verification point**:
```bash
mise run build && ./bin/agent-memory --version
```

**Depends on**: none

</slice>

<slice name="slice-2-vault-structure" order="2">

**Scope**: Vault structure definition and embedded templates. No CLI wiring yet — just the vault package with its types, structure definition, and embedded template files.

**Files to create/modify**:
- `internal/vault/structure.go` — `VaultEntry` type, `VaultStructure()` function, `templateFS` embed
- `internal/vault/templates/writing-protocol.md` — seed content
- `internal/vault/templates/tag-taxonomy.md` — seed content
- `internal/vault/templates/constraints-summary.md` — seed content
- `internal/vault/templates/status-lifecycle.md` — seed content

**Verification point**:
```bash
go test ./internal/vault/ -run TestVaultStructure
```
Unit test: `VaultStructure()` returns expected entries, all template files are readable from embedded FS.

**Depends on**: slice-1-project-skeleton

</slice>

<slice name="slice-3-init-core" order="3">

**Scope**: `vault.Init()` creates a fresh vault from scratch. Wired to `agent-memory init` CLI subcommand. JSON output to stdout.

**Files to create/modify**:
- `internal/vault/init.go` — `Init()` function, `InitOptions`, `InitResult`, `InitError` types
- `internal/cli/init.go` — `init` subcommand, calls `vault.Init()`, marshals result to JSON

**Verification point**:
```bash
mise run build && ./bin/agent-memory init /tmp/test-vault
# verify JSON output and directory structure
ls /tmp/test-vault/_meta/ /tmp/test-vault/_inbox/ /tmp/test-vault/notes/
```

**Depends on**: slice-2-vault-structure

</slice>

<slice name="slice-4-idempotency-and-repair" order="4">

**Scope**: Running `init` on an existing vault is a no-op ("ok"). Running `init` on a vault with missing pieces repairs them ("repaired"). Compatibility check: fail if required dir path exists as a file.

**Files to create/modify**:
- `internal/vault/init.go` — extend `Init()` with existence checks, repair logic, compatibility validation

**Verification point**:
```bash
# double init → status "ok"
./bin/agent-memory init /tmp/test-vault
# remove a file, re-init → status "repaired"
rm /tmp/test-vault/_meta/tag-taxonomy.md && ./bin/agent-memory init /tmp/test-vault
# create conflicting file → error
touch /tmp/conflicting && ./bin/agent-memory init /tmp/conflicting
```

**Depends on**: slice-3-init-core

</slice>

<slice name="slice-5-force-and-clean" order="5">

**Scope**: `--force` flag overwrites conflicts. `--clean --force` deletes vault before init. `--clean` without `--force` errors.

**Files to create/modify**:
- `internal/vault/init.go` — extend `Init()` to handle force/clean logic
- `internal/cli/init.go` — add `--force` and `--clean` flags

**Verification point**:
```bash
# --force overwrites conflict
mkdir -p /tmp/fv/_meta && touch /tmp/fv/_meta && ./bin/agent-memory init --force /tmp/fv
# --clean --force resets vault
./bin/agent-memory init --clean --force /tmp/test-vault
# --clean without --force errors
./bin/agent-memory init --clean /tmp/test-vault
```

**Depends on**: slice-4-idempotency-and-repair

</slice>

<slice name="slice-6-instructions" order="6">

**Scope**: `agent-memory instructions` subcommand outputs the agent configuration blurb to stdout.

**Files to create/modify**:
- `internal/cli/instructions.go` — `instructions` subcommand, prints blurb to stdout

**Verification point**:
```bash
./bin/agent-memory instructions | head -1
# should output: "# Agent Memory"
```

**Depends on**: slice-1-project-skeleton

</slice>

<slice name="slice-7-integration-tests" order="7">

**Scope**: BDD integration test suite covering all 10 test cases from the design. Uses ginkgo/gomega. Build tag `integration`.

**Files to create/modify**:
- `test/integration/init_test.go` — ginkgo suite: fresh init, path init, double init, repair, conflict, force, clean+force, clean-without-force, log entry, instructions output
- `test/integration/integration_suite_test.go` — ginkgo bootstrap
- `internal/testutil/helpers.go` — shared test helpers (temp dirs, binary path)

**Verification point**:
```bash
mise run test-integration
```

**Depends on**: slice-5-force-and-clean, slice-6-instructions

</slice>

</vertical_slices>

<implementation_order>

## Implementation Order

1. **slice-1-project-skeleton**: Buildable binary with mise tasks
2. **slice-2-vault-structure**: Vault definition and embedded templates
3. **slice-3-init-core**: Fresh vault creation wired to CLI
4. **slice-4-idempotency-and-repair**: Idempotent re-runs and repair
5. **slice-5-force-and-clean**: Force and clean flags
6. **slice-6-instructions**: Instructions subcommand (can run in parallel with slices 2-5)
7. **slice-7-integration-tests**: Full BDD test suite

**Parallel opportunities**: Slice 6 (instructions) depends only on slice 1 and can be built in parallel with slices 2-5. All other slices are sequential.

</implementation_order>

</structure_artifact>
