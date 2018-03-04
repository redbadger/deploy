package github

import (
	"context"
	"log"
	"os"

	"gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"gopkg.in/src-d/go-billy.v4/memfs"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

const (
	tokenEnvVar = "PERSONAL_ACCESS_TOKEN"
)

// GetRepo gets repo
func GetRepo(org, name, ref string) string {
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

	repo, _, err := client.Repositories.Get(context, org, name)
	if err != nil {
		log.Fatalf("Error getting repository information %v\n", err)
	}

	fs := memfs.New()
	r, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL: repo.GetCloneURL(),
	})
	if err != nil {
		log.Fatalf("Error cloning repository %v\n", err)
	}

	w, err := r.Worktree()
	if err != nil {
		log.Fatalf("Cannot get worktree %v\n", err)
	}
	err = w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(ref),
	})
	if err != nil {
		log.Fatalf("Cannot checkout hash %s: %v\n", ref, err)
	}
	realRef, err := r.Head()
	if err != nil {
		log.Fatalf("Cannot get head: %v\n", err)
	}
	return realRef.Hash().String()

}
