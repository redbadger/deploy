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
	Short: "request raises a PR against the deploy repo with the configuration to be deployed",
	Long:  `request raises a PR against the deploy repo with the configuration to be deployed`,
	Run: func(cmd *cobra.Command, args []string) {
		if !viper.IsSet(constants.TokenEnvVar) {
			log.Fatalf("environment variable %s is not exported.\n", constants.TokenEnvVar)
		}
		token := viper.GetString(constants.TokenEnvVar)

		project, err := cmd.Flags().GetString("project")
		if err != nil {
			log.Fatalf("Must specifiy project: %v", err)
		}
		apiURL, err := cmd.Flags().GetString("apiURL")
		if err != nil {
			log.Fatalf("Must specifiy apiURL: %v", err)
		}
		cloneURL, err := cmd.Flags().GetString("cloneURL")
		if err != nil {
			log.Fatalf("Must specifiy cloneURL: %v", err)
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
		request.Request(token, project, apiURL, cloneURL, org, repo, stacksDir)
	},
}

func init() {
	rootCmd.AddCommand(requestCmd)
	requestCmd.Flags().String("project", "", "Project name")
	requestCmd.MarkFlagRequired("project")
	requestCmd.Flags().String("apiURL", "https://api.github.com/", "Github API URL")
	requestCmd.Flags().String("cloneURL", "", "Repository Clone URL")
	requestCmd.MarkFlagRequired("cloneURL")
	requestCmd.Flags().String("org", "", "Organisation name")
	requestCmd.MarkFlagRequired("org")
	requestCmd.Flags().String("repo", "", "Repository name")
	requestCmd.MarkFlagRequired("repo")
	requestCmd.Flags().String("stacksDir", "stacks", "Name of stacks directory")
}
