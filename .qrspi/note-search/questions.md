<questions_artifact feature="note-search">

<feature>
**Name:** note-search
**Description:** Implement `agent-memory search` subcommand with two-phase retrieval: Phase 1 returns frontmatter-only matches, Phase 2 returns full bodies for selected slugs (capped at 10).
</feature>

<questions>

<category name="data-model">
## Data Model Questions

<question>
What fields does `note.Frontmatter` currently expose, and which are exported? The search subcommand needs to filter on `status`, `project`, `domain`, and `epistemic-type` — are these accessible as typed fields or raw strings?
<why>Forces exploration of: internal/note/note.go — the Frontmatter struct and its field types</why>
</question>

<question>
How does `note.Parse()` handle the body vs. frontmatter split? Does it return the body as a single string, or is further processing needed to extract it for Phase 2?
<why>Forces exploration of: internal/note/note.go — Parse() return type and body handling</why>
</question>

<question>
What is the current structure of `_meta/tag-taxonomy.md`? The ticket mentions tag alias expansion — what format are aliases stored in, and does any parsing code exist?
<why>Forces exploration of: internal/vault/templates/tag-taxonomy.md (embedded template) and any existing tag parsing</why>
</question>

<question>
How are notes stored on disk? Are all verified notes in `notes/` flat, or is there subdirectory nesting? Are inbox notes in `_inbox/` with the same naming convention?
<why>Forces exploration of: internal/vault/init.go (directory structure), internal/note/write.go (file naming)</why>
</question>
</category>

<category name="cli">
## CLI Questions

<question>
What is the established pattern for subcommand flag parsing and validation? How does `write-note` handle mutually exclusive flags or required flag combinations (e.g., `--phase=2` requires `--slugs`)?
<why>Forces exploration of: internal/cli/write_note.go — flag registration and RunE validation patterns</why>
</question>

<question>
How does the existing `--json` persistent flag work? Does each subcommand check it manually, or is there a shared output helper?
<why>Forces exploration of: internal/cli/root.go — persistent flag registration and how subcommands access it</why>
</question>

<question>
What is the human-readable output pattern? Does `write-note` have a human-readable mode, or is it JSON-only? What should search results look like for humans?
<why>Forces exploration of: internal/cli/write_note.go — output formatting for non-JSON mode</why>
</question>
</category>

<category name="search-logic">
## Search Logic Questions

<question>
How should the `<query>` positional argument work? Is it a substring match on title? A token match? Does it search body text too, or frontmatter only in Phase 1?
<why>Forces exploration of: design doc §4.3 (two-phase retrieval) and the roadmap's "grep + YAML parsing" description</why>
</question>

<question>
What does "tag alias expansion" mean concretely? If a user searches for domain "auth", should it also match notes tagged with "authentication"? Where is the alias mapping defined?
<why>Forces exploration of: _meta/tag-taxonomy.md template, design doc §9.3 search description</why>
</question>

<question>
Should Phase 1 search both `notes/` and `_inbox/`? The design doc says "status: verified only" which implies only `notes/`, but should there be a flag to include inbox?
<why>Forces exploration of: design doc §5.4 (memory tiers), vault directory structure</why>
</question>

<question>
How should slug matching work for Phase 2? Is it an exact match on the filename stem, or does it need fuzzy matching? What happens if a slug doesn't exist?
<why>Forces exploration of: internal/note/serialize.go — Slug() function, file naming conventions in write.go</why>
</question>
</category>

<category name="integration">
## Integration Questions

<question>
Does `internal/vault/discover.go` return a path that can be used directly to scan for notes? What type does `Discover()` return and what error cases does it handle?
<why>Forces exploration of: internal/vault/discover.go — Discover() signature and return type</why>
</question>

<question>
How does `write-note` locate the vault? Does it call `Discover()` directly, or is there a shared setup in the cobra command chain?
<why>Forces exploration of: internal/cli/write_note.go — vault path resolution pattern</why>
</question>

<question>
Are there any existing file-scanning utilities? Does any code already walk `notes/` or `_inbox/` and parse all notes?
<why>Forces exploration of: internal/note/ and internal/vault/ for any existing directory-walking code</why>
</question>
</category>

<category name="edge-cases">
## Edge Case Questions

<question>
What happens when the vault has no notes at all? Should `search` return an empty result, or a specific message?
<why>Forces exploration of: how write-note and other subcommands handle empty/missing vault state</why>
</question>

<question>
What if a note file in `notes/` has malformed frontmatter? Should search skip it silently, warn, or fail?
<why>Forces exploration of: internal/note/note.go — Parse() error handling, and the lenient unmarshaling decision from increment 2</why>
</question>

<question>
What is the performance profile? If a vault has hundreds of notes, is scanning all files on every search acceptable, or does the design expect an index?
<why>Forces exploration of: design doc §4.3 and §7 (reindex), whether search depends on pre-built indices</why>
</question>

<question>
How should the 10-note cap in Phase 2 behave? Hard error if >10 slugs provided? Silent truncation? Warning?
<why>Forces exploration of: design doc §9.3 search description for cap enforcement semantics</why>
</question>

<question>
Should search results include notes with `status: deprecated` or `status: superseded`? The design says "verified only" but there may be cases where agents need to find deprecated notes.
<why>Forces exploration of: design doc §5.5 (write protocol), status lifecycle in _meta/status-lifecycle.md</why>
</question>
</category>

</questions>

</questions_artifact>
