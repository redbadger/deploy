package request

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path"

	"github.com/google/go-github/github"
	gh "github.com/redbadger/deploy/github"
	log "github.com/sirupsen/logrus"
)

func buildCloneURL(githubURL, org, repo string) *url.URL {
	u, err := url.Parse(githubURL)
	if err != nil {
		log.WithError(err).Fatal("parsing github URL")
	}
	u.Path = path.Join(org, repo+".git")
	return u
}

// Request raises a PR against the deploy repo with the configuration to be deployed
func Request(namespace, manifestDir, sha, githubURL, apiURL, org, repo, token string) {
	branchName := fmt.Sprintf("deploy-%s", sha)
	tmpDir, err := ioutil.TempDir("/tmp", branchName)
	if err != nil {
		log.WithError(err).Fatal("creating tmp dir")
	}

	defer os.RemoveAll(tmpDir)

	cloneURL := buildCloneURL(githubURL, org, repo)
	authURL := url.URL{
		Scheme: cloneURL.Scheme,
		User:   url.UserPassword("dummy", token),
		Host:   cloneURL.Host,
	}

	credFile := path.Join(tmpDir, "git-credentials")
	err = ioutil.WriteFile(credFile, []byte(authURL.String()), 0600)
	if err != nil {
		log.WithError(err).Fatal("writing credentials file")
	}

	config := fmt.Sprintf("credential.helper=store --file=%s", credFile)
	srcDir := path.Join(tmpDir, "src")
	err = git(tmpDir, "clone",
		"--config", config,
		cloneURL.String(),
		srcDir,
	)
	if err != nil {
		log.WithError(err).Fatal("cloning cluster repository")
	}

	err = git(srcDir, "checkout",
		"-b", branchName,
	)
	if err != nil {
		log.WithError(err).Fatalf("creating new branch: %s", branchName)
	}

	err = git(srcDir, "rm", "-r", namespace)
	if err != nil {
		log.WithError(err).Fatalf("removing: %s", namespace)
	}

	err = copyDir(manifestDir, path.Join(srcDir, namespace))
	if err != nil {
		log.WithError(err).Fatal("copying manifests to repo")
	}

	err = git(srcDir, "add", "--all")
	if err != nil {
		log.WithError(err).Fatalf("adding: %s", namespace)
	}

	err = git(srcDir, "commit",
		"--message", fmt.Sprintf("%s at %s", namespace, sha),
		"--author", "Robot <robot>",
		"--allow-empty",
	)
	if err != nil {
		log.WithError(err).Fatal("commit")
	}

	err = git(srcDir, "push", "origin", branchName)
	if err != nil {
		log.WithError(err).Fatal("push")
	}

	// Raise PR ["deployments" repo] with requested changes

	ctx := context.Background()
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

func git(workingDir string, args ...string) (err error) {
	log.Info("git", args)
	cmd := exec.Command("git", args...)
	cmd.Env = os.Environ()
	cmd.Dir = workingDir
	err = cmd.Run()
	return
}
