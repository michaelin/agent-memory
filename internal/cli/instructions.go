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
			fmt.Print("# Agent Memory\n\nThis project uses agent-memory for persistent knowledge storage.\n\n## Vault Location\n\nThe vault is at `.agent-memory/` in this repository (or set `AGENT_MEMORY_VAULT`\nto point elsewhere).\n\n## Usage\n\nBefore starting work, check the vault for relevant context:\n1. Read `_meta/writing-protocol.md` in the vault for rules and conventions\n2. Check `_inbox/` for unprocessed notes\n3. Check `notes/` for existing knowledge\n\nWhen using agent-memory commands programmatically, pass `--json` for\nmachine-readable output.\n\nWriting to the vault is not yet enabled in this version.\n")
		},
	}
}
