<design_artifact feature="note-format-frontmatter" status="final">

<current_state>

## Current State

- **Vault scaffolding**: `internal/vault/` has `Init()`, `VaultStructure()`, `VaultEntry` struct, embedded templates via `embed.FS`. Vault creates `_meta/`, `_inbox/`, `_contested/`, `notes/` directories and seeds `_meta/*.md` files.
- **CLI**: Single `agent-memory` binary with cobra subcommands (`init`, `instructions`). `--json` persistent flag for dual output. Per ADR-0001, all tools are subcommands.
- **Writing protocol v1**: 17-line template at `internal/vault/templates/writing-protocol.md`. Describes directory structure and 4 basic rules. No note format, no frontmatter spec.
- **YAML library**: `go.yaml.in/yaml/v3 v3.0.4` is an indirect dependency (via ginkgo). No Go source imports it yet. This is the newer canonical import path for the same library the design doc calls `gopkg.in/yaml.v3`.
- **No note types exist**: No `Note`, `Frontmatter`, or parsing logic anywhere in the codebase.
- **No lint package**: `internal/lint/` does not exist.
- **No path constants**: Directory names like `_inbox` and `notes` are inline strings in `VaultStructure()`.

</current_state>

<desired_state>

## Desired End State

- **Note type and frontmatter struct** in a new `internal/note/` package that can parse a markdown file with YAML frontmatter into a structured Go type.
- **Rule-based validation architecture** where each lint check is a named, discrete rule (e.g., `NF001`) that can be independently tested, referenced in error output, and extended in future increments.
- **Body section validation** that checks for required markdown sections (`# Title`, `## Evidence`, `## Implications`, `## Related`) with an exemption for synthesis notes.
- **`agent-memory lint-note <file>` subcommand** that runs all validation rules. Human-readable output by default; JSON output with `--json`. Exit 0 on pass, 1 on failure. Error messages reference rule IDs.
- **Writing protocol v2** embedded template replacing v1 content with full agent instructions for note format.
- **Unit tests** for frontmatter parsing and each individual lint rule covering valid notes, missing fields, invalid values, malformed YAML, missing body sections, and placeholder detection.
- **Integration tests** for the `lint-note` subcommand exercising the binary end-to-end. Existing integration tests reviewed for any assertions that need updating (e.g., `instructions` output mentioning "Writing to the vault is not yet enabled").

</desired_state>

<design_decisions>

## Design Decisions

<decision id="1">
**Decision**: Place note types in a new `internal/note/` package, not in `internal/vault/`.

**Rationale**: `internal/vault/` currently handles vault scaffolding (init, structure, templates). Note parsing and validation is a distinct concern — it operates on individual files, not on the vault as a whole. Separating it keeps `internal/vault/` focused and avoids a grab-bag package. Future increments (`memory-write`, `memory-search`) will import `internal/note/` without pulling in vault init logic.

This diverges from the design doc at §8.2 which shows `internal/vault/` as the home for "note struct, frontmatter parsing". The design doc defines behaviour and strategy; package layout is an implementation detail. Everything in `internal/note/` — types, parsing, and validation together. A separate `internal/lint/` package only becomes useful when `lint-vault` arrives (increment 8) and needs cross-file checks that don't belong in the note package.
</decision>

<decision id="2">
**Decision**: Use `go.yaml.in/yaml/v3` (the newer canonical import path), not `gopkg.in/yaml.v3`.

**Rationale**: Already an indirect dependency at v3.0.4. Same library, newer import path. The design doc references the old path but the go.mod already has the new one. Using the newer path avoids pulling in a second version of the same library.
</decision>

<decision id="3">
**Decision**: Frontmatter struct uses Go struct tags for YAML mapping, with validation handled by discrete lint rules. Date fields are `string` with explicit format validation.

**Rationale**: The YAML library handles deserialization via struct tags (`yaml:"field-name"`). Validation is a separate concern that runs after parsing via named rules. This keeps parsing and validation cleanly separated — parse errors are "your YAML is broken", validation errors reference specific rule IDs.

Date fields use `string` rather than `time.Time` because the YAML library's auto-parsing of date-like strings (YAML 1.1 date resolution) is notoriously surprising. Explicit string validation against `YYYY-MM-DD` format is more predictable.

