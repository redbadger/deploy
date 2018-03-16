package github

import (
	"context"
	"fmt"

	gHttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"

	"gopkg.in/src-d/go-billy.v4/memfs"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

// GetRepo returns a git Repository cloned into a new in-memory filesystem
func GetRepo(apiURL, org, name, token, headRef, baseRef string) (r *git.Repository, err error) {
	context := context.Background()
	client, err := NewClient(apiURL, token)
	if err != nil {
		err = fmt.Errorf("cannot create github client: %v", err)
		return
	}

	repo, _, err := client.Repositories.Get(context, org, name)
	if err != nil {
		err = fmt.Errorf("cannot get github repo: %v", err)
		return
	}

	fs := memfs.New()
	url := repo.GetCloneURL()
	r, err = git.CloneContext(context, memory.NewStorage(), fs, &git.CloneOptions{
		URL:  url,
		Auth: &gHttp.BasicAuth{Username: "none", Password: token},
	})
	if err != nil {
		err = fmt.Errorf("cannot clone github repo (%s): %v", url, err)
		return
	}

	return
}
