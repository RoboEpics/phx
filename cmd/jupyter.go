package cmd

import (
	"github.com/spf13/cobra"
)

// jupyterCmd represents the jupyter command
var jupyterCmd = &cobra.Command{
	Use:     "jupyter",
	Aliases: []string{"jup"},
	Short:   "Run remote jupyter kernels",
}

func init() {
	rootCmd.AddCommand(jupyterCmd)
}