**Struct shape**:
```go
type Frontmatter struct {
    Title              string   `yaml:"title"`
    Created            string   `yaml:"created"`
    Updated            string   `yaml:"updated"`
    ReviewBy           string   `yaml:"review-by"`
    Status             string   `yaml:"status"`
    Confidence         string   `yaml:"confidence"`
    EpistemicType      string   `yaml:"epistemic-type"`
    Scope              string   `yaml:"scope"`
    Project            string   `yaml:"project"`
    Domain             []string `yaml:"domain"`
    SourceAgent        string   `yaml:"source-agent"`
    SourceArtifact     string   `yaml:"source-artifact"`
    VerifiedBy         string   `yaml:"verified-by"`
    VerifiedDate       string   `yaml:"verified-date"`
    RequiresHumanReview bool    `yaml:"requires-human-review"`
    UpdateType         string   `yaml:"update-type"`
    Targets            []string `yaml:"targets"`
    Tags               []string `yaml:"tags"`
}
```
</decision>

<decision id="4">
**Decision**: Parse frontmatter by splitting on `---` delimiters, then YAML-unmarshal the frontmatter block.

**Rationale**: Standard markdown frontmatter convention. Split the file content on the first two `---` lines. The content between them is YAML. Everything after the second `---` is the markdown body. No need for a full markdown parser at this stage — simple string splitting works for frontmatter extraction.

**Function signature**:
```go
func Parse(content []byte) (*Note, error)
```

Where `Note` contains `Frontmatter` and `Body string`.
</decision>

<decision id="5">
**Decision**: `domain` field strictly requires a YAML list. `domain: golang` (bare string) is rejected.

**Rationale**: The design doc shows `domain: [golang, auth]` as the canonical form. Accepting both string and list requires custom unmarshaling and creates ambiguity. Agents should always write the list form.

This aligns with the principle that fields with statically assignable and verifiable values should be populated by tooling (like `memory-write`) rather than entered freehand by agents. Since tools will always write the list form, bare string rejection is safe. If the YAML library silently unmarshals a bare string into `[]string{"golang"}`, we accept that behavior rather than adding explicit rejection — the tool-generated form will always be correct, and the linter's job is to catch structural problems, not police YAML style.
</decision>

<decision id="6">
**Decision**: Unknown/extra frontmatter fields are ignored (lenient unmarshaling).

**Rationale**: Strict unmarshaling (`KnownFields(true)`) would reject any field not in the struct. This is fragile — if a future increment adds a new field, notes written with that field would fail validation on an older version of the tool. Lenient unmarshaling is more forward-compatible. The linter's job is to check that required fields are present and valid, not to police what else is in the frontmatter.

Typos in field names (e.g., `epistmic-type` instead of `epistemic-type`) would silently pass as an unknown field while the required field would be reported as missing. The error message "missing field: epistemic-type" is correct and actionable. Accepted trade-off.
</decision>

<decision id="7">
**Decision**: `project` field handling with `scope: cross-project` — allow it to be present but empty. Do not reject or warn on a non-empty `project` with `scope: cross-project`.

**Rationale**: The design doc says "cross-project notes leave this blank". This is guidance, not a hard constraint. A note might originate from a specific project but apply cross-project. Validation only enforces: if `scope: project`, then `project` must be non-empty. No warning on the reverse case — keep the error list clean for actual problems.
</decision>

<decision id="8">
**Decision**: Placeholder detection checks for these patterns in required string fields only:
- Empty string `""`
- Angle-bracket placeholders: any value matching `<.*>`
- Common placeholder words: `TODO`, `TBD`, `FIXME`, `xxx`, `placeholder` (case-insensitive)

**Rationale**: The design doc says "no frontmatter fields with placeholder values" but doesn't define what constitutes a placeholder. These patterns cover the most common cases. The angle-bracket pattern catches the design doc's own example (`<repo-relative path or URL>`).

Placeholder detection applies only to required fields. Optional fields like `verified-by` are legitimately empty (`""`) until a note is verified.
</decision>

<decision id="9">
**Decision**: Body section validation uses simple string matching — check that the markdown body contains `# ` followed by text (title heading), `## Evidence`, `## Implications`, and `## Related` as line prefixes.

**Rationale**: No need for a full markdown AST parser. The sections are well-defined headings. A line-prefix check is sufficient and fast. The design doc mentions Goldmark for wikilink extraction in `lint-vault`, but that's a future increment concern. For `lint-note`, simple string matching works.

**Synthesis exemption**: If `epistemic-type: synthesis`, check for `## Synthesis` and `## Contributing notes` instead of `## Evidence` and `## Implications`. `## Related` is still required. Per design doc §5.3.
</decision>

<decision id="10">
**Decision**: The `lint-note` subcommand is `agent-memory lint-note <file>`. It takes exactly one file path argument. It follows the existing dual-output pattern: human-readable by default, JSON with `--json`.

**Rationale**: Per ADR-0001, all tools are subcommands. Follows the same dual-output pattern as `init` — human-readable output by default for developer use, `--json` for machine consumption by tools like `memory-write` and `memory-promote`.

