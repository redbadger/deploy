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
func GetRepo(ctx context.Context, cloneURL, token string) (r *git.Repository, err error) {
	r, err = git.CloneContext(ctx, memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:  cloneURL,
		Auth: &gHttp.BasicAuth{Username: "none", Password: token},
	})
	if err != nil {
		err = fmt.Errorf("cannot clone github repo (%s): %v", cloneURL, err)
		return
	}

	return
}
