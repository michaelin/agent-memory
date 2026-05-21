# Writing Protocol v3

This vault stores persistent knowledge for AI agents working in this project.

## Directory Structure

- `_meta/` — vault metadata, rules, and configuration
- `_inbox/` — unprocessed notes awaiting review
- `_contested/` — notes with conflicting or uncertain information
- `notes/` — processed, reliable knowledge

## Note Format

Every note file must begin with a YAML frontmatter block delimited by `---` lines,
followed by a plain-text Markdown body.

### Required Frontmatter Fields

| Field | Description | Allowed Values |
|---|---|---|
| `title` | Human-readable title of the note | Any non-empty string |
| `created` | Date the note was first created | YYYY-MM-DD |
| `updated` | Date the note was last updated | YYYY-MM-DD |
| `status` | Lifecycle status of the note | `inbox`, `verified`, `deprecated`, `contested`, `superseded` |
| `confidence` | Confidence level in the note's content | `low`, `medium`, `high` |
| `epistemic-type` | How the knowledge was derived | `observation`, `pattern`, `constraint`, `decision`, `assumption`, `synthesis` |
| `scope` | Whether the note applies to this project or globally | `project`, `cross-project` |
| `source-agent` | Identifier of the agent that created the note | Any non-empty string |
| `source-artifact` | The artifact or context that produced this note | Any non-empty string |
| `domain` | List of knowledge domains this note belongs to | Non-empty list of strings |

### Optional Frontmatter Fields

| Field | Description |
|---|---|
| `review-by` | Date by which the note should be reviewed (YYYY-MM-DD) |
| `project` | Project name — required when `scope` is `project` |
| `verified-by` | Agent that verified the note's content |
| `verified-date` | Date the note was verified (YYYY-MM-DD) |
| `requires-human-review` | Boolean flag for notes needing human review |
| `update-type` | Type of update (`append`, `replace`, `retract`) |
| `targets` | List of note IDs this note targets |
| `tags` | Free-form tags for additional categorisation |

## Body Section Requirements

The note body must contain the following sections as Markdown headings:

- `# <Title>` — an H1 heading matching the note title
- `## Related` — links to related notes

For all epistemic types **except** `synthesis`:
- `## Evidence` — supporting evidence
- `## Implications` — what this means for the project

For `synthesis` notes:
- `## Synthesis` — the synthesised conclusion
- `## Contributing notes` — notes that contributed to this synthesis

## Validation

Use the `agent-memory lint-note` command to validate a note before writing it to the vault:

```
agent-memory lint-note path/to/note.md
agent-memory lint-note --json path/to/note.md
```

### Lint Rule Reference

| Rule | Description |
|---|---|
| NF001 | Required frontmatter fields must be present and non-empty |
| NF002 | Date fields must use YYYY-MM-DD format |
| NF003 | Enum fields must contain one of their allowed values |
| NF004 | Field `project` is required when `scope` is `project` |
| NF005 | Note body must contain required section headings |
| NF006 | Required fields must not contain placeholder values |
| NF007 | Field `domain` must be a non-empty list |

## Writing Notes

Use the `agent-memory write-note` command to write a new note to the vault.

### CLI Syntax

```
agent-memory write-note [--vault <path>] --type <epistemic-type> --title <title> [--project <name>] [--domain <domain>...] [--scope <scope>] [--source-artifact <artifact>] [--source-agent <agent>] [--confidence <level>] [--tags <tag>...] [--force] <body-file>
```

`<body-file>` is a path to a Markdown file containing the note body, or `-` to read from stdin.

### Auto-populated Fields

The tool sets the following fields automatically — the agent does not need to supply them:

| Field | Value |
|---|---|
| `created` | Today's date (YYYY-MM-DD) |
| `updated` | Today's date (YYYY-MM-DD) |
| `status` | `inbox` |
| `review-by` | TTL from today: assumption=30d, observation=90d, pattern=180d, constraint/decision=365d |
| `requires-human-review` | `true` (only set for `constraint` and `decision` types) |
| `source-agent` | `agent-memory-cli` (if `--source-agent` is not provided) |

### Similarity Check

Before writing, the tool scans `_inbox/` and `notes/` for existing notes. If any existing note title has a Jaccard similarity ≥ 0.7 with the incoming title, the write is refused and the candidate notes are listed in the response.

Use `--force` to bypass the similarity check and write the note regardless.

### JSON Output

When the root command is invoked with `--json`, `write-note` returns structured output:

```json
// On success:
{"status": "written", "path": "_inbox/2026-05-21-my-note-title.md"}

// On refusal (similarity check):
{"status": "refused", "reason": "similar note exists", "candidates": ["_inbox/2026-05-01-existing-note.md"]}
```

## Rules

1. Read `_meta/` files before starting work to understand vault conventions.
2. Check `_inbox/` for unprocessed context that may be relevant.
3. Do not modify `_meta/` files unless explicitly instructed.
4. Notes in `notes/` are considered reliable unless marked otherwise.
5. Always validate notes with `agent-memory lint-note` before writing to the vault.
