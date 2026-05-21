package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/michaelin/agent-memory/internal/note"
	"github.com/michaelin/agent-memory/internal/vault"
)

func newDeprecateCmd() *cobra.Command {
	var (
		vaultFlag    string
		slugFlag     string
		supersededBy string
	)

	cmd := &cobra.Command{
		Use:   "deprecate",
		Short: "Deprecate a note from notes/ to _deprecated/",
		Long: `Deprecate a note by moving it from notes/ to _deprecated/ and
recording the superseding note slug in its frontmatter.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := note.ValidateSlug(slugFlag); err != nil {
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

			result, err := note.Deprecate(vaultPath, slugFlag, supersededBy)
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

			fmt.Printf("✓ Deprecated: %s → %s\n", result.Slug, result.MovedTo)
			return nil
		},
	}

	cmd.Flags().StringVar(&vaultFlag, "vault", "", "Path to the vault (overrides AGENT_MEMORY_VAULT and auto-discovery)")
	cmd.Flags().StringVar(&slugFlag, "slug", "", "Slug of the note to deprecate")
	cmd.Flags().StringVar(&supersededBy, "superseded-by", "", "Slug of the note that supersedes this one")

	_ = cmd.MarkFlagRequired("slug")

	return cmd
}
