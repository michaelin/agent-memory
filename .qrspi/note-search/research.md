<research_artifact feature="note-search">

<codebase_map>

## Codebase Map

| File | Purpose | Key Contents |
|------|---------|--------------|
| `internal/note/note.go` | Note type + Parse() | `Note{Frontmatter, Body string}`, `Frontmatter` struct with all fields as exported typed fields, `Parse(content []byte) (*Note, error)` |
| `internal/note/lint.go` | Lint rules NF001–NF007 | `Lint(*Note) *LintResult`, `Rules() []Rule`, enum valid values hardcoded in NF003 |
| `internal/note/serialize.go` | Serialize + Slug | `Serialize(*Note) ([]byte, error)`, `Slug(title string) string` |
| `internal/note/similarity.go` | Jaccard similarity | `NormalizeTokens(s string) []string`, `Jaccard(a, b []string) float64` |
| `internal/note/wikilinks.go` | Wikilink extraction | `ExtractWikilinks(text string) []string` — goldmark-based, skips code blocks |
| `internal/note/source_agent.go` | Harness detection | `DetectSourceAgent() string` |
| `internal/note/write.go` | Write pipeline | `Write(WriteOptions) (WriteResult, error)`, `findSimilarNotes()`, `resolveWikilinks()`, `resolveInboxPath()`, `appendLog()` |
| `internal/vault/discover.go` | Vault discovery | `Discover(explicitPath string) (string, error)` — returns absolute path string |
| `internal/vault/init.go` | Vault initialization | `Init()` function, embedded templates |
| `internal/cli/root.go` | Root cobra command | `NewRootCmd() *cobra.Command`, `var jsonOutput bool` (package-level), `--json` persistent flag |
| `internal/cli/write_note.go` | write-note subcommand | `newWriteNoteCmd() *cobra.Command`, `--vault` flag → `vault.Discover()`, JSON/human output branching |
| `internal/cli/instructions.go` | instructions subcommand | Agent blurb output |
| `internal/vault/templates/tag-taxonomy.md` | Tag taxonomy template | Placeholder only: `<!-- Placeholder: define project-specific tags here -->` — no structured format, no alias syntax |
| `internal/vault/templates/status-lifecycle.md` | Status lifecycle | Lists: inbox, active, contested, archived, deprecated |

**Module relationships:**
- `internal/cli/` → imports `internal/note/` and `internal/vault/`
- `internal/note/write.go` → calls `Parse()`, `Lint()`, `Serialize()`, `Slug()`, `NormalizeTokens()`, `Jaccard()`, `ExtractWikilinks()`, `DetectSourceAgent()`
- `internal/note/write.go` → uses `os.ReadDir()` directly to scan `_inbox/` and `notes/` (in `findSimilarNotes()` and `resolveWikilinks()`)
- No shared file-scanning utility exists

</codebase_map>

<question_answers>

## Question Findings

<finding question="What fields does Frontmatter currently expose, and which are exported?">

**What exists:** All fields are exported. The struct at `internal/note/note.go:23-42` has:
- `Title string`, `Created string`, `Updated string`, `ReviewBy string`
- `Status string`, `Confidence string`, `EpistemicType string`
- `Scope string`, `Project string`
- `Domain []string`, `Tags []string`, `Targets []string`
- `SourceAgent string`, `SourceArtifact string`
- `VerifiedBy string`, `VerifiedDate string`
- `RequiresHumanReview bool`, `UpdateType string`

All filterable fields (`Status`, `Project`, `Domain`, `EpistemicType`) are plain strings or `[]string`. No typed enums — valid values are hardcoded in lint rule NF003.

**Where:** `internal/note/note.go:23-42`, `internal/note/lint.go:136-139`

**What's missing:** No method to check if a note matches a set of filter criteria. Each caller must do field comparisons manually.

</finding>

<finding question="How does Parse() handle the body vs. frontmatter split?">

