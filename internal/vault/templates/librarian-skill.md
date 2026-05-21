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
agent-memory lint-note <path>
```

If lint fails, add the file to the **skipped** list with the lint errors and
continue to the next file.

## Step 3 — Check epistemic type

Parse the `type` field from the note's frontmatter:

- `observation`, `pattern`, `assumption`, `synthesis` → proceed to Step 4
  (automatic promotion).
- `constraint`, `decision` → proceed to Step 5 (human confirmation required).

## Step 4 — Promote automatically

Run:

```
agent-memory promote <path>
```

Add the note to the **promoted** list. Then check whether the note's
`superseded-by` field is set; if so, proceed to Step 6.

## Step 5 — Request human confirmation

Present the note title, type, and body summary to the human. Ask:

> "This note is a `<type>`. Do you approve promoting it to `notes/`?"

- If approved → run `agent-memory promote <path>`, add to **promoted** list,
  then check `superseded-by` (Step 6).
- If rejected → add to **skipped** list with reason "human declined".

## Step 6 — Deprecate superseded notes

If the promoted note's `superseded-by` field names a slug, run:

```
agent-memory deprecate <slug>
```

Record the deprecation in the summary.

## Step 7 — Return summary

Report:

- **Promoted**: count and list of promoted note titles.
- **Deprecated**: count and list of deprecated note slugs.
- **Skipped**: count and list of skipped files with reasons.
- **Escalated**: count and list of notes awaiting human confirmation (if any
  were deferred rather than answered inline).
