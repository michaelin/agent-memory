<research_artifact feature="note-format-frontmatter">

<codebase_map>

## Codebase Map

Relevant files and their purposes:

| File | Purpose | Key Contents |
|------|---------|--------------|
| `internal/vault/structure.go` | Defines vault structure and embedded templates | `VaultEntry` struct, `VaultStructure()` func returning 9 entries (4 dirs + 5 files), `embed.FS` for `templates/*` |
| `internal/vault/init.go` | Vault initialization and repair | `Init()`, `InitOptions`, `InitResult`, `InitError`, `MarshalError()`, `writeTemplate()` |
| `internal/vault/templates/writing-protocol.md` | V1 writing protocol (embedded) | 17 lines, basic directory structure description and 4 rules. No note format info. |
| `internal/vault/templates/*.md` | Other seed templates | `tag-taxonomy.md`, `constraints-summary.md`, `status-lifecycle.md`, `log.md` |
| `internal/cli/root.go` | Root cobra command | `NewRootCmd()`, `--json` persistent flag (`jsonOutput` package var), registers `init` and `instructions` subcommands |
| `internal/cli/init.go` | Init subcommand | `newInitCmd()`, dual output (JSON via `json.Marshal` or human-readable `fmt.Printf`), `--force` and `--clean` flags |
| `internal/cli/instructions.go` | Instructions subcommand | `newInstructionsCmd()`, hardcoded string output (not embedded), no `--json` support |
| `cmd/agent-memory/main.go` | Binary entrypoint | Calls `cli.NewRootCmd().Execute()`, exits 1 on error |
| `internal/testutil/helpers.go` | Test helpers | `BuildBinary()`, `RunBinary()` (returns stdout/stderr/exitCode), `MustStat()`, `findModuleRoot()` |
| `internal/testutil/reporter.go` | Ginkgo tree reporter | `PrintTreeReport()` for nested spec output |
| `internal/vault/init_test.go` | Unit tests for Init | Ginkgo Describe/Context/It blocks, uses `GinkgoT().TempDir()` for isolation |
| `internal/vault/structure_test.go` | Unit tests for VaultStructure | Verifies 9 entries, dir-before-file ordering, template readability |
| `test/integration/init_test.go` | Integration tests for CLI | Build tag `//go:build integration`, uses `testutil.BuildBinary`/`RunBinary`, tests both `--json` and human-readable output |
| `test/integration/integration_suite_test.go` | Integration suite bootstrap | Ginkgo suite runner |
| `internal/vault/vault_suite_test.go` | Vault unit suite bootstrap | Ginkgo suite runner |
| `docs/AGENT_MEMORY_DESIGN.md` | Full design document | Note format spec at §5.3, lint-note spec at §8.5, project layout at §8.2 |

**Module relationships:**
- `cmd/agent-memory/main.go` → imports `internal/cli`
- `internal/cli/init.go` → imports `internal/vault`
- `test/integration/` → imports `internal/testutil`
- `internal/vault/structure.go` → embeds `internal/vault/templates/*`

**Dependencies:**
- `go.yaml.in/yaml/v3 v3.0.4` — present as indirect dependency (pulled by ginkgo)
- `gopkg.in/yaml.v3 v3.0.1` — also in go.sum (older version, likely transitive)
- No Go source files currently import any YAML library

</codebase_map>

<question_answers>

## Question Findings

<finding question="What types and structs already exist in internal/vault/ — particularly around vault structure, directory constants, and embedded templates?">

**What exists:**
- `VaultEntry` struct with `Path string`, `IsDir bool`, `Template string` fields
- `VaultStructure()` returns `[]VaultEntry` — 9 entries: 4 directories (`_meta`, `_inbox`, `_contested`, `notes`) and 5 files (all in `_meta/`)
- `templateFS` is an `embed.FS` using `//go:embed templates/*` directive
- `InitOptions` struct with `Force` and `Clean` bools
- `InitResult` struct with `Status`, `Vault`, `Repaired` fields (JSON-tagged)
- `InitError` struct with `Error` field (JSON-tagged)
- `writeTemplate()` helper reads from `templateFS` and writes with `0600` perms
- `MarshalError()` helper marshals errors to JSON

