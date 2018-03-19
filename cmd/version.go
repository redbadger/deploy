package cmd

import (
	"fmt"

	"github.com/redbadger/deploy/constants"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version of the deploy command",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("deploy version %s\n", constants.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
