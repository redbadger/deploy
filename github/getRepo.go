package github

import (
	"context"
	"log"
	"os"

	"gopkg.in/src-d/go-billy.v4"

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

// GetRepo returns an in memory filesystem with commit checked out
func GetRepo(org, name, ref string) (fs billy.Filesystem, err error) {
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
		return
	}

	fs = memfs.New()
	r, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL: repo.GetCloneURL(),
	})
	if err != nil {
		return
	}

	w, err := r.Worktree()
	if err != nil {
		return
	}
	err = w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(ref),
	})
	if err != nil {
		return
	}
	return
}