**Where:** `internal/vault/structure.go`, `internal/vault/init.go`

**What's missing:** No note-related types exist. No frontmatter struct. No parsing logic. No path-building helpers beyond what `VaultStructure()` provides (it uses relative paths joined with `filepath.Join`).

</finding>

<finding question="Does the codebase already use go.yaml.in/yaml/v3 anywhere, or is it only pulled in transitively?">

**What exists:**
- `go.yaml.in/yaml/v3 v3.0.4` is listed as `// indirect` in go.mod
- `gopkg.in/yaml.v3 v3.0.1` is also in go.sum (older version)
- No `.go` files in the project import any YAML library — zero grep hits for `yaml` in Go source

**Where:** `go.mod`, `go.sum`

**What's missing:** No existing YAML parsing patterns to follow. The design doc at §8.2 references `gopkg.in/yaml.v3` as the intended library for `internal/vault/` frontmatter parsing. Both `go.yaml.in/yaml/v3` (v3.0.4) and `gopkg.in/yaml.v3` (v3.0.1) are in go.sum. `go.yaml.in` is the newer canonical import path for the same library.

</finding>

<finding question="How does the existing vault structure define the _inbox/ and notes/ directories? Are there any path-building helpers or constants for these locations?">

**What exists:**
- `_inbox` and `notes` are defined as `VaultEntry{Path: "_inbox", IsDir: true}` and `VaultEntry{Path: "notes", IsDir: true}` in `VaultStructure()`
- Paths are relative strings; `Init()` joins them with `filepath.Join(absPath, entry.Path)`
- No standalone constants like `InboxDir = "_inbox"` exist
- No path-building helper functions beyond the `VaultStructure()` slice

**Where:** `internal/vault/structure.go`

**What's missing:** No reusable path constants or helper functions for constructing paths to specific vault directories. Each caller would need to know the string `"_inbox"` or `"notes"` directly.

</finding>

<finding question="How is the existing agent-memory CLI structured with cobra? What's the pattern for adding new subcommands?">

**What exists:**
- `NewRootCmd()` in `internal/cli/root.go` creates the root command
- Subcommands are created by unexported factory functions: `newInitCmd()`, `newInstructionsCmd()`
- Each factory returns a `*cobra.Command`
- Root command registers them via `cmd.AddCommand(newInitCmd())` etc.
- `--json` is a persistent flag on root, stored in package-level `var jsonOutput bool`
- `main.go` calls `cli.NewRootCmd().Execute()` and exits 1 on error

**Where:** `internal/cli/root.go`, `internal/cli/init.go`, `internal/cli/instructions.go`, `cmd/agent-memory/main.go`

**What's missing:** The design doc at §8.2 shows `lint-note` as a separate binary under `cmd/lint-note/`. The current `cmd/` directory only has `cmd/agent-memory/`. The design doc also shows `agent-memory lint` as a human-readable wrapper that runs `lint-note` on all files. These are two separate things: `lint-note` (standalone binary) and `agent-memory lint` (subcommand wrapper).

</finding>

<finding question="What's the pattern for CLI output in the existing codebase?">

**What exists:**
- Dual output pattern: `--json` flag controls whether JSON or human-readable text is emitted
- JSON output: `json.Marshal(result)` → `fmt.Println(string(out))`
- Human-readable: `fmt.Printf("Vault created at %s\n", result.Vault)`
- Error JSON: `json.Marshal(vault.InitError{Error: err.Error()})` → `fmt.Println(string(out))`
- When `--json` is set and error occurs: JSON error to stdout, `cmd.SilenceUsage = true`, `cmd.SilenceErrors = true`, nothing to stderr
- When `--json` is not set and error occurs: cobra default error handling (stderr)

