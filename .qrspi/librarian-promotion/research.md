<research_artifact feature="librarian-promotion">

<codebase_map>

## Codebase Map

| File | Purpose | Key Contents |
|------|---------|--------------|
| `internal/note/note.go` | Note and Frontmatter types, Parse() | `Frontmatter` struct with all fields including `Status` (string), `RequiresHumanReview` (bool), `UpdateType` (string), `Targets` ([]string), `Tags` ([]string). `Parse()` unmarshals YAML frontmatter + body. |
| `internal/note/serialize.go` | Serialize() and Slug() | `Serialize()` marshals via `yaml.Marshal` then wraps in `---` delimiters. `Slug()` produces lowercase alphanumeric-hyphen slugs, max 60 chars. |
| `internal/note/write.go` | Write(), findSimilarNotes(), resolveWikilinks(), appendLog() | `Write()` assembles note, lints, similarity-checks, resolves wikilinks, writes to `_inbox/`, appends log. `findSimilarNotes()` scans both `_inbox/` and `notes/`. `appendLog()` is a private function. |
| `internal/note/lint.go` | Lint(), Rules(), LintResult, LintError, rules NF001-NF007 | `Lint()` takes `*Note`, returns `*LintResult`. Rules: NF001 (required fields), NF002 (date format), NF003 (enum values — status allows `inbox`, `verified`, `deprecated`, `contested`, `superseded`), NF004 (project required when scope=project), NF005 (body sections), NF006 (placeholder detection), NF007 (domain non-empty). |
| `internal/note/similarity.go` | NormalizeTokens(), Jaccard() | Token normalization (lowercase, strip non-ASCII, remove stop words) and Jaccard similarity coefficient. |
| `internal/note/wikilinks.go` | ExtractWikilinks() | Returns `[]string` of unique wikilink slugs from content, skipping code blocks/spans. Returns raw slugs only — no resolution status. |
| `internal/note/helpers.go` | IsPlaceholder(), IsValidDate() | Utility functions used by lint rules. |
| `internal/note/source_agent.go` | DetectSourceAgent() | Auto-detects calling agent from environment. |
| `internal/vault/discover.go` | Discover() | Resolves vault path: explicit > env var > walk-up > error. |
| `internal/vault/init.go` | Init() | Vault initialization with embedded templates. |
| `internal/cli/root.go` | NewRootCmd() | Root cobra command. `jsonOutput` package-level bool, persistent `--json` flag. Subcommands added via `cmd.AddCommand()`. |
| `internal/cli/write_note.go` | newWriteNoteCmd() | Cobra subcommand for `write-note`. Uses `vault.Discover(vaultFlag)` for vault resolution. Flags registered on the command. Dual output (JSON/human). |
| `internal/cli/lint_note.go` | newLintNoteCmd() | Cobra subcommand for `lint-note`. Takes file path arg. Calls `note.Parse()` then `note.Lint()`. Dual output. |
| `internal/cli/init.go` | newInitCmd() | Cobra subcommand for `init`. |
| `internal/cli/instructions.go` | newInstructionsCmd() | Prints agent configuration blurb. Hardcoded string. Mentions `lint-note` and `write-note` but not promotion or Librarian. |
| `internal/testutil/helpers.go` | BuildBinary(), RunBinary(), MustStat(), WriteTestNote() | Test helpers: build binary, run with args, assert file exists, write test note to directory. |
| `internal/testutil/reporter.go` | Ginkgo reporter setup | Test reporter configuration. |

**Module relationships:**
- `internal/cli/*` → imports `internal/note` and `internal/vault`
- `internal/note/write.go` → uses `internal/note/lint.go`, `similarity.go`, `wikilinks.go`, `serialize.go`, `helpers.go`
- `internal/cli/root.go` → registers all subcommands
- `cmd/agent-memory/main.go` → calls `cli.NewRootCmd().Execute()`

</codebase_map>

<question_answers>

## Question Findings

