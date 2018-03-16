package cmd

import (
	"github.com/redbadger/deploy/agent"
	"github.com/spf13/cobra"
)

// agentCmd represents the agent command
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run deploy in agent mode",
	Long: `
	1.  watches for PR updates on a webhook
	2.  clones the repo to an in-memory filesystem
	3.  checks out the commit SHA
	4.  walks down any top-level directories that contain changes
	5.  gathers yaml files (however they are nested)
	6.  applies the manifests to a Kubernetes cluster using kubctl.
`,
	Run: func(cmd *cobra.Command, args []string) {
		agent.Agent()
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// agentCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// agentCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
