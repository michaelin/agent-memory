<questions_artifact feature="note-format-frontmatter">

<feature>
**Name:** note-format-frontmatter
**Description:** Define and validate the note structure (YAML frontmatter + markdown body) so agents can write notes that tools can parse. Includes a `lint-note` binary and writing protocol v2.
</feature>

<questions>

<category name="data-model">
## Data Model Questions

<question>
What types and structs already exist in `internal/vault/` — particularly around vault structure, directory constants, and embedded templates? How are the `_meta/` seed files currently defined and embedded?
<why>Forces exploration of: internal/vault/structure.go, internal/vault/init.go — need to understand existing type patterns and embed usage before adding note/frontmatter types</why>
</question>

<question>
Does the codebase already use `go.yaml.in/yaml/v3` (listed as indirect dependency) anywhere, or is it only pulled in transitively? Are there any existing YAML parsing patterns?
<why>Forces exploration of: go.mod, go.sum, and grep for yaml usage — determines whether we introduce a new direct dependency or follow an existing pattern</why>
</question>

<question>
How does the existing vault structure define the `_inbox/` and `notes/` directories? Are there any path-building helpers or constants for these locations?
<why>Forces exploration of: internal/vault/structure.go — the lint-note binary needs to know where notes live and how paths are constructed</why>
</question>
</category>

<category name="api">
## CLI / Binary Questions

<question>
How is the existing `agent-memory` CLI structured with cobra? What's the pattern for adding new subcommands? Is `lint-note` intended as a separate binary or a subcommand of `agent-memory`?
<why>Forces exploration of: cmd/agent-memory/main.go, internal/cli/root.go, internal/cli/init.go — need to understand the CLI wiring pattern before adding lint-note</why>
</question>

<question>
What's the pattern for CLI output in the existing codebase? Does it use structured JSON output, plain text, or both? How are errors reported to the user?
<why>Forces exploration of: internal/cli/*.go — lint-note needs JSON output (`{"valid": true/false, "errors": [...]}`) and needs to follow existing conventions</why>
</question>

<question>
How does the `instructions` subcommand work? Does it read from embedded files or generate content dynamically? This is relevant because writing-protocol v2 will be an embedded template.
<why>Forces exploration of: internal/cli/instructions.go — understand the pattern for serving embedded content to agents</why>
</question>
</category>

<category name="integration">
## Integration Questions

<question>
How are the existing integration tests structured? What test helpers exist in `internal/testutil/`? How do tests create and tear down temporary vaults?
<why>Forces exploration of: test/integration/init_test.go, internal/testutil/helpers.go, internal/testutil/reporter.go — lint-note tests need temporary vaults with test notes</why>
</question>

<question>
How are unit tests structured in `internal/vault/`? Do they use ginkgo/gomega Describe/It blocks? What's the convention for table-driven vs. individual test cases?
<why>Forces exploration of: internal/vault/init_test.go, internal/vault/structure_test.go — frontmatter parsing will need extensive validation tests</why>
</question>

<question>
What does the existing writing-protocol.md template contain (the v1 version seeded by init)? How is it embedded and written during vault init?
<why>Forces exploration of: internal/vault/structure.go or embedded templates — v2 replaces this content, need to understand the upgrade/replacement mechanism</why>
</question>
</category>

<category name="edge-cases">
## Edge Case Questions

<question>
How should the frontmatter parser handle the `project` field when `scope: cross-project`? The ticket says project is "required if scope: project" — should it be rejected if present with cross-project scope, or just ignored?
<why>Forces exploration of: the ticket and design doc for conditional validation rules — this is an ambiguity that affects validation logic</why>
</question>

<question>
What constitutes a "placeholder value" that lint-note should detect? Are there specific placeholder patterns (e.g., `TODO`, `TBD`, `<placeholder>`, empty strings) defined anywhere?
<why>Forces exploration of: AGENT_MEMORY_DESIGN.md and ROADMAP.md — placeholder detection needs concrete patterns to check against</why>
</question>

<question>
How should lint-note handle notes that have extra/unknown frontmatter fields beyond the required and optional sets? Should it warn, ignore, or reject them?
<why>Forces exploration of: design doc for strictness policy — affects whether the YAML struct uses strict or lenient unmarshaling</why>
</question>

<question>
The ticket specifies `gopkg.in/yaml.v3` but go.mod already has `go.yaml.in/yaml/v3`. Are these the same library under different import paths, or different libraries? Which should be used?
<why>Forces exploration of: go.mod, go.sum — need to resolve the correct YAML library before implementation</why>
</question>

<question>
Should the `domain` field accept a single string as well as a list, or strictly require a YAML list? YAML allows both `domain: golang` and `domain: [golang]` — what's the canonical form?
<why>Forces exploration of: design doc and ticket — affects YAML struct tag and custom unmarshaling logic</why>
</question>
</category>

</questions>

</questions_artifact>