**Where:** `internal/cli/init.go`

**What's missing:** No shared output helper. Each command implements the JSON/human-readable branching inline.

</finding>

<finding question="How does the instructions subcommand work?">

**What exists:**
- Hardcoded string in a `fmt.Print()` call — not embedded from a file
- No `--json` support (always plain text)
- Content includes: vault location, usage instructions, mentions `--json` flag
- Ends with "Writing to the vault is not yet enabled in this version."

**Where:** `internal/cli/instructions.go`

**What's missing:** No embedded file pattern for instructions content. The writing-protocol template is embedded via `embed.FS` in `internal/vault/structure.go`, but the instructions subcommand uses a hardcoded string.

</finding>

<finding question="How are the existing integration tests structured?">

**What exists:**
- Build tag `//go:build integration` on integration test files
- `BeforeSuite` builds the binary once via `testutil.BuildBinary(GinkgoTB())`
- Binary path stored in package-level `var binPath string`
- Tests use `testutil.RunBinary(binPath, args...)` which returns `stdout, stderr, exitCode`
- Temp dirs via `GinkgoT().TempDir()` (auto-cleaned by ginkgo)
- Tests verify both `--json` and human-readable output paths
- JSON responses parsed with `json.Unmarshal` into `map[string]interface{}`

**Where:** `test/integration/init_test.go`, `test/integration/integration_suite_test.go`

**What's missing:** No helper for creating a pre-initialized vault with test notes. No helper for writing test note files.

</finding>

<finding question="How are unit tests structured in internal/vault/?">

**What exists:**
- Ginkgo `Describe`/`Context`/`It` blocks (dot-imported `ginkgo/v2` and `gomega`)
- `BeforeEach` creates temp dirs via `GinkgoT().TempDir()`
- Individual `It` blocks per test case (not table-driven)
- Assertions use gomega matchers: `Expect(x).To(Equal(y))`, `Expect(err).NotTo(HaveOccurred())`, `ContainSubstring`, `BeADirectory`, `BeAnExistingFile`, `HaveLen`
- Suite file at `internal/vault/vault_suite_test.go`

**Where:** `internal/vault/init_test.go`, `internal/vault/structure_test.go`

**What's missing:** No table-driven test patterns observed. No `DescribeTable`/`Entry` usage.

</finding>

<finding question="What does the existing writing-protocol.md template contain (v1)?">

**What exists:**
- 17-line file at `internal/vault/templates/writing-protocol.md`
- Title: "# Writing Protocol"
- Describes directory structure (`_meta/`, `_inbox/`, `_contested/`, `notes/`)
- 4 rules: read `_meta/` first, check `_inbox/`, don't modify `_meta/`, notes in `notes/` are reliable
- No mention of note format, frontmatter, or validation

**Where:** `internal/vault/templates/writing-protocol.md`

**What's missing:** No note format specification. No frontmatter field descriptions. No validation instructions. This is the v1 content that would be replaced/expanded by v2.

</finding>

<finding question="How should the frontmatter parser handle the project field when scope: cross-project?">

**What exists:**
- Design doc §5.3 states: `project: project-slug  # source project; cross-project notes leave this blank`
- This implies `project` should be blank/empty when `scope: cross-project`, not absent
- No explicit statement about rejecting `project` when `scope: cross-project`

**Where:** `docs/AGENT_MEMORY_DESIGN.md` line 686

**What's missing:** No explicit rule about whether a non-empty `project` with `scope: cross-project` is an error or just ignored.

</finding>

<finding question="What constitutes a placeholder value?">

**What exists:**
- Design doc §8.5 states: "No frontmatter fields with placeholder values" as a lint check
- No explicit list of placeholder patterns defined anywhere in the codebase or design doc
- The note format example in §5.3 uses actual values, not placeholders
- The `source-artifact` field example uses `"<repo-relative path or URL>"` with angle brackets — this is a documentation placeholder pattern

**Where:** `docs/AGENT_MEMORY_DESIGN.md` lines 1345, 689

