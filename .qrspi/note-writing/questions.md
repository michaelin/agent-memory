<questions_artifact feature="note-writing">

<feature>
**Name:** note-writing
**Description:** Implement `agent-memory write-note` subcommand so agents can write validated notes to the vault's `_inbox/`.
</feature>

<questions>

<category name="data-model">
## Data Model Questions

<question>
How does the existing `note.Parse()` function handle round-tripping — if we parse a note and re-serialize it, does the YAML library preserve field order, comments, or formatting? Or do we need a separate serialization path for writing notes?
<why>Forces exploration of: `internal/note/note.go`, `go.yaml.in/yaml/v3` marshaling behavior, whether `Frontmatter` struct tags support both marshal and unmarshal</why>
</question>

<question>
What fields in the `Frontmatter` struct should be auto-populated by the write tool vs. provided by the caller? The ticket mentions "created, updated, status, TTL assignment" as tool-assembled. What about `review-by`, `requires-human-review`, `update-type`, `targets`, `tags`?
<why>Forces exploration of: `internal/note/note.go` Frontmatter struct, design doc §5.3 field descriptions, the distinction between required and optional fields in NF001</why>
</question>

<question>
What is the current log entry format in `_meta/log.md`? The init command writes log entries — what's the exact format, and should write-note follow the same pattern?
<why>Forces exploration of: `internal/vault/init.go` log writing code, `_meta/log.md` format after init</why>
</question>

<question>
Does the vault package expose any function for resolving the vault path, or is that always passed explicitly? The write-note command needs to know where the vault is to find `_inbox/` and `_meta/log.md`.
<why>Forces exploration of: `internal/vault/init.go`, `internal/cli/init.go`, vault discovery logic (or lack thereof)</why>
</question>

</category>

<category name="api">
## CLI Interface Questions

<question>
How does the existing `init` subcommand handle its `--json` output for success vs. error cases? What JSON shape does it return? The write-note command needs to follow the same pattern.
<why>Forces exploration of: `internal/cli/init.go` RunE function, JSON output structure, error handling pattern</why>
</question>

<question>
How should the note content be provided to `write-note` — as a file path argument (like `lint-note`), via stdin, or as structured flags/arguments for individual fields? What does the design doc suggest?
<why>Forces exploration of: `internal/cli/lint_note.go` argument handling, design doc §8.5, cobra argument patterns in the codebase</why>
</question>

<question>
What exit codes does the codebase use? Is exit 1 always "error" or are there distinctions (e.g., exit 1 for validation failure, exit 2 for I/O error)?
<why>Forces exploration of: `internal/cli/init.go`, `internal/cli/lint_note.go`, cobra error handling</why>
</question>

</category>

<category name="integration">
## Integration Questions

<question>
How does the vault's `_inbox/` directory get created? Is it guaranteed to exist after `vault init`? What happens if someone runs `write-note` on a vault that hasn't been initialized?
<why>Forces exploration of: `internal/vault/init.go`, `VaultStructure()`, whether write-note should check vault existence first</why>
</question>

<question>
The ticket mentions wikilink validation (warn on unresolved links). What does a wikilink look like in the note body? Are there any existing wikilink parsing utilities, or does this need to be built from scratch?
<why>Forces exploration of: design doc §5.3 wikilink format, any existing markdown parsing in the codebase, `internal/note/lint.go` body section checking</why>
</question>

<question>
The ticket mentions similarity check using Jaccard on normalized word tokens. Are there any existing text processing utilities in the codebase? What notes would be compared against — only `_inbox/` or also `notes/`?
<why>Forces exploration of: any text/string utilities, design doc similarity check description, vault directory structure</why>
</question>

<question>
How does the existing `lint-note` validation integrate? Should `write-note` call `note.Lint()` internally before writing, or is validation the caller's responsibility?
<why>Forces exploration of: `internal/note/lint.go` Lint() function, whether it's designed for programmatic use or just CLI</why>
</question>

</category>

<category name="edge-cases">
## Edge Case Questions

<question>
What happens if two agents try to write notes with the same slug at the same time? Is there any file locking or conflict detection in the codebase?
<why>Forces exploration of: file I/O patterns in `internal/vault/init.go`, any concurrency handling, os.Create vs os.OpenFile flags</why>
</question>

<question>
How should slug generation work? The ticket says `{YYYY-MM-DD}-{slug}.md` — what characters are allowed in the slug? How is the slug derived from the title? What if the title produces a slug that collides with an existing file?
<why>Forces exploration of: any existing slug/filename generation utilities, filesystem naming constraints</why>
</question>

<question>
What happens if the vault's `_meta/log.md` doesn't exist or is corrupted? Should write-note create it, skip logging, or error?
<why>Forces exploration of: `internal/vault/init.go` log writing, file append patterns, error handling for missing meta files</why>
</question>

<question>
The ticket mentions TTL assignment. Where is TTL defined in the note format? Is it a frontmatter field? The Frontmatter struct in increment 2 doesn't have a TTL field.
<why>Forces exploration of: `internal/note/note.go` Frontmatter struct, design doc §5.3, roadmap increment 5 (promotion & lifecycle) for TTL context</why>
</question>

<question>
Should `write-note` refuse to overwrite an existing file at the target path, or should it have a `--force` flag like `init`?
<why>Forces exploration of: `internal/vault/init.go` force flag handling, file creation patterns</why>
</question>

</category>

</questions>

</questions_artifact>
