package cmd

import (
	"github.com/spf13/cobra"
)

// serviceaccountCmd represents the serviceaccount command
var serviceaccountCmd = &cobra.Command{
	Use:     "serviceaccount",
	Short:   "Manage ServiceAccounts",
	Aliases: []string{"sa"},
}

func init() {
	rootCmd.AddCommand(serviceaccountCmd)
}
