package cli

import (
	"github.com/spf13/cobra"
)

// jsonOutput controls whether commands emit JSON instead of human-readable text.
var jsonOutput bool

// NewRootCmd creates the root cobra command with all subcommands registered.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agent-memory",
		Short:   "Agent memory vault management",
		Version: "0.1.0",
	}
	cmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output machine-readable JSON")
	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newInstructionsCmd())
	return cmd
}
