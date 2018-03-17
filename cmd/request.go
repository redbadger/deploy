package cmd

import (
	"log"

	"github.com/redbadger/deploy/constants"
	"github.com/redbadger/deploy/request"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var requestCmd = &cobra.Command{
	Use:   "request",
	Short: "Raise a PR against the cluster repo with the configuration to be deployed",
	Long: `
Raise a PR against the cluster repo with the configuration to be deployed:

1. checks out the cluster repo specified
2. copies the specified manifests into a new branch
3. commits, pushes and raises a PR requesting deployment
	`,
	Run: func(cmd *cobra.Command, args []string) {
		if !viper.IsSet(constants.TokenEnvVar) {
			log.Fatalf("environment variable %s is not exported.\n", constants.TokenEnvVar)
		}
		token := viper.GetString(constants.TokenEnvVar)

		project, err := cmd.Flags().GetString("project")
		if err != nil {
			log.Fatalf("Must specifiy project: %v", err)
		}
		githubURL, err := cmd.Flags().GetString("githubURL")
		if err != nil {
			log.Fatalf("Must specifiy githubURL: %v", err)
		}
		apiURL, err := cmd.Flags().GetString("apiURL")
		if err != nil {
			log.Fatalf("Must specifiy apiURL: %v", err)
		}
		org, err := cmd.Flags().GetString("org")
		if err != nil {
			log.Fatalf("Must specifiy org: %v", err)
		}
		repo, err := cmd.Flags().GetString("repo")
		if err != nil {
			log.Fatalf("Must specifiy repo: %v", err)
		}
		stacksDir, err := cmd.Flags().GetString("stacksDir")
		if err != nil {
			log.Fatalf("Must specifiy stacksDir: %v", err)
		}
		request.Request(token, project, githubURL, apiURL, org, repo, stacksDir)
	},
}

func init() {
	rootCmd.AddCommand(requestCmd)
	requestCmd.Flags().String("project", "", "Project name")
	requestCmd.MarkFlagRequired("project")
	requestCmd.Flags().String("githubURL", "https://github.com", "Github URL")
	requestCmd.Flags().String("apiURL", "https://api.github.com/", "Github API URL")
	requestCmd.Flags().String("org", "", "Organisation name")
	requestCmd.MarkFlagRequired("org")
	requestCmd.Flags().String("repo", "", "Repository name")
	requestCmd.MarkFlagRequired("repo")
	requestCmd.Flags().String("stacksDir", "stacks", "Name of stacks directory")
}
