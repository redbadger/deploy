package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// NewClient creates a new github client
func NewClient(apiURL, token string) (client *github.Client, err error) {
	tokenService := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tokenClient := oauth2.NewClient(context.Background(), tokenService)

	client, err = github.NewEnterpriseClient(apiURL, apiURL, tokenClient)
	if err != nil {
		err = fmt.Errorf("Cannot create github client: %v", err)
		return
	}
	return
}