**Human-readable output** (default):
```
✓ note is valid
```
or:
```
✗ note has 3 errors:
  NF001: missing required field: confidence
  NF003: invalid epistemic-type: "foo"
  NF005: missing section: ## Evidence
```

**JSON output** (`--json`):
```json
{"valid": true}
{"valid": false, "errors": [{"rule": "NF001", "message": "missing required field: confidence"}, {"rule": "NF003", "message": "invalid epistemic-type: \"foo\""}, {"rule": "NF005", "message": "missing section: ## Evidence"}]}
```

Single file at a time — batch linting across all files is `agent-memory lint` (a future increment 8 concern).
</decision>

<decision id="11">
**Decision**: Writing protocol v2 replaces v1 content in the embedded template at `internal/vault/templates/writing-protocol.md`.

**Rationale**: The template is embedded and written during `vault init`. Existing vaults that were initialized with v1 will keep v1 until re-initialized. This is acceptable — the writing protocol is a reference document, not executable code. Agents read it at session start; they'll get v2 content on new vaults.

Init behavior is unchanged — it only creates missing files, it doesn't overwrite existing ones. If someone wants v2 on an existing vault, they can delete `_meta/writing-protocol.md` and re-run init (which will repair it with v2 content). This preserves idempotency.
</decision>

<decision id="12">
**Decision**: Validation uses a rule-based architecture where each check is a named, discrete rule with an ID, following the pattern established by tools like markdownlint.

**Rationale**: A rule-based architecture makes the linter extendable — new rules can be added in future increments without restructuring existing code. Each rule has an ID (e.g., `NF001`), a description, and a check function. Error messages reference the rule ID, making it easy to look up what a specific failure means and why it exists.

**Rule architecture**:
```go
// Rule represents a single lint check.
type Rule struct {
    ID          string
    Description string
    Check       func(note *Note) []string // returns error messages, empty = pass
}
```

A `Lint(note *Note) *LintResult` function runs all registered rules and collects results. Each error in the result includes the rule ID and message.

**Initial rule set for increment 2**:

| Rule ID | Description |
|---------|-------------|
| NF001 | Required frontmatter field present and non-empty |
| NF002 | Valid date format (YYYY-MM-DD) for created, updated, review-by |
| NF003 | Valid enum value for status, epistemic-type, confidence, scope |
| NF004 | Project required when scope is project |
| NF005 | Required body sections present |
| NF006 | No placeholder values in required fields |
| NF007 | Domain field is a non-empty list |

Rules are registered in a slice, iterated in order. No enable/disable mechanism in this increment — all rules always run. Enable/disable can be added later if needed.

**LintResult shape**:
```go
type LintError struct {
    Rule    string `json:"rule"`
    Message string `json:"message"`
}

type LintResult struct {
    Valid  bool        `json:"valid"`
    Errors []LintError `json:"errors,omitempty"`
}
```
</decision>

</design_decisions>

<data_flow>

## Data Flow

```
File on disk → Read bytes → Split on --- delimiters → YAML unmarshal frontmatter → Run lint rules → Return LintResult
```

For the CLI subcommand:
```
agent-memory lint-note <file> → Read file → note.Parse() → note.Lint() → human-readable or JSON result to stdout → exit 0 or 1
```

Key types:
- `note.Note` — contains `Frontmatter` and `Body string`
- `note.Frontmatter` — struct with all fields
- `note.Rule` — `ID string`, `Description string`, `Check func(*Note) []string`
- `note.LintError` — `Rule string`, `Message string`
- `note.LintResult` — `Valid bool`, `Errors []LintError`
- `note.Parse(content []byte) (*Note, error)` — splits frontmatter from body, unmarshals YAML
- `note.Lint(note *Note) *LintResult` — runs all rules, collects errors

</data_flow>

<resolved_questions>

## Resolved Questions

1. **Date fields as string vs time.Time**: String with explicit format validation.
2. **Domain bare string behavior**: Accept whatever the YAML library does; tools will always write list form.
3. **Placeholder detection scope**: Required fields only.
4. **lint-note output**: Dual output (human-readable default, JSON with `--json`), same pattern as `init`.
5. **Writing protocol upgrade path**: Preserve idempotency, no overwrite on existing vaults.
6. **Warn on project + cross-project?**: Silent, no warning.
7. **Unknown fields**: Lenient unmarshaling, accepted trade-off on typo detection.
8. **Package placement**: `internal/note/`, diverging from design doc (implementation detail).
9. **Existing integration tests**: Need review — `instructions` test asserts "Writing to the vault is not yet enabled" which may need updating.
10. **Validation architecture**: Rule-based with named IDs (NF001-NF007), extendable for future increments.

</resolved_questions>

</design_artifact>