<finding question="What fields in Frontmatter struct are relevant to promotion? Is Status typed or raw string? Are RequiresHumanReview, UpdateType, and Targets defined?">

**What exists:** The `Frontmatter` struct in `internal/note/note.go` (lines 23-42) has all relevant fields already defined:
- `Status string` — raw string, not a typed enum. Valid values enforced by lint rule NF003: `inbox`, `verified`, `deprecated`, `contested`, `superseded`.
- `RequiresHumanReview bool` — already defined with yaml tag `requires-human-review`.
- `UpdateType string` — already defined with yaml tag `update-type`.
- `Targets []string` — already defined with yaml tag `targets`.
- `ReviewBy string` — already defined with yaml tag `review-by`.
- `VerifiedBy string` and `VerifiedDate string` — already defined.
- `Tags []string` — already defined.

**Where:** `internal/note/note.go:23-42`

**What's missing:** No `SupersededBy` or `DeprecatedBy` field exists in the struct. If the deprecation protocol requires a forward-link field in frontmatter (e.g., `superseded-by: slug`), it would need to be added. Currently the struct has no such field — forward links would need to go in the body's `## Related` section or a new frontmatter field.

</finding>

<finding question="How does Serialize() handle round-tripping? Does it preserve field order, comments, and body?">

**What exists:** `Serialize()` in `internal/note/serialize.go` (lines 16-34) uses `yaml.Marshal(n.Frontmatter)` to produce the YAML block, then wraps it in `---` delimiters and appends the body. This means:
- Field order is determined by `yaml.Marshal`, which follows struct field order (Go struct definition order), NOT the original file's field order.
- YAML comments are NOT preserved — `yaml.Marshal` produces clean YAML from the struct.
- Body content IS preserved exactly (it's stored as a string and written back verbatim).
- Empty/zero-value fields: `yaml.Marshal` with no `omitempty` tags will emit all fields. The struct has no `omitempty` tags, so all fields are always emitted, including empty strings and false bools.

**Where:** `internal/note/serialize.go:16-34`

**What's missing:** No mechanism to preserve original YAML formatting or field order. A round-trip (Parse → modify field → Serialize) will reorder fields to match struct order and strip any comments. This is acceptable for promotion (the note is being moved, not edited in place by a human), but worth noting.

</finding>

<finding question="What does Slug() produce, and does it match the filename convention used by Write()?">

**What exists:** `Slug()` in `serialize.go` (lines 46-76) converts a title to lowercase, replaces spaces with hyphens, strips non-ASCII and non-alphanumeric-hyphen characters, collapses consecutive hyphens, trims leading/trailing hyphens, and caps at 60 characters. Empty input returns empty string; all-non-ASCII returns `"note"`.

`Write()` in `write.go` uses `resolveInboxPath()` (lines 281-305) which produces filenames as `{date}-{slug}.md` in `_inbox/`. The slug comes from `Slug(opts.Title)`. Collision handling appends `-2`, `-3`, etc.

The filename in `_inbox/` is `{YYYY-MM-DD}-{slug}.md`. There is no corresponding convention for `notes/` — no code currently writes to `notes/`.

**Where:** `internal/note/serialize.go:46-76`, `internal/note/write.go:137,145,281-305`

**What's missing:** No function exists to extract the slug from an inbox filename (i.e., strip the date prefix). No convention is defined for filenames in `notes/`. The `resolveWikilinks()` function in `write.go` (lines 236-276) checks `notes/{slug}.md` for resolution, implying the `notes/` convention is `{slug}.md` (no date prefix).

</finding>

<finding question="What is the current filename format for notes in _inbox/? Does promote need to rename?">

**What exists:** Inbox filenames are `{YYYY-MM-DD}-{slug}.md` (e.g., `2026-05-21-my-observation.md`). This is produced by `resolveInboxPath()` in `write.go:281-305`.

The `resolveWikilinks()` function (write.go:236-276) checks for notes in `notes/` using the pattern `notes/{slug}.md` — no date prefix. It checks `_inbox/` using suffix matching: any file ending with `-{slug}.md`.

**Where:** `internal/note/write.go:281-305` (inbox path), `internal/note/write.go:236-276` (wikilink resolution showing notes/ convention)

**What's missing:** No code currently writes to `notes/`. The wikilink resolution logic implies `notes/` files are named `{slug}.md` (no date prefix), which means promotion would need to rename the file (strip the date prefix).

</finding>

<finding question="How are existing subcommands structured? Pattern for adding new ones? How is --json handled?">

**What exists:** Each subcommand is in its own file in `internal/cli/`:
- `init.go` → `newInitCmd()`
- `instructions.go` → `newInstructionsCmd()`
- `lint_note.go` → `newLintNoteCmd()`
- `write_note.go` → `newWriteNoteCmd()`

Pattern: each file defines a `newXxxCmd() *cobra.Command` function. `root.go` calls `cmd.AddCommand(newXxxCmd())` for each. Flags are registered on the command in the factory function using `cmd.Flags().StringVar(...)` etc.

`jsonOutput` is a package-level `bool` in `root.go` (line 8), set via a persistent flag on the root command (line 17). All subcommands access it directly as `jsonOutput`. Each subcommand handles dual output inline: check `jsonOutput`, marshal to JSON or print human-readable.

**Where:** `internal/cli/root.go`, `internal/cli/write_note.go`, `internal/cli/lint_note.go`

**What's missing:** No shared output helper — each subcommand reimplements the JSON/human output pattern. No shared vault-flag helper — `write_note.go` defines its own `vaultFlag` string and calls `vault.Discover(vaultFlag)`.

</finding>

<finding question="How does write-note handle vault discovery? Shared helper or independent?">

**What exists:** `write_note.go` defines a local `vaultFlag string` variable (line 17), registers it as `--vault` flag (line 132), and calls `vault.Discover(vaultFlag)` (line 62). This is self-contained within the subcommand — no shared helper.

`vault.Discover()` in `internal/vault/discover.go` takes an explicit path string and falls back through env var → walk-up → error.

**Where:** `internal/cli/write_note.go:17,62,132`, `internal/vault/discover.go:18-64`

**What's missing:** No shared vault flag registration. Each subcommand that needs vault discovery must define its own `--vault` flag and call `vault.Discover()` independently. This is the established pattern.

</finding>

<finding question="What does the log-writing pattern look like? Is there a shared log-writing helper?">

**What exists:** `appendLog()` in `internal/note/write.go` (lines 309-336) is a private (unexported) function. It:
- Creates `_meta/` dir if needed
- Opens `_meta/log.md` with `O_APPEND|O_CREATE|O_WRONLY`
- Writes format: `\n## {RFC3339 timestamp} write | {type} | {slug} | by:{agent}\n`
- Example: `## 2026-05-21T14:30:00+03:00 write | observation | my-note | by:research/analyst-a`

The function is private to the `note` package. It takes `vaultPath`, `epistemicType`, `slug`, and `sourceAgent` as parameters.

**Where:** `internal/note/write.go:309-336`

**What's missing:** `appendLog()` is unexported and hardcodes the `write` action. A promote action would need either: (a) a new exported log-writing function, (b) making `appendLog` exported and parameterizing the action, or (c) a new private function in whatever package implements promote. The log format uses `## ` prefix (h2 heading) for each entry.

</finding>

<finding question="How does ExtractWikilinks() work? Does it return resolved vs. unresolved status?">

**What exists:** `ExtractWikilinks()` in `internal/note/wikilinks.go` (lines 28-93) returns `[]string` — a list of unique wikilink slugs found in the content. It:
- Parses content with goldmark to identify code block/span byte ranges
- Applies regex `\[\[([^\]]+)\]\]` to find all wikilinks
- Skips matches inside code ranges
- Extracts slug from `[[slug|alias]]` format (takes part before `|`)
- Deduplicates, preserves first-appearance order

It returns raw slugs only. No resolution status. No information about whether the target exists.

Resolution logic exists separately in `resolveWikilinks()` (write.go:236-276), which is also private. It calls `ExtractWikilinks()`, then checks each slug against `notes/{slug}.md` and `_inbox/*-{slug}.md`.

**Where:** `internal/note/wikilinks.go:28-93`, `internal/note/write.go:236-276`

**What's missing:** No exported wikilink resolution function. `resolveWikilinks()` is private to the `note` package and returns `[]string` of warning messages, not structured data. The promote subcommand would need either to call the private function (if in the same package) or to have an exported version.

</finding>

<finding question="How does findSimilarNotes() work? Threshold? Could it be extracted?">

**What exists:** `findSimilarNotes()` in `internal/note/write.go` (lines 183-232) is a private function. It:
- Scans both `_inbox/` and `notes/` directories
- Reads each `.md` file, parses it, extracts title tokens via `NormalizeTokens()`
- Computes Jaccard similarity against incoming tokens
- Returns all notes with similarity >= 0.7 as `[]SimilarNote`
- Skips symlinks, directories, non-.md files, and unparseable files

The threshold (0.7) is hardcoded on line 221.

**Where:** `internal/note/write.go:183-232`

**What's missing:** `findSimilarNotes()` is unexported. It already scans both `_inbox/` and `notes/`, so it could serve the Librarian's deduplication needs. To reuse it, it would need to be exported or a new exported wrapper created. The threshold is not configurable.

</finding>

<finding question="What does Lint() return? What are rules NF001-NF007? Can promote call Lint() directly?">

**What exists:** `Lint()` in `internal/note/lint.go` (lines 44-61) takes `*Note` and returns `*LintResult`. `LintResult` has `Valid bool` and `Errors []LintError`. `LintError` has `Rule string` and `Message string`.

Rules:
- NF001: Required fields non-empty (title, created, updated, status, confidence, epistemic-type, scope, source-artifact)
- NF002: Date fields valid YYYY-MM-DD (created, updated, review-by)
- NF003: Enum fields valid (status: inbox/verified/deprecated/contested/superseded; epistemic-type: 6 types; confidence: low/medium/high; scope: project/cross-project)
- NF004: project required when scope=project
- NF005: Body sections present (# title, ## Evidence, ## Implications, ## Related; or ## Synthesis, ## Contributing notes for synthesis type)
- NF006: No placeholder values in required fields
- NF007: domain must be non-empty list

`Lint()` is exported and takes a `*Note`. The promote subcommand can call it directly — no need to shell out to the CLI binary.

**Where:** `internal/note/lint.go:44-61` (Lint function), `internal/note/lint.go:65-237` (rules)

**What's missing:** Nothing — `Lint()` is fully usable programmatically. Note: NF003 already accepts `status: verified` as valid, so changing status before linting would pass.

</finding>

<finding question="What is the structure of existing skills? SKILL.md template?">

**What exists:** Skills are stored in `~/.config/opencode/skills/{skill-name}/SKILL.md`. The format uses YAML frontmatter with `name` and `description` fields, followed by markdown content. Example from `adversarial-code-review/SKILL.md`:
```
---
name: adversarial-code-review
description: >
  ...multi-line description...
---
# Skill Title
...instructions...
```

Skills can have subdirectories (e.g., `workflows/`, `templates/`, `reference/`, `shared/`). The SKILL.md is the entry point. Skills are referenced by name in the `<available_skills>` section of agent configurations.

There is a `create-agent-skills` skill available that provides guidance on creating skills.

**Where:** `~/.config/opencode/skills/*/SKILL.md`

**What's missing:** No Librarian skill exists yet. The skill would need to be created at `~/.config/opencode/skills/librarian/SKILL.md` (or similar).

</finding>

<finding question="How does agent-memory instructions describe the workflow? Does it mention Librarian or promotion?">

**What exists:** The `instructions` subcommand in `internal/cli/instructions.go` (lines 9-17) outputs a hardcoded string. The current output:
- Mentions vault location at `.agent-memory/`
- Tells agents to read `_meta/writing-protocol.md`
- Tells agents to check `_inbox/` and `notes/`
- Documents `lint-note` and `write-note` usage
- Mentions `--json` flag and `--force` flag

**Where:** `internal/cli/instructions.go:15`

**What's missing:** No mention of promotion, the Librarian, or what happens after notes land in `_inbox/`. No mention of the Librarian skill invocation as the last step of the workflow. The instructions stop at "write a note."

</finding>

<finding question="What tool-use declarations does a skill need? How are bash permissions scoped?">

**What exists:** Based on the skill format observed, skills are markdown instruction documents. They do not have explicit tool-use declarations or permission scoping in their SKILL.md files. The skill content is injected into the conversation context when invoked, and the agent's existing tool access (bash, read, write, etc.) is used.

Bash tool access is controlled at the agent level (in agent definition files), not at the skill level. Skills inherit the invoking agent's tool permissions.

**Where:** `~/.config/opencode/skills/*/SKILL.md` (observed format)

**What's missing:** Skills do not declare their own tool permissions. The Librarian skill would rely on the invoking agent's bash access to call `agent-memory promote`, `agent-memory lint-note`, etc. This means the Librarian skill is a set of instructions, not a sandboxed execution environment.

</finding>

<finding question="What happens if _inbox/ contains a note whose slug collides with an existing file in notes/?">

**What exists:** No code currently moves files from `_inbox/` to `notes/`. The `resolveInboxPath()` function handles collisions within `_inbox/` by appending `-2`, `-3`, etc. There is no equivalent collision-handling logic for `notes/`.

The `findSimilarNotes()` function scans both directories and would detect title similarity, but slug collision (same slug, different title) is a separate concern.

**Where:** `internal/note/write.go:281-305` (inbox collision handling)

**What's missing:** No collision handling for `notes/` directory. The promote subcommand would need to define behavior: error on collision, or handle it (e.g., this is a replacement/deprecation case).

</finding>

<finding question="How should promote handle inbox-to-inbox wikilinks? Unresolved or allowed?">

**What exists:** The `resolveWikilinks()` function in `write.go:236-276` checks BOTH `notes/{slug}.md` AND `_inbox/*-{slug}.md` when resolving wikilinks. A wikilink to another inbox note resolves successfully (no warning generated).

**Where:** `internal/note/write.go:236-276`

**What's missing:** The question of whether promotion should block on wikilinks that resolve to `_inbox/` (not yet promoted) vs. only blocking on truly unresolved links is a policy decision not encoded in the current code. The existing resolution logic treats inbox notes as valid targets.

</finding>

<finding question="What is the deprecation mechanism? Is there a superseded-by field?">

**What exists:** The `Frontmatter` struct has:
- `Status string` — can be set to `deprecated` or `superseded` (both valid per NF003)
- `UpdateType string` — exists but unused in current code
- `Targets []string` — exists but unused in current code

There is no `SupersededBy` or `DeprecatedBy` field in the struct. The `Targets` field (yaml tag: `targets`) could potentially serve this purpose, but its semantics are not defined in the code.

**Where:** `internal/note/note.go:39-40` (UpdateType, Targets)

**What's missing:** No `superseded-by` field. No code that sets `status: deprecated` or `status: superseded`. No code that adds forward links. The deprecation mechanism would need to either: (a) add a new `SupersededBy string` field to `Frontmatter`, or (b) use the existing `Targets` field with a convention, or (c) rely on body content (adding a link in `## Related`).

</finding>

<finding question="How does --confirmed interact with the Librarian? What if called without --confirmed on a constraint?">

**What exists:** No `promote` subcommand exists yet. The `requiresHumanReview()` function in `write.go:61-63` returns true for `constraint` and `decision` types. The `Write()` function sets `RequiresHumanReview: requiresHumanReview(opts.EpistemicType)` in the frontmatter (write.go:101).

**Where:** `internal/note/write.go:61-63,101`

**What's missing:** The `--confirmed` flag behavior is entirely unimplemented. The promote subcommand needs to define: (a) read the note's `RequiresHumanReview` field, (b) if true and `--confirmed` not passed, refuse with an error, (c) if true and `--confirmed` passed, proceed with promotion.

</finding>

<finding question="Are there existing tests that exercise note file movement? What test utilities exist?">

**What exists:** `internal/testutil/helpers.go` provides:
- `BuildBinary(t)` — builds the `agent-memory` binary, returns path
- `RunBinary(binPath, args...)` — runs binary, returns stdout/stderr/exitCode
- `RunBinaryWithStdin(binPath, stdin, args...)` — same but with stdin
- `RunBinaryCmd(binPath, args...)` — returns `*exec.Cmd` for custom setup
- `MustStat(t, path)` — asserts path exists
- `WriteTestNote(t, dir, filename, content)` — writes a test note file

Integration tests are in `test/integration/` with `//go:build integration` tag.

**Where:** `internal/testutil/helpers.go`

**What's missing:** No test helper for file movement (rename/move between directories). No helper to create a fully initialized vault for testing (though `BuildBinary` + `RunBinary` with `init` could do this). No helper to assert a file does NOT exist (inverse of `MustStat`).

</finding>

</question_answers>

<patterns>

## Existing Patterns and Conventions

- **One file per subcommand**: Each CLI subcommand lives in its own file in `internal/cli/`, with a factory function `newXxxCmd() *cobra.Command`.
- **Dual output**: Every subcommand checks `jsonOutput` (package-level bool) and emits either JSON or human-readable output. JSON output uses `json.Marshal` + `fmt.Println`. No shared helper.
- **Vault discovery per-command**: Each subcommand that needs the vault defines its own `--vault` flag and calls `vault.Discover()` independently.
- **Lint-then-write**: `Write()` calls `Lint()` before writing. Lint failure returns `WriteResult{Status:"refused"}` — not an error.
- **Private helpers**: `appendLog()`, `findSimilarNotes()`, `resolveWikilinks()`, `resolveInboxPath()` are all unexported in the `note` package.
- **Error handling**: Subcommands return errors from `RunE`. `cmd.SilenceUsage = true` is set before returning errors to suppress usage output. `cmd.SilenceErrors = true` is set when JSON output has already been printed.
- **Filename conventions**: Inbox files are `{date}-{slug}.md`. Notes files are `{slug}.md` (implied by wikilink resolution, not yet implemented).
- **Test pattern**: Integration tests use `//go:build integration`, build the binary via `testutil.BuildBinary()`, and run it via `testutil.RunBinary()`. Unit tests use Ginkgo/Gomega with `Describe/Context/It` blocks.
- **YAML round-trip**: `Parse()` → struct → `Serialize()` does not preserve original formatting. Struct field order determines output order.

</patterns>

<gaps>

## Gaps and Missing Capabilities

- **No promote subcommand**: No code exists to move notes from `_inbox/` to `notes/`. No CLI command, no library function.
- **No exported log-writing function**: `appendLog()` is private and hardcodes the `write` action. Promote needs a log entry with a different action.
- **No exported wikilink resolution**: `resolveWikilinks()` is private and returns warning strings, not structured data. Promote needs to check for unresolved links as a hard gate.
- **No exported similarity scan**: `findSimilarNotes()` is private. The Librarian's deduplication check needs the same logic.
- **No `superseded-by` frontmatter field**: The struct has no field for forward-linking to a replacement note. `Targets` exists but has no defined semantics.
- **No notes/ filename convention code**: No function generates the target path in `notes/`. Wikilink resolution implies `{slug}.md` but no code produces this.
- **No Librarian skill**: No skill definition exists.
- **Instructions don't mention promotion or Librarian**: The `instructions` output stops at write-note usage.
- **No file-move test helpers**: No test utility for asserting file movement between directories.
- **No collision handling for notes/**: `resolveInboxPath()` handles inbox collisions but nothing equivalent exists for `notes/`.

</gaps>

</research_artifact>