**What exists:** `Parse(content []byte) (*Note, error)` splits on `---` delimiters. Returns `Note{Frontmatter, Body string}` where Body is the full text after the closing `---`, trimmed of leading/trailing whitespace. Body is a single string — no further processing needed to access it.

**Where:** `internal/note/note.go:61-101`

**What's missing:** Nothing. Body is directly accessible as `note.Body`.

</finding>

<finding question="What is the current structure of _meta/tag-taxonomy.md?">

**What exists:** The embedded template at `internal/vault/templates/tag-taxonomy.md` contains only:
```
# Tag Taxonomy

<!-- Placeholder: define project-specific tags here -->
```

No structured format. No alias syntax. No parsing code exists anywhere in the codebase for tag-taxonomy.md.

**Where:** `internal/vault/templates/tag-taxonomy.md`

**What's missing:** No tag alias format is defined. No parser exists. Tag alias expansion cannot be implemented without first defining the format.

</finding>

<finding question="How are notes stored on disk?">

**What exists:** `_inbox/` files are named `{YYYY-MM-DD}-{slug}.md` with collision suffixes `-2`, `-3`, etc. (see `resolveInboxPath()` at write.go:281-305). `notes/` directory exists but is flat — no subdirectories. The `findSimilarNotes()` function at write.go:183-232 scans both `_inbox/` and `notes/` using `os.ReadDir()`, filtering for `.md` files and skipping symlinks and directories.

**Where:** `internal/note/write.go:183-232`, `internal/note/write.go:281-305`

**What's missing:** No notes currently exist in `notes/` because promotion hasn't been implemented yet (increment 5). All written notes land in `_inbox/`.

</finding>

<finding question="What is the established pattern for subcommand flag parsing and validation?">

**What exists:** `write-note` uses cobra's `cmd.Flags().StringVar()` for each flag, with `cmd.MarkFlagRequired()` for `--type` and `--title`. No custom mutual-exclusion logic exists. Validation happens implicitly through the `note.Write()` pipeline (lint catches invalid values). The `--vault` flag is per-command, not persistent.

**Where:** `internal/cli/write_note.go:132-146`

**What's missing:** No pattern for mutually exclusive flags or conditional requirements (e.g., "if --phase=2 then --slugs is required"). Cobra supports `cmd.MarkFlagsRequiredTogether()` and `cmd.MarkFlagsMutuallyExclusive()` but neither is used yet.

</finding>

<finding question="How does the existing --json persistent flag work?">

**What exists:** `var jsonOutput bool` is a package-level variable in `internal/cli/root.go:8`. It's registered as a persistent flag on the root command: `cmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, ...)`. Each subcommand checks `jsonOutput` directly (it's in the same package). No shared output helper — each command does its own `json.Marshal()` + `fmt.Println()`.

**Where:** `internal/cli/root.go:8,17`

**What's missing:** No shared output helper function. Each subcommand reimplements the JSON marshaling and printing pattern.

</finding>

<finding question="What is the human-readable output pattern?">

**What exists:** `write-note` has both modes. JSON mode: `json.Marshal(result)` → `fmt.Println()`. Human mode: `fmt.Printf("✓ Written: %s\n", result.Path)` for success, `fmt.Printf("✗ Refused: %s\n", result.Reason)` with candidate listing for refusal. Uses ✓/✗ Unicode symbols.

**Where:** `internal/cli/write_note.go:101-129`

**What's missing:** No shared formatting utilities. No table formatting for multi-row results (search results would need this).

</finding>

<finding question="How should the query positional argument work?">

**What exists:** No search implementation exists. The `findSimilarNotes()` function in write.go does title-based Jaccard similarity matching — it scans all `.md` files, parses each, extracts the title, tokenizes, and computes Jaccard. This is the closest existing pattern to a search. It does NOT search body text.

**Where:** `internal/note/write.go:183-232`, `internal/note/similarity.go`

