package cmd

import (
	"log"

	"github.com/redbadger/deploy/agent"
	"github.com/redbadger/deploy/constants"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	path string
	port uint16
)

var agentCmd = &cobra.Command{
	Use:     "agent",
	Aliases: []string{"daemon", "bot"},
	Short:   "Run deploy as an agent",
	Long: `
Run deploy as an agent:

1.  watches for PR updates on a webhook
2.  clones the repo to an in-memory filesystem
3.  checks out the commit SHA
4.  walks down any top-level directories that contain changes
5.  gathers yaml files (however they are nested)
6.  applies the manifests to a Kubernetes cluster using kubctl.
`,
	Example: `deploy agent &`,
	Run: func(cmd *cobra.Command, args []string) {
		if !viper.IsSet(constants.SecretEnvVar) {
			log.Fatalf("environment variable %s is not exported.\n", constants.SecretEnvVar)
		}
		if !viper.IsSet(constants.TokenEnvVar) {
			log.Fatalf("environment variable %s is not exported.\n", constants.TokenEnvVar)
		}

		secret := viper.GetString(constants.SecretEnvVar)
		token := viper.GetString(constants.TokenEnvVar)

		agent.Agent(port, path, token, secret)
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.Flags().Uint16VarP(&port, "port", "p", 3016, "Port for webhook listener")
	agentCmd.Flags().StringVar(&path, "path", "/webhooks", "Path for webhook url")
}
