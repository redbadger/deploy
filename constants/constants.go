package constants

const (
	// Version is the application version reported by `deploy version` and `deploy --version`
	Version = "0.1"
	// SecretEnvVar is the name of the environment variable that holds the webhook secret
	SecretEnvVar = "DEPLOY_SECRET"
	// TokenEnvVar is the name of the environment variable that holds the github personal access token
	TokenEnvVar = "PERSONAL_ACCESS_TOKEN"
)
