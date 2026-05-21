package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/michaelin/agent-memory/internal/note"
	"github.com/spf13/cobra"
)

func newLintNoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint-note <file>",
		Short: "Lint a note file against the agent-memory note format rules",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			data, err := os.ReadFile(path)
			if err != nil {
				if jsonOutput {
					result := &note.LintResult{
						Valid: false,
						Errors: []note.LintError{
							{Rule: "read", Message: err.Error()},
						},
					}
					out, merr := json.Marshal(result)
					if merr != nil {
						cmd.SilenceUsage = true
						return fmt.Errorf("marshal JSON: %w", merr)
					}
					fmt.Println(string(out))
					cmd.SilenceUsage = true
					cmd.SilenceErrors = true
					return fmt.Errorf("%w", err)
				}
				cmd.SilenceUsage = true
				return err
			}

			n, err := note.Parse(data)
			if err != nil {
				if jsonOutput {
					result := &note.LintResult{
						Valid: false,
						Errors: []note.LintError{
							{Rule: "parse", Message: err.Error()},
						},
					}
					out, merr := json.Marshal(result)
					if merr != nil {
						cmd.SilenceUsage = true
						return fmt.Errorf("marshal JSON: %w", merr)
					}
					fmt.Println(string(out))
					cmd.SilenceUsage = true
					cmd.SilenceErrors = true
					return fmt.Errorf("%w", err)
				}
				cmd.SilenceUsage = true
				return err
			}

			result := note.Lint(n)

			if jsonOutput {
				out, err := json.Marshal(result)
				if err != nil {
					cmd.SilenceUsage = true
					return fmt.Errorf("marshal JSON: %w", err)
				}
				fmt.Println(string(out))
				if !result.Valid {
					cmd.SilenceUsage = true
					cmd.SilenceErrors = true
					return fmt.Errorf("note has %d error(s)", len(result.Errors))
				}
				return nil
			}

			// Human-readable output.
			if result.Valid {
				fmt.Println("✓ note is valid")
				return nil
			}

			fmt.Fprintf(os.Stdout, "✗ note has %d error(s):\n", len(result.Errors))
			for _, e := range result.Errors {
				fmt.Fprintf(os.Stdout, "  %s: %s\n", e.Rule, e.Message)
			}
			cmd.SilenceUsage = true
			return fmt.Errorf("note has %d error(s)", len(result.Errors))
		},
	}
	return cmd
}
