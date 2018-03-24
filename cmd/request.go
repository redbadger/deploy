package cmd

import (
	"log"

	"github.com/redbadger/deploy/constants"
	"github.com/redbadger/deploy/request"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	stacksDir string
	project   string
	sha       string
	githubURL string
	apiURL    string
	org       string
	repo      string
)

var requestCmd = &cobra.Command{
	Use:     "request",
	Aliases: []string{"pr"},
	Short:   "Raise a PR against the cluster repo with the configuration to be deployed",
	Long: `
Raise a PR against the cluster repo with the configuration to be deployed:

1. checks out the cluster repo specified
2. copies the specified manifests into a new branch
3. commits, pushes and raises a PR requesting deployment
	`,
	Example: `deploy request --stacksDir=example --project=guestbook --sha=41e8650 --org=redbadger --repo=cluster-local`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if !viper.IsSet(constants.TokenEnvVar) {
			log.Fatalf("environment variable %s is not exported.\n", constants.TokenEnvVar)
		}
		token = viper.GetString(constants.TokenEnvVar)
	},
	Run: func(cmd *cobra.Command, args []string) {
		request.Request(stacksDir, project, sha, githubURL, apiURL, org, repo, token)
	},
}

func init() {
	rootCmd.AddCommand(requestCmd)
	requestCmd.Flags().StringVar(&stacksDir, "stacksDir", "stacks", "Name of stacks directory")

	requestCmd.Flags().StringVar(&project, "project", "", "Project name")
	requestCmd.MarkFlagRequired("project")

	requestCmd.Flags().StringVar(&sha, "sha", "", "Commit SHA")
	requestCmd.MarkFlagRequired("sha")

	requestCmd.Flags().StringVar(&githubURL, "githubURL", "https://github.com", "Github URL")

	requestCmd.Flags().StringVar(&apiURL, "apiURL", "https://api.github.com/", "Github API URL")

	requestCmd.Flags().StringVar(&org, "org", "", "Organisation name")
	requestCmd.MarkFlagRequired("org")

	requestCmd.Flags().StringVar(&repo, "repo", "", "Repository name")
	requestCmd.MarkFlagRequired("repo")
}
