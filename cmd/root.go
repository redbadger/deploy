package cmd

import (
	homedir "github.com/mitchellh/go-homedir"
	"github.com/redbadger/deploy/constants"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	secret  string
	token   string
)

var rootCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy to Kubernetes through a cluster repository",
	Long: `
Deploy runs in two modes:

1. as an agent: deploy agent
   runs in Kubernetes and deploys application configuration contained in a PR

2. as a cli command: deploy request
   usually run by CI/CD pipeline
	`,
	Version: constants.Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.deploy.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			log.Fatal(err)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".deploy")
	}

	viper.AutomaticEnv()
	viper.BindEnv(constants.SecretEnvVar)
	viper.BindEnv(constants.TokenEnvVar)

	if err := viper.ReadInConfig(); err == nil {
		log.WithField("file", viper.ConfigFileUsed()).Info("Using config")
	}
}
