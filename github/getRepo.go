package github

import (
	"context"
	"log"
	"os"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	tokenEnvVar = "PERSONAL_ACCESS_TOKEN"
)

type Repo struct {
	Org  string
	Name string
	Ref  string
}

func GetRepo(r Repo) string {
	token, ok := os.LookupEnv(tokenEnvVar)
	if ok == false {
		log.Fatalf("Environment variable %s is not exported.", tokenEnvVar)
	}
	context := context.Background()
	tokenService := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tokenClient := oauth2.NewClient(context, tokenService)
	client := github.NewClient(tokenClient)

	repo, _, err := client.Repositories.Get(context, r.Org, r.Name)
	if err != nil {
		log.Fatalf("Error getting repository information %v\n", err)
	}

	return repo.GetCloneURL()
}