**What's missing:** No query matching logic. No substring search. No body search. The design doc says "grep + YAML parsing" for Phase 1 but no grep-like functionality exists.

</finding>

<finding question="What does tag alias expansion mean concretely?">

**What exists:** The tag-taxonomy.md template is a placeholder with no structured content. No alias mapping is defined anywhere. No parsing code exists for tag aliases.

**Where:** `internal/vault/templates/tag-taxonomy.md`

**What's missing:** Everything. No format, no parser, no expansion logic. This feature cannot be implemented without defining the tag-taxonomy format first.

</finding>

<finding question="Should Phase 1 search both notes/ and _inbox/?">

**What exists:** `findSimilarNotes()` scans both `_inbox/` and `notes/`. The design doc says "status: verified only" for search results. Notes in `_inbox/` have `status: inbox` by definition (set by Write at write.go:91). Notes in `notes/` would have `status: verified` (after promotion, which doesn't exist yet).

**Where:** `internal/note/write.go:186-189`

**What's missing:** No promotion exists yet, so `notes/` is always empty. Searching only `notes/` would return nothing until increment 5.

</finding>

<finding question="How should slug matching work for Phase 2?">

**What exists:** `Slug()` in serialize.go converts a title to a filesystem-safe slug (lowercase, alphanumeric + hyphens, max 60 chars). Inbox files are named `{date}-{slug}.md`. Notes in `notes/` would presumably be named `{slug}.md` (based on `resolveWikilinks()` checking `notes/{slug}.md` at write.go:252).

**Where:** `internal/note/serialize.go:46-76`, `internal/note/write.go:252`

**What's missing:** No function to derive a slug from a filename. No function to list all slugs in the vault. The slug-to-file mapping is implicit.

</finding>

<finding question="Does Discover() return a path that can be used directly to scan for notes?">

**What exists:** `Discover(explicitPath string) (string, error)` returns an absolute path string to the vault directory (e.g., `/path/to/.agent-memory`). The caller can then join subdirectories: `filepath.Join(vaultPath, "notes")`, `filepath.Join(vaultPath, "_inbox")`. This is exactly what `write-note` does.

**Where:** `internal/vault/discover.go:18-63`

**What's missing:** Nothing. The return value is directly usable.

</finding>

<finding question="How does write-note locate the vault?">

**What exists:** `write-note` declares a `--vault` flag (string, default empty). It calls `vault.Discover(vaultFlag)` which handles the priority chain: explicit path → env var → walk-up → error. This is done inside the `RunE` function.

**Where:** `internal/cli/write_note.go:62,132`

**What's missing:** No shared vault-resolution middleware. Each subcommand must repeat the `--vault` flag declaration and `vault.Discover()` call.

</finding>

<finding question="Are there any existing file-scanning utilities?">

**What exists:** `findSimilarNotes()` in write.go is the only directory-scanning code. It uses `os.ReadDir()` → filter `.md` → skip symlinks/dirs → `os.ReadFile()` → `Parse()`. This pattern is not extracted into a reusable function.

**Where:** `internal/note/write.go:183-232`

**What's missing:** No shared "scan vault notes" utility. The scanning logic is embedded in `findSimilarNotes()`.

</finding>

<finding question="What happens when the vault has no notes at all?">

**What exists:** `findSimilarNotes()` handles missing directories gracefully: `if os.IsNotExist(err) { continue }`. An empty directory returns an empty slice. `write-note` treats empty candidates as "no similar notes found" and proceeds.

**Where:** `internal/note/write.go:193-195`

**What's missing:** No explicit "vault is empty" messaging. The pattern is: empty results → empty JSON array or no output.

</finding>

<finding question="What if a note file in notes/ has malformed frontmatter?">

**What exists:** `findSimilarNotes()` silently skips files that fail to parse: `if err != nil || parsed == nil { continue }`. No warning is emitted. The lenient unmarshaling decision from increment 2 means unknown YAML fields are silently ignored, but malformed YAML returns an error from `Parse()`.

**Where:** `internal/note/note.go:90-94`, `internal/note/write.go:214-217`

**What's missing:** No mechanism to report skipped/malformed files during scanning.

</finding>

<finding question="What is the performance profile?">

**What exists:** `findSimilarNotes()` reads every `.md` file in `_inbox/` and `notes/`, parsing each one fully (YAML decode + body extraction). For the similarity check during write, this is acceptable because it runs once per write. For search, this would run on every query.

**Where:** `internal/note/write.go:183-232`

**What's missing:** No index. No caching. No pre-built frontmatter index. The design doc mentions `memory-reindex` (increment 7) as the index builder, but search is increment 4. Search must work without pre-built indices.

</finding>

<finding question="How should the 10-note cap in Phase 2 behave?">

**What exists:** No cap enforcement exists anywhere in the codebase. The design doc says "capped at 10" but doesn't specify the error behavior.

**Where:** N/A

**What's missing:** No precedent for cap enforcement. No pattern for "too many items requested" errors.

</finding>

<finding question="Should search results include notes with status: deprecated or status: superseded?">

**What exists:** The valid status values are: `inbox`, `verified`, `deprecated`, `contested`, `superseded` (from NF003 at lint.go:136). The design doc says "status: verified only" for search. No `--status` filter flag exists anywhere.

**Where:** `internal/note/lint.go:136`

**What's missing:** No status filtering logic. No `--include-status` or `--all-statuses` flag pattern.

</finding>

</question_answers>

<patterns>

## Existing Patterns and Conventions

- **Subcommand factory pattern**: Each subcommand is an unexported function `newXxxCmd() *cobra.Command` in `internal/cli/`, registered via `AddCommand()` in `NewRootCmd()`.
- **Vault resolution**: `--vault` flag per command → `vault.Discover(vaultFlag)` in `RunE`. Not shared.
- **JSON output branching**: Check `jsonOutput` (package-level bool) → `json.Marshal()` → `fmt.Println()`. Each command reimplements this.
- **Human output**: `fmt.Printf()` with ✓/✗ symbols. No table formatting.
- **Error handling in JSON mode**: Marshal a `map[string]string{"status": "error", "error": "<static message>"}`, print it, then return the real error. Static error strings — no filesystem paths leaked.
- **File scanning**: `os.ReadDir()` → filter `.md` → skip symlinks/dirs → `os.ReadFile()` → `Parse()`. Inline in `findSimilarNotes()`, not extracted.
- **Malformed file handling**: Silent skip (`continue` on parse error). No warnings emitted.
- **Result types**: Dedicated struct with JSON tags (e.g., `WriteResult`). Status field as string discriminator (`"written"`, `"refused"`).

</patterns>

<gaps>

## Gaps and Missing Capabilities

- **No shared note-scanning utility**: The directory walk + parse pattern in `findSimilarNotes()` is not reusable. Search will need the same pattern.
- **No tag-taxonomy format or parser**: Tag alias expansion is specified in the roadmap but the taxonomy file is a placeholder with no structure. Cannot implement alias expansion without defining the format first.
- **No query matching logic**: No substring, token, or keyword matching exists. `Jaccard()` exists for similarity but is title-only and threshold-based (≥ 0.7), not a general search mechanism.
- **No slug-from-filename derivation**: Inbox files are `{date}-{slug}.md`, notes files would be `{slug}.md`. No function extracts the slug from a filename.
- **No status filtering**: All scanning code processes all files regardless of status. No filter-by-status capability.
- **No shared vault flag or output helpers**: Each subcommand reimplements vault discovery and JSON output. Not blocking but creates duplication.
- **No promotion yet**: `notes/` is always empty. Searching only verified notes returns nothing until increment 5.

</gaps>

</research_artifact>
