package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	gh "github.com/redbadger/deploy/github"
	"gopkg.in/go-playground/webhooks.v3"
	"gopkg.in/go-playground/webhooks.v3/github"
)

const (
	secretEnvVar = "DEPLOY_SECRET"
	path         = "/webhooks"
	port         = 3016
)

func main() {
	secret, ok := os.LookupEnv(secretEnvVar)
	if ok == false {
		log.Fatalf("Environment variable %s is not exported.", secretEnvVar)
	}

	hook := github.New(&github.Config{Secret: secret})
	hook.RegisterEvents(handlePullRequest, github.PullRequestEvent)

	err := webhooks.Run(hook, ":"+strconv.Itoa(port), path)
	if err != nil {
		fmt.Println(err)
	}
}

func handlePullRequest(payload interface{}, header webhooks.Header) {
	fmt.Println("Handling PR")
	pl := payload.(github.PullRequestPayload)

	fmt.Printf("PR #%d, SHA %s\n", pl.PullRequest.Number, pl.PullRequest.Head.Sha)
	repo := gh.Repo{pl.Repository.Owner.Login, pl.Repository.Name, pl.PullRequest.Head.Sha}
	fmt.Printf("Clone URL %s\n", gh.GetRepo(repo))
}
