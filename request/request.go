package request

import (
	"bytes"
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
	git(tmpDir, "clone",
		"--config", config,
		"--config", "user.email=robot",
		"--config", "user.name=Robot",
		cloneURL.String(),
		srcDir,
	)
	git(srcDir, "checkout", "-b", branchName)
	git(srcDir, "rm", "-r", "--ignore-unmatch", namespace)

	err = copyDir(manifestDir, path.Join(srcDir, namespace))
	if err != nil {
		log.WithError(err).Fatal("copying manifests to repo")
	}

	git(srcDir, "add", "--all")
	git(srcDir, "commit",
		"--message", fmt.Sprintf("%s at %s", namespace, sha),
		"--allow-empty",
	)
	git(srcDir, "push", "origin", branchName)

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

func git(workingDir string, args ...string) {
	log.Info("git", args)
	cmd := exec.Command("git", args...)
	cmd.Env = os.Environ()
	cmd.Dir = workingDir
	var o, e bytes.Buffer
	cmd.Stderr = &e
	cmd.Stdout = &o
	err := cmd.Run()
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"stdout": o.String(),
			"stderr": e.String(),
		}).Fatal()
	}
}
