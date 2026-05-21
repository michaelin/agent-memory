# Writing Protocol v2

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
| `status` | Lifecycle status of the note | `draft`, `active`, `archived`, `deprecated` |
| `confidence` | Confidence level in the note's content | `low`, `medium`, `high` |
| `epistemic-type` | How the knowledge was derived | `observation`, `inference`, `synthesis`, `hypothesis`, `procedure` |
| `scope` | Whether the note applies to this project or globally | `project`, `global` |
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

## Rules

1. Read `_meta/` files before starting work to understand vault conventions.
2. Check `_inbox/` for unprocessed context that may be relevant.
3. Do not modify `_meta/` files unless explicitly instructed.
4. Notes in `notes/` are considered reliable unless marked otherwise.
5. Always validate notes with `agent-memory lint-note` before writing to the vault.
