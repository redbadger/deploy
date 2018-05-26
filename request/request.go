package request

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"time"

	"github.com/google/go-github/github"
	"github.com/redbadger/deploy/filesystem"
	gh "github.com/redbadger/deploy/github"
	log "github.com/sirupsen/logrus"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	gHttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

func buildCloneURL(githubURL, org, repo string) string {
	u, err := url.Parse(githubURL)
	if err != nil {
		log.WithError(err).Fatal("parsing github URL")
	}
	u.Path = path.Join(org, repo+".git")
	return u.String()
}

// Request raises a PR against the deploy repo with the configuration to be deployed
func Request(namespace, manifestDir, sha, githubURL, apiURL, org, repo, token string) {
	// Create in-mem FS w/ cloned deployments repo
	ctx := context.Background()
	r, err := gh.GetRepo(ctx, buildCloneURL(githubURL, org, repo), token)
	if err != nil {
		log.WithError(err).Fatal() // err has enough info
	}

	// Create a new branch to the current HEAD
	headRef, err := r.Head()
	if err != nil {
		log.WithError(err).Error("getting HEAD")
	}
	branchName := "deploy-" + sha
	branchRefName := plumbing.ReferenceName(path.Join("refs", "heads", branchName))
	ref := plumbing.NewHashReference(branchRefName, headRef.Hash())
	err = r.Storer.SetReference(ref)
	if err != nil {
		log.WithError(err).Error("setting reference")
	}

	// get a working tree and switch to the newly created branch
	w, err := r.Worktree()
	if err != nil {
		log.WithError(err).Fatal("creating worktree from repository")
	}
	err = w.Checkout(&git.CheckoutOptions{
		Hash: ref.Hash(),
	})

	// Delete destination directory
	destDir := "/" + namespace
	info, err := w.Filesystem.Lstat(destDir)
	if err == nil && info.IsDir() {
		err = filesystem.Remove(destDir, w.Filesystem)
		if err != nil {
			log.WithError(err).Fatal("removing destination directory")
		}
	}

	// Copy all of manifestDir in to our in-mem FS
	err = filesystem.Copy(manifestDir, destDir, w.Filesystem)
	if err != nil {
		log.WithError(err).Fatal("copying files to in-mem FS")
	}

	// TODO: resolve; get registry; etc.

	// git add -A
	_, err = w.Add(".")
	if err != nil {
		log.WithError(err).Fatal("'git add' files")
	}

	// git commit
	commit, err := w.Commit(fmt.Sprintf("%s at %s", namespace, sha), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Robot",
			Email: "robot",
			When:  time.Now(),
		},
	})
	obj, _ := r.CommitObject(commit)
	log.WithField("object", obj).Info("commit object")

	err = r.Storer.SetReference(plumbing.NewHashReference(branchRefName, obj.Hash))
	if err != nil {
		log.WithError(err).Error("setting reference")
		return
	}

	// check if commit was empty?
	if commit == plumbing.ZeroHash {
		log.Error("ZeroHash commit returned")
		return
	}
	if err != nil {
		log.WithError(err).Error("commit")
		return
	}

	// Push branch to remote
	err = r.Push(&git.PushOptions{
		Auth: &gHttp.BasicAuth{Username: "none", Password: token},
	})
	if err != nil {
		log.WithError(err).Error("push")
		return
	}

	// Raise PR ["deployments" repo] with requested changes
	client, err := gh.NewClient(ctx, apiURL, token)

	title := namespace + " deployment request"
	head := branchName
	base := "master"
	body := "Deployment request for " + namespace + " at " + sha

	pr, _, err := client.PullRequests.Create(ctx, org, repo, &github.NewPullRequest{
		Title: &title,
		Head:  &head,
		Base:  &base,
		Body:  &body,
	})
	if err != nil {
		log.WithError(err).Error("creating PR")
	} else {
		log.WithField("pullRequest", *pr.Number).Info("pull request raised")
	}
}
