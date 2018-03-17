package request

import (
	"context"
	"log"
	"net/url"
	"path"
	"time"

	"github.com/google/go-github/github"
	"github.com/redbadger/deploy/filesystem"
	gh "github.com/redbadger/deploy/github"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	gHttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

// Request raises a PR against the deploy repo with the configuration to be deployed
func Request(token, project, githubURL, apiURL, org, repo, stacksDir string) {
	// Create in-mem FS w/ cloned deployments repo
	u, err := url.Parse(githubURL)
	if err != nil {
		log.Fatalf("cannot parse github URL: %v\n", err)
	}
	u.Path = path.Join(org, repo+".git")
	cloneURL := u.String()
	r, err := gh.GetRepo(cloneURL, org, repo, token, "master", "master")
	if err != nil {
		log.Fatalln(err) // err has enough info
	}

	w, err := r.Worktree()
	if err != nil {
		log.Fatalf("cannot create worktree from repository! %v", err)
	}

	// Create a new branch to the current HEAD
	headRef, err := r.Head()
	ref := plumbing.NewHashReference("refs/heads/newdeployment", headRef.Hash()) // TODO: set canonical + unique branch name
	err = r.Storer.SetReference(ref)
	if err != nil {
		log.Printf("error setting reference, %v", err)
	}

	// Switch to newly created branch
	err = w.Checkout(&git.CheckoutOptions{
		Hash: ref.Hash(),
	})

	// Delete destination directory
	dest := "/" + project
	err = filesystem.Remove(dest, w.Filesystem)
	if err != nil {
		log.Fatalf("cannot remove destination directory: %v", err)
	}
	// Copy all of sourceDir in to our in-mem FS
	sourceDir := path.Join(stacksDir, project)
	log.Printf("copying from %s to %s", sourceDir, dest)
	err = filesystem.Copy(sourceDir, dest, w.Filesystem)
	if err != nil {
		log.Fatalf("cannot copy files in to in-mem FS! %v", err)
	}

	// TODO: resolve; get registry; etc.

	// git add -A
	_, err = w.Add(".")
	if err != nil {
		log.Fatalf("cannot 'git add' files! %v", err)
	}

	// git commit
	commit, err := w.Commit("Commit message!", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Robot",
			Email: "robot",
			When:  time.Now(),
		},
	})
	obj, _ := r.CommitObject(commit)
	log.Printf("commit obj: %v", obj)

	err = r.Storer.SetReference(plumbing.NewReferenceFromStrings("refs/heads/newdeployment", obj.Hash.String()))
	if err != nil {
		log.Printf("error setting reference, %v", err)
	}

	// check if commit was empty?
	if commit == plumbing.ZeroHash {
		log.Print("ZeroHash commit returned, exiting.")
		return
	}
	if err != nil {
		log.Printf("error returned from commit: %v", err)
	}

	// Push branch to remote
	err = r.Push(&git.PushOptions{
		Auth: &gHttp.BasicAuth{Username: "none", Password: token},
	})
	if err != nil {
		log.Printf("error pushing: %v", err)
	}
	// Raise PR ["deployments" repo] with requested changes
	client, err := gh.NewClient(apiURL, token)

	title := project + " deployment request"
	head := "newdeployment"
	base := "master"
	body := "# Hello"

	pr, _, err := client.PullRequests.Create(context.Background(), org, repo, &github.NewPullRequest{
		Title: &title,
		Head:  &head,
		Base:  &base,
		Body:  &body,
	})
	if err != nil {
		log.Printf("Error creating PR: %v", err)
	} else {
		log.Printf("Pull request #%d raised!", *pr.Number)
	}
}