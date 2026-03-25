package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion <bash|zsh|fish>",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for bash, zsh, or fish.

Install examples:
  poeusage completion bash > /etc/bash_completion.d/poeusage
  poeusage completion zsh > ~/.zsh/completions/_poeusage
  poeusage completion fish > ~/.config/fish/completions/poeusage.fish`,
	Args: cobra.ExactArgs(1),
	// Override PersistentPreRunE to skip API key requirement
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		default:
			return &UsageError{Msg: fmt.Sprintf("unknown shell %q: must be bash, zsh, or fish", args[0])}
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
