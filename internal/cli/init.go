package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/michaelin/agent-memory/internal/vault"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var force, clean bool

	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Initialize a new agent memory vault",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var vaultPath string
			if len(args) > 0 {
				vaultPath = args[0]
				if strings.TrimSpace(vaultPath) == "" {
					return fmt.Errorf("vault path must not be empty")
				}
			} else {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				vaultPath = filepath.Join(cwd, ".agent-memory")
			}

			result, err := vault.Init(vaultPath, vault.InitOptions{
				Force: force,
				Clean: clean,
			})
			if err != nil {
				if jsonOutput {
					out, _ := json.Marshal(vault.InitError{Error: err.Error()})
					fmt.Println(string(out))
					cmd.SilenceUsage = true
					cmd.SilenceErrors = true
					return fmt.Errorf("%w", err)
				}
				return err
			}

			if jsonOutput {
				out, err := json.Marshal(result)
				if err != nil {
					return err
				}
				fmt.Println(string(out))
				return nil
			}

			// Human-readable output
			switch result.Status {
			case "created":
				fmt.Printf("Vault created at %s\n", result.Vault)
			case "ok":
				fmt.Println("Vault already up to date")
			case "repaired":
				fmt.Printf("Repaired: %s\n", strings.Join(result.Repaired, ", "))
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite conflicting files or folders")
	cmd.Flags().BoolVar(&clean, "clean", false, "Delete the vault before initializing (requires --force)")
	return cmd
}
