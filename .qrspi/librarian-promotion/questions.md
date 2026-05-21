<questions_artifact feature="librarian-promotion">

<feature>
**Name:** librarian-promotion
**Description:** Implement the Librarian as a skill-triggered agent and the `agent-memory promote` subcommand that moves validated notes from `_inbox/` to `notes/`.
</feature>

<questions>

<category name="data-model">
## Data Model Questions

<question>
What fields in `internal/note/note.go`'s `Frontmatter` struct are relevant to promotion? Specifically: is `Status` already a typed field with the `verified`/`deprecated`/`superseded` values, or is it a raw string? Are `RequiresHumanReview`, `UpdateType`, and `Targets` already defined?
<why>Forces exploration of: `internal/note/note.go` — need to know if the struct already supports the fields the promote subcommand needs to read and write, or if new fields must be added.</why>
</question>

<question>
How does `internal/note/serialize.go`'s `Serialize()` function handle round-tripping? If we parse a note, change `status` from `inbox` to `verified`, and re-serialize, does the output preserve the original field order, comments, and body content exactly?
<why>Forces exploration of: `internal/note/serialize.go` — promotion must update frontmatter without corrupting the note body or losing formatting.</why>
</question>

<question>
What does the `Slug()` function in `internal/note/serialize.go` produce, and does it match the filename convention used by `Write()` in `internal/note/write.go`? The promote subcommand needs to look up notes by slug.
<why>Forces exploration of: `internal/note/serialize.go` and `internal/note/write.go` — need to understand the slug-to-filename mapping for both `_inbox/` and `notes/`.</why>
</question>

<question>
What is the current filename format for notes in `_inbox/`? Is it `{YYYY-MM-DD}-{slug}.md` as the design doc says? Does the promote subcommand need to rename the file when moving to `notes/`, or keep the same name?
<why>Forces exploration of: `internal/note/write.go` (`Write()` function) — need to understand the file naming convention to implement the move correctly.</why>
</question>

</category>

<category name="cli">
## CLI & Subcommand Questions

<question>
How are existing subcommands structured in `internal/cli/`? What is the pattern for adding a new subcommand — does each get its own file? How are flags registered? How is the `--json` output flag handled?
<why>Forces exploration of: `internal/cli/root.go`, `internal/cli/write_note.go`, and any other CLI files — need to follow the established pattern for the new `promote` subcommand.</why>
</question>

<question>
How does the existing `write-note` subcommand handle vault discovery? Is there a shared helper, or does each subcommand call `vault.Discover()` independently?
<why>Forces exploration of: `internal/cli/write_note.go` and `internal/vault/discover.go` — the promote subcommand needs vault discovery too, and we should reuse the existing pattern.</why>
</question>

<question>
What does the log-writing pattern look like? The `Write()` function in `internal/note/write.go` presumably appends to `_meta/log.md`. What format does it use, and is there a shared log-writing helper?
<why>Forces exploration of: `internal/note/write.go` and any log-writing code — promote needs to write its own log entries in the same format.</why>
</question>

</category>

<category name="integration">
## Integration & Wikilink Questions

<question>
How does `ExtractWikilinks()` in `internal/note/wikilinks.go` work? Does it return resolved vs. unresolved status, or just the raw link targets? How would the promote subcommand check whether all wikilinks in a note resolve to existing files?
<why>Forces exploration of: `internal/note/wikilinks.go` — promote must block on unresolved wikilinks, so we need to understand what the existing extraction gives us and what resolution logic is needed.</why>
</question>

<question>
How does `findSimilarNotes()` in `internal/note/write.go` work? What does it scan, what threshold does it use, and could it be extracted into a shared utility for the Librarian's deduplication check?
<why>Forces exploration of: `internal/note/write.go` (`findSimilarNotes()`) and `internal/note/similarity.go` — the Librarian needs similarity checking against `notes/` (not just `_inbox/`), so we need to understand if the existing code can be reused.</why>
</question>

<question>
What does `internal/note/lint.go`'s `Lint()` function return? What are the current lint rules (NF001-NF007)? Can the promote subcommand call `Lint()` directly, or does it need the CLI `lint-note` binary?
<why>Forces exploration of: `internal/note/lint.go` — promote should call the lint function programmatically, not shell out to the binary.</why>
</question>

</category>

<category name="skill-definition">
## Librarian Skill Questions

<question>
What is the structure of existing skills in this project or in `~/.config/opencode/skills/`? Is there a SKILL.md template or convention the Librarian skill should follow?
<why>Forces exploration of: any existing skill definitions — need to understand the format, permissions model, and tool declarations for the Librarian skill.</why>
</question>

<question>
How does the `agent-memory instructions` output currently describe the workflow? Does it mention the Librarian or promotion at all, or does it stop at "notes land in _inbox"?
<why>Forces exploration of: the instructions template (likely in `internal/vault/templates/` or the `instructions` subcommand) — the instructions need to tell agents to invoke the Librarian skill after writing notes.</why>
</question>

<question>
What tool-use declarations does a skill need to call `agent-memory promote`, `agent-memory lint-note`, and read/write vault files? How are bash tool permissions scoped in a skill definition?
<why>Forces exploration of: skill definition format and permissions model — the Librarian skill needs specific tool access.</why>
</question>

</category>

<category name="edge-cases">
## Edge Case Questions

<question>
What happens if `_inbox/` contains a note whose slug collides with an existing file in `notes/`? Is this a deduplication case, a replacement case, or an error?
<why>Forces exploration of: the file move logic — need to define behavior for slug collisions during promotion.</why>
</question>

<question>
How should the promote subcommand handle a note that references another inbox note via wikilink (not yet promoted)? Is this an unresolved link that blocks promotion, or should inbox-to-inbox links be allowed?
<why>Forces exploration of: wikilink resolution logic — need to define the resolution scope (just `notes/`? or `notes/` + `_inbox/`?).</why>
</question>

<question>
What is the deprecation mechanism? When the Librarian deprecates an old note (sets `status: deprecated` with a forward link), does it modify the old note's frontmatter in place? Does it add a `superseded-by` field? Is this field already in the `Frontmatter` struct?
<why>Forces exploration of: `internal/note/note.go` frontmatter struct and the design doc's deprecation protocol — need to know what fields exist and what needs to be added.</why>
</question>

<question>
How does the `--confirmed` flag interact with the Librarian skill? The skill is an LLM agent — it asks the human, gets confirmation, then calls `promote --confirmed`. But what if the binary is called without `--confirmed` on a constraint note? Does it error, or does it set `requires-human-review: true` and exit?
<why>Forces exploration of: the design doc section 9.4 and the constraint/decision promotion flow — need to define the exact CLI behavior for the confirmation gate.</why>
</question>

<question>
Are there any existing tests that exercise note file movement (rename/move between directories)? What test utilities exist in `internal/testutil/`?
<why>Forces exploration of: `internal/testutil/` and existing test files — need to understand what test helpers are available for integration tests that move files between `_inbox/` and `notes/`.</why>
</question>

</category>

</questions>

</questions_artifact>
