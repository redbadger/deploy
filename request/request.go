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

func buildCloneURL(githubURL, org, repo string) string {
	u, err := url.Parse(githubURL)
	if err != nil {
		log.Fatalf("cannot parse github URL: %v\n", err)
	}
	u.Path = path.Join(org, repo+".git")
	return u.String()
}

// Request raises a PR against the deploy repo with the configuration to be deployed
func Request(stacksDir, project, sha, githubURL, apiURL, org, repo, token string) {
	// Create in-mem FS w/ cloned deployments repo
	cloneURL := buildCloneURL(githubURL, org, repo)
	r, err := gh.GetRepo(cloneURL, org, repo, token, "master", "master")
	if err != nil {
		log.Fatalln(err) // err has enough info
	}

	// Create a new branch to the current HEAD
	headRef, err := r.Head()
	if err != nil {
		log.Printf("error getting HEAD, %v", err)
	}
	branchName := "deploy-" + sha
	branchRefName := plumbing.ReferenceName(path.Join("refs", "heads", branchName))
	ref := plumbing.NewHashReference(branchRefName, headRef.Hash())
	err = r.Storer.SetReference(ref)
	if err != nil {
		log.Printf("error setting reference, %v", err)
	}

	// get a working tree and switch to the newly created branch
	w, err := r.Worktree()
	if err != nil {
		log.Fatalf("cannot create worktree from repository! %v", err)
	}
	err = w.Checkout(&git.CheckoutOptions{
		Hash: ref.Hash(),
	})

	// Delete destination directory
	destDir := "/" + project
	info, err := w.Filesystem.Lstat(destDir)
	if err == nil && info.IsDir() {
		err = filesystem.Remove(destDir, w.Filesystem)
		if err != nil {
			log.Fatalf("cannot remove destination directory: %v", err)
		}
	}

	// Copy all of sourceDir in to our in-mem FS
	sourceDir := path.Join(stacksDir, project)
	err = filesystem.Copy(sourceDir, destDir, w.Filesystem)
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

	err = r.Storer.SetReference(plumbing.NewHashReference(branchRefName, obj.Hash))
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
	head := branchName
	base := "master"
	body := "Deployment request for " + project + " at " + sha

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
