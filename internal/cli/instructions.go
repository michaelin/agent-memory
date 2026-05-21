package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newInstructionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "instructions",
		Short: "Output agent configuration blurb for piping into agent instruction files",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print("# Agent Memory\n\nThis project uses agent-memory for persistent knowledge storage.\n\n## Vault Location\n\nThe vault is at `.agent-memory/` in this repository.\n\n## Usage\n\nBefore starting work, check the vault for relevant context:\n1. Read `_meta/writing-protocol.md` in the vault for the note format spec and rules\n2. Check `_inbox/` for unprocessed notes\n3. Check `notes/` for existing knowledge\n\nWhen using agent-memory commands programmatically, pass `--json` for\nmachine-readable output.\n\n## Note Validation\n\nThe note format is defined in `_meta/writing-protocol.md`. Use `agent-memory lint-note`\nto validate a note file before writing it to the vault:\n\n    agent-memory lint-note path/to/note.md\n    agent-memory lint-note --json path/to/note.md\n\nWriting to the vault is not yet enabled. Validate notes with lint-note and refer to\n`_meta/writing-protocol.md` for the full format specification.\n")
		},
	}
}
