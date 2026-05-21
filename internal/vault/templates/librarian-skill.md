---
name: librarian-workflow
description: >
  Step-by-step workflow for the Librarian subagent. Guides inbox scanning,
  linting, type-based routing, promotion, deprecation, and summary reporting.
---

# Librarian Workflow

Follow these steps in order each time the Librarian is invoked.

## Step 1 — Scan inbox

List all `.md` files under `_inbox/`. If the inbox is empty, return a summary
stating no notes were pending and exit.

## Step 2 — Lint each note

For each file, run:

```
agent-memory lint-note --json <path>
```

If lint fails, add the file to the **skipped** list with the lint errors and
continue to the next file.

## Step 3 — Check note type

Parse the `type` field from the note's frontmatter:

- `observation`, `pattern`, `assumption`, `synthesis` → proceed to Step 4
  (automatic promotion).
- `constraint`, `decision` → proceed to Step 5 (human confirmation required).

## Step 4 — Promote automatically

Extract the slug from the filename. Inbox filenames follow the pattern
`{date}-{slug}.md` — the slug is everything after the first hyphen-separated
date prefix.

Run:

```
agent-memory promote --slug=<slug> --json
```

Add the note to the **promoted** list. Then proceed to Step 6.

## Step 5 — Request human confirmation

Present the note title, type, and body summary to the human. Ask:

> "This note is a `<type>`. Do you approve promoting it to `notes/`?"

- If approved → run:
  ```
  agent-memory promote --slug=<slug> --confirmed --json
  ```
  Add to **promoted** list, then proceed to Step 6.
- If rejected → add to **skipped** list with reason "human declined".

## Step 6 — Check for superseded notes

After promoting a note, check whether it replaces an existing note in `notes/`.
Look for wikilinks in the note body that reference notes in `notes/` — if the
note body explicitly states it replaces or supersedes another note, deprecate
the old one:

```
agent-memory deprecate --slug=<old-slug> --superseded-by=<new-slug> --json
```

If no supersession is indicated, skip this step.

## Step 7 — Return summary

Report:

- **Promoted**: count and list of promoted note titles.
- **Deprecated**: count and list of deprecated note slugs with their replacements.
- **Skipped**: count and list of skipped files with reasons (lint errors, human declined).
- **Escalated**: count and list of notes awaiting human confirmation (if any
  were deferred rather than answered inline).
