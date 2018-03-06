package github

import (
	"context"
	"fmt"
	"log"
	"os"

	"gopkg.in/src-d/go-git.v4/plumbing"

	"gopkg.in/src-d/go-billy.v4"

	gHttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"

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
func GetRepo(apiURL, org, name, headRef, baseRef string) (fs billy.Filesystem, err error) {
	token, ok := os.LookupEnv(tokenEnvVar)
	if ok == false {
		log.Fatalf("Environment variable %s is not exported.", tokenEnvVar)
	}
	context := context.Background()
	tokenService := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tokenClient := oauth2.NewClient(context, tokenService)
	client, err := github.NewEnterpriseClient(apiURL, apiURL, tokenClient)
	if err != nil {
		return
	}

	repo, _, err := client.Repositories.Get(context, org, name)
	if err != nil {
		return
	}

	fs = memfs.New()
	r, err := git.CloneContext(context, memory.NewStorage(), fs, &git.CloneOptions{
		URL:  repo.GetCloneURL(),
		Auth: &gHttp.BasicAuth{Username: "none", Password: token},
	})
	if err != nil {
		return
	}

	files, err := GetChangedProjects(r, headRef, baseRef)
	if err != nil {
		return
	}
	fmt.Println(files)

	w, err := r.Worktree()
	if err != nil {
		return
	}
	err = w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(headRef),
	})
	if err != nil {
		return
	}
	return
}
