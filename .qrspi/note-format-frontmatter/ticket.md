# Increment 2: Note Format & Frontmatter Parsing

**Goal:** Define and validate the note structure so agents can write notes that tools can parse.

**What becomes possible after this increment:**
- Agents understand the note format and frontmatter requirements
- Notes can be validated for structural correctness
- The writing protocol document explains what agents should do

**Scope:**

### Note Format Specification
- Frontmatter (YAML) + body (markdown)
- Required frontmatter fields:
  ```yaml
  title: "Concise factual claim"
  created: 2026-04-24
  updated: 2026-04-24
  status: inbox
  epistemic-type: observation
  confidence: medium
  scope: project
  project: project-slug
  domain: [golang]
  source-agent: research/analyst-a
  source-artifact: "<repo-relative path or URL>"
  ```
- Optional fields (populated by tools, not agents):
  ```yaml
  review-by: ""
  verified-by: ""
  verified-date: ""
  requires-human-review: false
  update-type: ""
  targets: []
  tags: []
  ```
- Body sections (required):
  - `# Title` (repeats frontmatter title)
  - `## Evidence` (where this was observed)
  - `## Implications` (what other agents need to know)
  - `## Related` (wikilinks to related notes)

### Frontmatter Parsing
- YAML parsing (`gopkg.in/yaml.v3`)
- Field validation rules:
  - `title`: non-empty string
  - `created`, `updated`: ISO date format (YYYY-MM-DD)
  - `status`: one of `inbox`, `verified`, `deprecated`, `contested`, `superseded`
  - `epistemic-type`: one of `observation`, `pattern`, `constraint`, `decision`, `assumption`, `synthesis`
  - `confidence`: one of `low`, `medium`, `high`
  - `scope`: one of `project`, `cross-project`
  - `project`: non-empty string (required if `scope: project`)
  - `domain`: list of strings
  - `source-agent`: non-empty string
  - `source-artifact`: non-empty string
- Malformed YAML is rejected with clear error message
- Missing required fields are rejected with clear error message

### `lint-note` Binary
- Single-file validation
- Checks all frontmatter fields are present and valid
- Checks all required body sections are present
- Checks no placeholder values remain
- Exit 0 on pass, 1 on failure
- JSON output: `{"valid": true}` or `{"valid": false, "errors": ["...", "..."]}`

### Writing Protocol (v2 -- Agent Instructions)
- Document at `_meta/writing-protocol.md`
- Content: Full agent instructions for note format, frontmatter fields, body sections, and validation

### Verification (Integration Tests)
- **Valid note:** Create a note with all required fields, run `lint-note`, verify pass
- **Missing field:** Create a note missing a required field, run `lint-note`, verify fail with specific error
- **Invalid value:** Create a note with invalid `epistemic-type`, run `lint-note`, verify fail
- **Malformed YAML:** Create a note with broken YAML, run `lint-note`, verify fail
- **Body sections:** Create a note missing `## Evidence`, run `lint-note`, verify fail
- **Placeholder values:** Create a note with placeholder text, run `lint-note`, verify fail
- **Agent verification test:** Have an agent read the writing protocol and write a valid note, verify it passes `lint-note`

## Design Reference
- See AGENT_MEMORY_DESIGN.md sections on note format and frontmatter
- See Increment 1 vault-scaffolding for existing vault structure
