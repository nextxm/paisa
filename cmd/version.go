package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var Version = "0.7.4"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Version:", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
