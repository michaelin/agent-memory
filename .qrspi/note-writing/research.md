<research_artifact feature="note-writing">

<codebase_map>

## Codebase Map

### Packages
- `internal/note/` — `Note`, `Frontmatter` types, `Parse()`, `Lint()`, `Rules()`, `IsPlaceholder()`, `IsValidDate()`. 7 lint rules NF001–NF007. No serialization/marshaling functions exist.
- `internal/vault/` — `Init()`, `InitOptions`, `InitResult`, `InitError`, `MarshalError()`, `VaultStructure()`, `VaultEntry`, embedded `templateFS`. Handles vault creation, repair, log writing.
- `internal/cli/` — `NewRootCmd()`, `newInitCmd()`, `newLintNoteCmd()`, `newInstructionsCmd()`. Package-level `jsonOutput bool` for `--json` flag. Cobra-based.
- `internal/testutil/` — `BuildBinary()`, `RunBinary()`, `MustStat()`, `WriteTestNote()`, `findModuleRoot()`.
- `cmd/agent-memory/` — main entry point.

### Vault Structure (from `VaultStructure()`)
Directories: `_meta`, `_inbox`, `_contested`, `notes`
Files: `_meta/writing-protocol.md`, `_meta/tag-taxonomy.md`, `_meta/constraints-summary.md`, `_meta/status-lifecycle.md`, `_meta/log.md`

### Log Format
In `init.go` line 163: `\n## [%s] init | vault initialized\n` where `%s` is `time.Now().Format(time.RFC3339)`.
The design doc §9 specifies write log format as: `## [YYYY-MM-DD HH:MM] write | <type> | <slug> | by:<agent>`.

### File I/O Patterns
- Init uses `os.MkdirAll`, `os.WriteFile`, `os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)` for log.
- No file locking anywhere in the codebase.
- No slug generation utilities exist.
- No text similarity/Jaccard utilities exist.

### CLI Patterns
- `--json` is a persistent flag on root command, stored in package-level `jsonOutput` var.
- Init returns `InitResult` struct as JSON on success, `InitError` on failure.
- Lint-note returns `LintResult` struct as JSON.
- Error paths: JSON mode prints JSON to stdout, then returns error (cobra prints to stderr). `SilenceUsage` and `SilenceErrors` set before returning errors in JSON mode.
- Exit codes: 0 = success, 1 = error (no distinction between validation failure and I/O error).

### Frontmatter Struct
18 fields total. YAML struct tags support both marshal and unmarshal (same tag). `go.yaml.in/yaml/v3` handles both directions. No TTL field exists in the struct.

### Writing Protocol v2
Currently at `internal/vault/templates/writing-protocol.md`. **Contains wrong enum values**: status shows `draft, active, archived, deprecated` and epistemic-type shows `observation, inference, synthesis, hypothesis, procedure` and scope shows `project, global`. These don't match the lint rules (which use the correct spec values). This is a bug from the initial implementation that wasn't caught by the QA fix.

</codebase_map>

<question_answers>

## Question Answers

### Data Model

**Q: Does Parse() support round-tripping?**
A: The `Frontmatter` struct has `yaml:"..."` tags that work for both marshal and unmarshal. `yaml.Marshal()` would produce valid YAML from a `Frontmatter` value. However, field order in marshaled output follows struct field order, not original YAML order. No round-trip function exists — only `Parse()` (unmarshal direction). The `Body` field is a plain string, so body round-tripping is trivial.

**Q: Which fields should be auto-populated vs. caller-provided?**
A: Design doc §9 is explicit: the tool fills `created`, `updated`, `status: inbox`, `confidence` (defaults to `medium` if unset), `source-agent` (from environment), `verified-by`/`verified-date` (blank), `tags` (blank), and `review-by` from per-type TTL table. The agent provides: `epistemic-type`, claim body, `## Evidence`, `source-artifact`, and one of `--new-claim`/`--update`/`--contest`. The agent also provides `scope`, `project`, `domain`, and `title`.

**Q: Log entry format?**
A: Init uses `## [RFC3339] init | vault initialized`. Design doc specifies write format as `## [YYYY-MM-DD HH:MM] write | <type> | <slug> | by:<agent>`. These are different timestamp formats — init uses RFC3339 (with timezone), design doc uses shorter format.

**Q: Vault path resolution?**
A: No vault discovery function exists. Init takes an explicit path argument or defaults to `$CWD/.agent-memory`. The `init.go` CLI resolves the path. No shared vault resolution utility. Design doc §5.1 describes future discovery: env var → walk up → global fallback. Not implemented yet.

