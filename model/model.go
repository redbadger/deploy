package model

// The DeploymentRequest type carries all the information needed to request a deployment
type DeploymentRequest struct {
	// URL is the repository URL
	URL string
	// CloneURL is the URL used to clone the repo
	CloneURL string
	// Token is the user's github Personal Access Token
	Token string
	// The repo owner
	Owner string
	// the repo name
	Repo string
	// the SHA of the HEAD
	HeadRef string
	// the SHA of the BASE
	BaseRef string
}
