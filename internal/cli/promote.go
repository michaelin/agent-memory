package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/michaelin/agent-memory/internal/note"
	"github.com/michaelin/agent-memory/internal/vault"
)

func newPromoteCmd() *cobra.Command {
	var (
		vaultFlag string
		slugFlag  string
		confirmed bool
	)

	cmd := &cobra.Command{
		Use:   "promote",
		Short: "Promote a note from _inbox/ to notes/",
		Long: `Promote a note from _inbox/ to notes/ after validation.

The note must pass lint and wikilink resolution checks. Notes with
requires-human-review: true (constraint and decision types) require
the --confirmed flag.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateSlug(slugFlag); err != nil {
				cmd.SilenceUsage = true
				return err
			}

			vaultPath, err := vault.Discover(vaultFlag)
			if err != nil {
				cmd.SilenceUsage = true
				if jsonOutput {
					out, _ := json.Marshal(map[string]string{"status": "error", "error": "vault not found"})
					fmt.Println(string(out))
					cmd.SilenceErrors = true
					return fmt.Errorf("%w", err)
				}
				return err
			}

			result, err := note.Promote(vaultPath, slugFlag, confirmed)
			if err != nil {
				cmd.SilenceUsage = true
				if jsonOutput {
					out, _ := json.Marshal(result)
					fmt.Println(string(out))
					cmd.SilenceErrors = true
					return fmt.Errorf("%w", err)
				}
				fmt.Printf("✗ Error: %s\n", result.Error)
				return err
			}

			if jsonOutput {
				out, merr := json.Marshal(result)
				if merr != nil {
					cmd.SilenceUsage = true
					return fmt.Errorf("marshal JSON: %w", merr)
				}
				fmt.Println(string(out))
				return nil
			}

			fmt.Printf("✓ Promoted: %s → %s\n", result.Slug, result.To)
			return nil
		},
	}

	cmd.Flags().StringVar(&vaultFlag, "vault", "", "Path to the vault (overrides AGENT_MEMORY_VAULT and auto-discovery)")
	cmd.Flags().StringVar(&slugFlag, "slug", "", "Slug of the note to promote")
	cmd.Flags().BoolVar(&confirmed, "confirmed", false, "Confirm promotion of constraint/decision notes requiring human review")

	_ = cmd.MarkFlagRequired("slug")

	return cmd
}
