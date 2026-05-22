package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const instructionsBlurb = `<!-- BEGIN agent-memory -->
# Agent Memory

This project uses agent-memory for persistent knowledge storage.

## Vault Location

The vault is at ` + "`.agent-memory/`" + ` in this repository.

## Reading from Memory

Before starting work, check the vault for relevant context:
1. Read ` + "`_meta/writing-protocol.md`" + ` in the vault for the note format spec and rules
2. Check ` + "`notes/`" + ` for existing verified knowledge
3. Check ` + "`_inbox/`" + ` for unprocessed notes

When using agent-memory commands programmatically, pass ` + "`--json`" + ` for
machine-readable output.

## Writing to Memory

**At the end of every task, write your durable findings to memory.** Use the
` + "`memory-write`" + ` tool (if available via plugin) or ` + "`agent-memory write-note`" + ` to
record knowledge that would be valuable to a different agent working on a
different task in this codebase. All writes go through the Librarian for
validation and promotion.

Do not write routine observations that are obvious from reading the code.
Write things that required investigation, that surprised you, or that
future agents would otherwise have to re-derive.

### What to write — epistemic types

| Type | When to use | Example |
|---|---|---|
| ` + "`observation`" + ` | You directly read or verified something in the codebase | "The auth middleware runs before cors in server.go" |
| ` + "`pattern`" + ` | You noticed a recurring theme across 2+ observations | "All services use the repository pattern" |
| ` + "`constraint`" + ` | An external rule the system must respect (requires human confirmation) | "All API responses must include X-Request-ID headers" |
| ` + "`decision`" + ` | A deliberate choice made by a human (requires human confirmation) | "We chose PostgreSQL over MongoDB for the user store" |
| ` + "`assumption`" + ` | A belief you hold without verification (30-day TTL) | "The CI pipeline probably runs on Ubuntu 22.04" |

If unsure, use ` + "`observation`" + `. The Librarian will reclassify if needed.

### How to write

    agent-memory write-note --type observation --title "My Note" --scope cross-project \
      --domain testing --source-artifact ci-run --confidence high body.md
    agent-memory write-note --json --type observation --title "My Note" ... body.md

Pass ` + "`-`" + ` as the body file to read from stdin. Use ` + "`--force`" + ` to bypass similarity
refusal. Refer to ` + "`_meta/writing-protocol.md`" + ` for the full format specification.

## Note Validation

Use ` + "`agent-memory lint-note`" + ` to validate a note file:

    agent-memory lint-note path/to/note.md
    agent-memory lint-note --json path/to/note.md

## Librarian Workflow

The Librarian subagent manages note lifecycle: scanning inbox, promoting
verified notes, and deprecating superseded ones.

### Promote a note

Move a note from ` + "`_inbox/`" + ` to ` + "`notes/`" + ` after validation:

    agent-memory promote --slug=<slug> --json
    agent-memory promote --slug=<slug> --confirmed --json   # for constraint/decision notes

The ` + "`--confirmed`" + ` flag is required for notes with ` + "`requires-human-review: true`" + `
(constraint and decision types). It is silently ignored for other types.

### Deprecate a note

Move a note from ` + "`notes/`" + ` to ` + "`_deprecated/`" + ` with a forward link:

    agent-memory deprecate --slug=<slug> --superseded-by=<new-slug> --json

### Deployment templates

Librarian agent and skill templates are installed at:
- ` + "`_meta/templates/librarian-agent.md`" + `
- ` + "`_meta/templates/librarian-skill.md`" + `

Copy these to your OpenCode harness configuration to deploy the Librarian.
<!-- END agent-memory -->
`

func newInstructionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "instructions",
		Short: "Output agent configuration blurb for piping into agent instruction files",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(instructionsBlurb)
		},
	}
}
