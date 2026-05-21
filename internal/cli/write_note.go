package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/michaelin/agent-memory/internal/note"
	"github.com/michaelin/agent-memory/internal/vault"
)

func newWriteNoteCmd() *cobra.Command {
	var (
		vaultFlag      string
		typeFlag       string
		titleFlag      string
		projectFlag    string
		domainFlag     []string
		scopeFlag      string
		sourceArtifact string
		sourceAgent    string
		confidenceFlag string
		tagsFlag       []string
		forceFlag      bool
	)

	cmd := &cobra.Command{
		Use:   "write-note <body-file>",
		Short: "Write a note to the agent-memory vault",
		Long: `Write a note to the agent-memory vault.

The body is read from <body-file>. Pass - to read from stdin.

Required flags: --type, --title, --domain, --scope, --source-artifact, --confidence.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bodyFile := args[0]

			// Read body content.
			var bodyBytes []byte
			var err error
			if bodyFile == "-" {
				bodyBytes, err = io.ReadAll(os.Stdin)
			} else {
				bodyBytes, err = os.ReadFile(bodyFile)
			}
			if err != nil {
				cmd.SilenceUsage = true
				if jsonOutput {
					out, _ := json.Marshal(map[string]string{"status": "error", "error": err.Error()})
					fmt.Println(string(out))
					cmd.SilenceErrors = true
					return fmt.Errorf("%w", err)
				}
				return err
			}

			// Resolve vault.
			vaultPath, err := vault.Discover(vaultFlag)
			if err != nil {
				cmd.SilenceUsage = true
				if jsonOutput {
					out, _ := json.Marshal(map[string]string{"status": "error", "error": err.Error()})
					fmt.Println(string(out))
					cmd.SilenceErrors = true
					return fmt.Errorf("%w", err)
				}
				return err
			}

			opts := note.WriteOptions{
				VaultPath:      vaultPath,
				Body:           string(bodyBytes),
				EpistemicType:  typeFlag,
				Title:          titleFlag,
				Project:        projectFlag,
				Domain:         domainFlag,
				Scope:          scopeFlag,
				SourceArtifact: sourceArtifact,
				SourceAgent:    sourceAgent,
				Confidence:     confidenceFlag,
				Tags:           tagsFlag,
				Force:          forceFlag,
			}

			result, err := note.Write(opts)
			if err != nil {
				cmd.SilenceUsage = true
				if jsonOutput {
					out, _ := json.Marshal(map[string]string{"status": "error", "error": err.Error()})
					fmt.Println(string(out))
					cmd.SilenceErrors = true
					return fmt.Errorf("%w", err)
				}
				return err
			}

			if jsonOutput {
				out, merr := json.Marshal(result)
				if merr != nil {
					cmd.SilenceUsage = true
					return fmt.Errorf("marshal JSON: %w", merr)
				}
				fmt.Println(string(out))
			if result.Status == "refused" {
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
				return fmt.Errorf("note refused: %s", result.Reason)
			}
			return nil
		}

		// Human-readable output.
		if result.Status == "written" {
				fmt.Printf("✓ Written: %s\n", result.Path)
				return nil
			}

			// Refused.
			fmt.Printf("✗ Refused: %s\n", result.Reason)
			for _, c := range result.Candidates {
				fmt.Printf("  similar: %s (score %.2f)\n", c.Path, c.Similarity)
			}
			cmd.SilenceUsage = true
			return fmt.Errorf("note refused: %s", result.Reason)
		},
	}

	cmd.Flags().StringVar(&vaultFlag, "vault", "", "Path to the vault (overrides AGENT_MEMORY_VAULT and auto-discovery)")
	cmd.Flags().StringVar(&typeFlag, "type", "", "Epistemic type (observation, pattern, constraint, decision, assumption, synthesis)")
	cmd.Flags().StringVar(&titleFlag, "title", "", "Note title")
	cmd.Flags().StringVar(&projectFlag, "project", "", "Project name (required when scope is 'project')")
	cmd.Flags().StringSliceVar(&domainFlag, "domain", nil, "Domain tags (comma-separated or repeated flag)")
	cmd.Flags().StringVar(&scopeFlag, "scope", "", "Scope (project or cross-project)")
	cmd.Flags().StringVar(&sourceArtifact, "source-artifact", "", "Source artifact identifier")
	cmd.Flags().StringVar(&sourceAgent, "source-agent", "", "Source agent identifier")
	cmd.Flags().StringVar(&confidenceFlag, "confidence", "", "Confidence level (low, medium, high)")
	cmd.Flags().StringSliceVar(&tagsFlag, "tags", nil, "Additional tags (comma-separated or repeated flag)")
	cmd.Flags().BoolVar(&forceFlag, "force", false, "Bypass similarity refusal and write unconditionally")

	return cmd
}