### CLI Interface

**Q: Init JSON output shape?**
A: Success: `{"status":"created","vault":"/abs/path"}` (or `"ok"` or `"repaired"` with `repaired` array). Error: `{"error":"message"}`. Both printed to stdout.

**Q: How should note content be provided?**
A: Design doc §9 shows `memory-write` taking flags: `--new-claim`, `--update=<slug>`, `--contest=<slug>`, `--source-artifact=<path>`, plus the claim body. The body content delivery mechanism isn't specified — could be stdin, a file argument, or a temp file. `lint-note` takes a file path argument. No stdin reading pattern exists in the codebase.

**Q: Exit codes?**
A: Only 0 and 1 are used. Cobra returns exit 1 for any error. No distinction between validation and I/O errors.

### Integration

**Q: Is `_inbox/` guaranteed to exist?**
A: Yes, `VaultStructure()` includes `{Path: "_inbox", IsDir: true}`. After `vault init`, `_inbox/` exists. If someone runs write-note without init, `_inbox/` won't exist.

**Q: Wikilink format and parsing?**
A: Design doc uses `[[note-slug]]` format (double brackets). No wikilink parsing exists in the codebase. The design doc §9 says `memory-write` flags unresolved links as warnings but doesn't block. `lint-vault` (increment 8) does the full wikilink resolution. Goldmark is mentioned for future use.

**Q: Similarity check scope?**
A: Design doc §9 step 1: "Scan `notes/` and `_inbox/` for notes with overlapping project/domain and a high title-similarity score (Jaccard on normalised word tokens)." Both directories are scanned.

**Q: Should write-note call Lint() internally?**
A: Design doc §9 step 3: "Invoke `lint-note` against the assembled note. Refuse on failure." Yes, the tool validates before writing.

### Edge Cases

**Q: Concurrent writes?**
A: No file locking exists. `os.WriteFile` and `os.OpenFile` are the only file I/O patterns. No concurrency protection.

**Q: Slug generation?**
A: No slug utilities exist. Design doc shows `_inbox/{YYYY-MM-DD}-{slug}.md`. No specification of how slug is derived from title. No collision handling specified.

**Q: Missing log.md?**
A: Init creates `_meta/log.md` from template. Init uses `os.O_CREATE` flag so it creates if missing. No explicit handling for corrupted log.

**Q: TTL field?**
A: No TTL field in `Frontmatter` struct. Design doc §5.6 mentions per-type TTL table but the TTL is expressed via `review-by` date, not a separate field. The tool calculates `review-by = created + TTL` based on epistemic type.

**Q: Overwrite protection?**
A: No `--force` flag pattern for write operations. Init has `--force` for overwrite. No specification in design doc for what happens on slug collision.

</question_answers>

<patterns>

## Observed Patterns

1. **Result struct pattern**: Each command returns a typed result struct (`InitResult`, `LintResult`) that serializes to JSON. Error structs are separate (`InitError`).
2. **Dual output in RunE**: Check `jsonOutput`, marshal to JSON and print, or print human-readable. Error paths print JSON first, then return error.
3. **Vault operations in `internal/vault/`**: Init is the only vault operation. It takes a path and options, returns result or error. No shared vault utilities beyond structure definition.
4. **Note operations in `internal/note/`**: Parse and Lint are the only operations. Both are pure functions (no I/O). The CLI does the I/O.
5. **Log writing**: Append-only via `os.OpenFile` with `O_APPEND|O_WRONLY|O_CREATE`.
6. **Template embedding**: `embed.FS` for vault templates. Templates are written during init only.

</patterns>

<gaps>

## Gaps

1. **No vault discovery**: No function to find the vault from an arbitrary working directory. Write-note needs to know where the vault is.
2. **No slug generation**: No utility to convert a title to a filesystem-safe slug.
3. **No similarity/Jaccard**: No text comparison utilities exist.
4. **No wikilink parsing**: No `[[slug]]` extraction from markdown body.
5. **No note serialization**: `Parse()` exists but no `Serialize()` or `Marshal()` to produce a complete note file from a `Note` struct.
6. **No vault existence check**: No function to verify a path is a valid vault (has the expected structure).
7. **No file collision handling**: No pattern for handling filename conflicts.
8. **Writing protocol v2 has wrong enum values**: The template at `internal/vault/templates/writing-protocol.md` lines 24-27 still show the pre-QA-fix values (`draft/active/archived/deprecated`, `observation/inference/synthesis/hypothesis/procedure`, `project/global`). The lint rules use the correct values. This is a discrepancy.

</gaps>

</research_artifact>
