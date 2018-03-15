package main

import (
	"log"
	"os"
	"time"

	"github.com/redbadger/deploy/copy"
	gh "github.com/redbadger/deploy/github"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	gHttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

const (
	githubTokenEnvVar = "GITHUB_TOKEN"
	projectNameEnvVar = "PROJECT_NAME"
	apiURL            = "https://api.github.com/"
	org               = "org"
	repoName          = "repo"
	stacksDir         = "stacks"
)

func main() {
	//  TODO: extract the right variables from the environment
	githubToken, present := os.LookupEnv(githubTokenEnvVar)
	if !present {
		log.Fatalf("Environment variable %s is not exported.\n", githubTokenEnvVar)
	}
	projectName, present := os.LookupEnv(projectNameEnvVar)
	if !present {
		log.Fatalf("Environment variable %s is not exported.\n", projectNameEnvVar)
	}

	// Create in-mem FS w/ cloned deployments repo
	r, err := gh.GetRepo(apiURL, org, repoName, githubToken, "master", "master")
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

	sourceDir := stacksDir + "/" + projectName

	err = copy.Copy(sourceDir, "/"+projectName, w.Filesystem)
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
		Auth: &gHttp.BasicAuth{Username: "none", Password: githubToken},
		// RefSpecs: []config.RefSpec{"+refs/heads/*:refs/remotes/origin/*"},
	})
	if err != nil {
		log.Printf("Error pushing: %v", err)
	}
	// Raise PR ["deployments" repo] with requested changes
}
