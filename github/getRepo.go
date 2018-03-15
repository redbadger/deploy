package github

import (
	"context"
	"fmt"

	gHttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"gopkg.in/src-d/go-billy.v4/memfs"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

// GetRepo returns a git Repository cloned into a new in-memory filesystem
func GetRepo(apiURL, org, name, token, headRef, baseRef string) (r *git.Repository, err error) {
	context := context.Background()
	tokenService := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tokenClient := oauth2.NewClient(context, tokenService)
	client, err := github.NewEnterpriseClient(apiURL, apiURL, tokenClient)
	if err != nil {
		err = fmt.Errorf("Cannot create github client: %v", err)
		return
	}

	repo, _, err := client.Repositories.Get(context, org, name)
	if err != nil {
		err = fmt.Errorf("Cannot get github repo: %v", err)
		return
	}

	fs := memfs.New()
	url := repo.GetCloneURL()
	r, err = git.CloneContext(context, memory.NewStorage(), fs, &git.CloneOptions{
		URL:  url,
		Auth: &gHttp.BasicAuth{Username: "none", Password: token},
	})
	if err != nil {
		err = fmt.Errorf("Cannot clone github repo (%s): %v", url, err)
		return
	}

	return
}
