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
func GetRepo(cloneURL, token string) (r *git.Repository, err error) {
	context := context.Background()
	r, err = git.CloneContext(context, memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:  cloneURL,
		Auth: &gHttp.BasicAuth{Username: "none", Password: token},
	})
	if err != nil {
		err = fmt.Errorf("cannot clone github repo (%s): %v", cloneURL, err)
		return
	}

	return
}
