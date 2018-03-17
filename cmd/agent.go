package cmd

import (
	"log"

	"github.com/redbadger/deploy/agent"
	"github.com/redbadger/deploy/constants"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run deploy in agent mode",
	Long: `
Run deploy in agent mode:

1.  watches for PR updates on a webhook
2.  clones the repo to an in-memory filesystem
3.  checks out the commit SHA
4.  walks down any top-level directories that contain changes
5.  gathers yaml files (however they are nested)
6.  applies the manifests to a Kubernetes cluster using kubctl.
`,
	Run: func(cmd *cobra.Command, args []string) {
		port, err := cmd.Flags().GetUint16("port")
		if err != nil {
			log.Fatalf("Must specifiy port: %v", err)
		}
		path, err := cmd.Flags().GetString("path")
		if err != nil {
			log.Fatalf("Must specifiy path: %v", err)
		}

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
	agentCmd.Flags().Uint16P("port", "p", 3016, "Port for webhook listener")
	agentCmd.Flags().String("path", "/webhooks", "Path for webhook url")
}