**What's missing:** No concrete list of placeholder patterns. Common patterns in the wild: empty strings `""`, angle-bracket placeholders `<...>`, `TODO`, `TBD`, `FIXME`, `xxx`, `placeholder`.

</finding>

<finding question="How should lint-note handle extra/unknown frontmatter fields?">

**What exists:**
- Design doc §5.3 defines required and optional fields exhaustively
- No explicit statement about unknown fields
- The design doc does not mention strict vs lenient unmarshaling

**Where:** `docs/AGENT_MEMORY_DESIGN.md` §5.3

**What's missing:** No policy on unknown fields. The YAML library supports `KnownFields(true)` for strict mode.

</finding>

<finding question="go.yaml.in/yaml/v3 vs gopkg.in/yaml.v3 — same library?">

**What exists:**
- `go.yaml.in/yaml/v3 v3.0.4` in go.mod (indirect)
- `gopkg.in/yaml.v3 v3.0.1` in go.sum only
- `go.yaml.in` is the newer canonical import path for the same YAML library (formerly `gopkg.in/yaml.v3`)
- The design doc references `gopkg.in/yaml.v3` at §8.2 line 1141

**Where:** `go.mod`, `go.sum`, `docs/AGENT_MEMORY_DESIGN.md` line 1141

**What's missing:** The design doc references the old import path. The go.mod already has the newer path as an indirect dependency.

</finding>

<finding question="Should the domain field accept a single string as well as a list?">

**What exists:**
- Design doc §5.3 shows `domain: [golang, auth]` — YAML list syntax
- No mention of accepting a bare string

**Where:** `docs/AGENT_MEMORY_DESIGN.md` line 687

**What's missing:** No explicit statement about whether `domain: golang` (bare string) is accepted or must be `domain: [golang]`.

</finding>

</question_answers>

<patterns>

## Existing Patterns and Conventions

- **Embedded templates**: `//go:embed templates/*` with `embed.FS`, read via `templateFS.ReadFile(name)`, written with `os.WriteFile(path, data, 0600)`
- **CLI structure**: Unexported factory functions (`newXxxCmd()`) returning `*cobra.Command`, registered in `NewRootCmd()` via `AddCommand()`
- **Dual output**: `--json` persistent flag on root; each command branches on `jsonOutput` for JSON vs human-readable
- **JSON error pattern**: On error with `--json`, marshal error to JSON on stdout, silence cobra's stderr, return wrapped error for exit code 1
- **Result types**: Dedicated result/error structs with JSON tags in the domain package (`internal/vault`), not in CLI
- **Test isolation**: `GinkgoT().TempDir()` for temp dirs (auto-cleanup), `BeforeEach` for per-test setup
- **Integration test pattern**: Build binary once in `BeforeSuite`, run via `testutil.RunBinary()`, verify both JSON and human-readable output
- **File permissions**: Directories `0700`, files `0600`
- **Error wrapping**: `fmt.Errorf("context: %w", err)` throughout

</patterns>

<gaps>

## Gaps and Missing Capabilities

- **No note types**: No `Note`, `Frontmatter`, or related structs exist anywhere
- **No YAML parsing**: No Go source imports any YAML library; the indirect dependency is unused by project code
- **No lint-note binary**: `cmd/lint-note/` does not exist; design doc shows it as a separate binary
- **No lint package**: `internal/lint/` does not exist
- **No path constants**: Vault directory names (`_inbox`, `notes`, `_meta`, `_contested`) are only defined as strings in `VaultStructure()` entries, not as reusable constants
- **No shared output helpers**: Each CLI command implements JSON/human-readable branching inline
- **No test note helpers**: No helper for creating test notes with frontmatter in temp vaults
- **No placeholder pattern definition**: No concrete list of what constitutes a placeholder value
- **No policy on unknown frontmatter fields**: Strict vs lenient unmarshaling not specified
- **No policy on domain field format**: Single string vs list not specified

</gaps>

</research_artifact>
