package request

import (
	"context"
	"log"
	"net/url"
	"path"
	"time"

	"github.com/google/go-github/github"
	"github.com/redbadger/deploy/copy"
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
		log.Fatalf("Could not clone repo! %v", err)
	}

	// Copy all of sourceDir in to our in-mem FS
	w, err := r.Worktree()
	if err != nil {
		log.Fatal("Could not create worktree from repository!")
	}

	// Create a new branch to the current HEAD
	headRef, err := r.Head()
	ref := plumbing.NewHashReference("refs/heads/newdeployment", headRef.Hash()) // TODO: set canonical + unique branch name
	err = r.Storer.SetReference(ref)
	if err != nil {
		log.Printf("Error setting reference, %v", err)
	}

	// Switch to newly created branch
	err = w.Checkout(&git.CheckoutOptions{
		Hash: ref.Hash(),
	})

	sourceDir := stacksDir + "/" + project

	err = copy.Copy(sourceDir, "/"+project, w.Filesystem)
	if err != nil {
		log.Fatalf("Could not copy files in to in-mem FS! %v", err)
	}
	// TODO: resolve; get registry; etc.

	// git add -A
	_, err = w.Add(".")
	if err != nil {
		log.Fatalf("Could not 'git add' files! %v", err)
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
		log.Printf("Error setting reference, %v", err)
	}

	// check if commit was empty?
	if commit == plumbing.ZeroHash {
		log.Print("ZeroHash commit returned, exiting.")
		return
	}
	if err != nil {
		log.Printf("Error returned from commit: %v", err)
	}

	// Push branch to remote
	err = r.Push(&git.PushOptions{
		Auth: &gHttp.BasicAuth{Username: "none", Password: token},
		// RefSpecs: []config.RefSpec{"+refs/heads/*:refs/remotes/origin/*"},
	})
	if err != nil {
		log.Printf("Error pushing: %v", err)
	}
	// Raise PR ["deployments" repo] with requested changes
	client, err := gh.NewClient(apiURL, token)

	title := "New PR"
	head := "newdeployment"
	base := "master"
	body := "# Hello"

	pr, _, err := client.PullRequests.Create(context.Background(), "org", "repo", &github.NewPullRequest{
		Title: &title,
		Head:  &head,
		Base:  &base,
		Body:  &body,
	})
	if err != nil {
		log.Printf("Error creating PR: %v", err)
	} else {
		log.Printf("PR obj: %v", pr)
	}
}
