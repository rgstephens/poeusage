package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/gstephens/poeusage/internal/output"
)

var balanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Fetch and display current point balance",
	RunE:  runBalance,
}

func runBalance(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	bal, err := globalState.client.GetBalance(ctx)
	if err != nil {
		return err
	}

	output.PrintBalance(os.Stdout, bal, globalState.outOpts)
	return nil
}

func init() {
	rootCmd.AddCommand(balanceCmd)
}
