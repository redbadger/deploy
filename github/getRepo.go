package github

import (
	"context"
	"fmt"

	"gopkg.in/src-d/go-git.v4/plumbing"

	"gopkg.in/src-d/go-billy.v4"

	gHttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"gopkg.in/src-d/go-billy.v4/memfs"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

// GetRepo returns an in memory filesystem with commit checked out
func GetRepo(apiURL, org, name, token, headRef, baseRef string) (fs billy.Filesystem, changedDirs []string, err error) {
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

	fs = memfs.New()
	url := repo.GetCloneURL()
	r, err := git.CloneContext(context, memory.NewStorage(), fs, &git.CloneOptions{
		URL:  url,
		Auth: &gHttp.BasicAuth{Username: "none", Password: token},
	})
	if err != nil {
		err = fmt.Errorf("Cannot clone github repo (%s): %v", url, err)
		return
	}

	changedDirs, err = GetChangedProjects(r, headRef, baseRef)
	if err != nil {
		err = fmt.Errorf("Error identifying changed top level directories: %v", err)
		return
	}

	w, err := r.Worktree()
	if err != nil {
		err = fmt.Errorf("Error getting work tree: %v", err)
		return
	}
	err = w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(headRef),
	})
	if err != nil {
		err = fmt.Errorf("Error checking out %s: %v", headRef, err)
		return
	}
	return
}
