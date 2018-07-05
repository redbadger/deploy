package agent

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/google/go-github/github"
	log "github.com/sirupsen/logrus"

	"github.com/redbadger/deploy/git"
	gh "github.com/redbadger/deploy/github"
	"github.com/redbadger/deploy/kubectl"
	"github.com/redbadger/deploy/model"
)

type updater func(state, msg, comment string) (err error)

var patterns = []string{"*.yml", "*.yaml"}
var namespaceTemplate = `---
apiVersion: v1
kind: Namespace
metadata:
  name: %s`

func visit(files *[]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // can't walk here, but continue walking elsewhere
		}
		if info.IsDir() {
			return nil // not a file.  ignore.
		}
		for _, pattern := range patterns {
			matched, err := filepath.Match(pattern, info.Name())
			if err != nil {
				return err // malformed pattern, this is fatal.
			}
			if matched {
				contents, err := ioutil.ReadFile(path)
				if err != nil {
					log.WithError(err).Fatal("reading file")
				}
				*files = append(*files, string(contents))
			}
		}
		return nil
	}
}

func createUpdater(
	ctx context.Context, client *github.Client, context, owner, repo string, number int, ref string,
) updater {
	return func(state, msg, comment string) (err error) {
		log.WithFields(log.Fields{
			"state":   state,
			"message": msg,
		}).Info("updating github")
		status := github.RepoStatus{
			State:       &state,
			Description: &msg,
			Context:     &context,
		}
		_, _, err = client.Repositories.CreateStatus(ctx, owner, repo, ref, &status)
		if err != nil {
			return fmt.Errorf("error updating status %v", err)
		}

		_, _, err = client.Issues.CreateComment(
			ctx, owner, repo, number,
			&github.IssueComment{Body: &comment},
		)
		if err != nil {
			return fmt.Errorf("error creating comment %v", err)
		}

		return nil
	}
}

func handleDeploymentRequest(req *model.DeploymentRequest) (err error) {
	ctx := context.Background()
	apiURL, err := APIRoot(req.URL)
	if err != nil {
		return
	}
	client, err := gh.NewClient(ctx, apiURL, req.Token)
	if err != nil {
		return
	}

	// we need to get the PR again, because there is a bug in the webhook payload
	// where the mergeable_state is `clean` when it should be `blocked`
	pr, _, err := client.PullRequests.Get(ctx, req.Owner, req.Repo, int(req.Number))
	if err != nil {
		return
	}

	headSha := *pr.Head.SHA
	update := createUpdater(ctx, client, "deploy", req.Owner, req.Repo, int(req.Number), headSha)

	state := *pr.MergeableState
	switch state {
	case "dirty", "blocked":
		err = update("error", "Deployment blocked!", "PR is currently blocked so doing nothing")
		log.WithField("MergeableState", state).Info("pull request is currently blocked so doing nothing")
		return
	case "unknown":
		err = update("error", "Deployment cannot proceed", "Retry not yet implemented")
		log.WithField("MergeableState", state).Info("periodically fetch")
		return
		// TODO implement this
	case "behind", "unstable", "has_hooks", "clean":
		log.WithField("MergeableState", state).Info("deploy starting")
		deploy(ctx, client, req, pr, update)
		return
	}
	return
}

func deploy(ctx context.Context, client *github.Client,
	req *model.DeploymentRequest, pr *github.PullRequest,
	update updater,
) (err error) {
	msg := "Deployment started!"
	err = update("pending", msg, msg)
	if err != nil {
		return
	}

	// merge master
	log.Info("merging master")
	headRef := *pr.Head.Ref
	master := "master"
	mergeReq := github.RepositoryMergeRequest{
		Base:          &headRef,
		Head:          &master,
		CommitMessage: nil,
	}
	commit, _, err := client.Repositories.Merge(ctx, req.Owner, req.Repo, &mergeReq)
	if err != nil {
		return fmt.Errorf("error merging master: %v", err)
	}
	if commit.SHA != nil {
		// we merged master, so abandon this request after updating status
		msg := fmt.Sprintf("Master was merged so deployment will occur on commit %s", *commit.SHA)
		err = update("success", msg, msg)
		return
	}

	tmpDir, err := ioutil.TempDir("/tmp", headRef)
	if err != nil {
		log.WithError(err).Fatal("creating tmp dir")
	}

	defer os.RemoveAll(tmpDir)

	cloneURL, err := url.Parse(req.CloneURL)
	if err != nil {
		log.WithError(err).Fatal("parsing github URL")
	}
	authURL := url.URL{
		Scheme: cloneURL.Scheme,
		User:   url.UserPassword("dummy", req.Token),
		Host:   cloneURL.Host,
	}

	credFile := path.Join(tmpDir, "git-credentials")
	err = ioutil.WriteFile(credFile, []byte(authURL.String()), 0600)
	if err != nil {
		log.WithError(err).Fatal("writing credentials file")
	}

	config := fmt.Sprintf("credential.helper=store --file=%s", credFile)
	srcDir := path.Join(tmpDir, "src")
	git.MustRun(tmpDir, "clone",
		"--branch", headRef,
		"--config", config,
		cloneURL.String(),
		srcDir,
	)

	baseSHA := *pr.Base.SHA
	changedDirs, err := git.GetChangedDirectories(srcDir, baseSHA)
	if err != nil {
		return fmt.Errorf("error identifying changed top level directories: %v", err)
	}

	succeeded := make(map[string]string)
	for _, dir := range changedDirs {
		log.WithField("directory", dir).Info("Walking dir")
		var contents []string
		err = filepath.Walk(path.Join(srcDir, dir), visit(&contents))
		if err != nil {
			return fmt.Errorf("error walking filesystem %v", err)
		}
		if len(contents) > 0 {
			manifests := joinManifests(dir, contents)
			out, err := apply(dir, manifests)
			if err != nil {
				msg := fmt.Sprintf("deployment of %s failed: %v", dir, err)
				comment := fmt.Sprintf("Deployment failed!\n%s", out)
				err1 := update("error", msg, comment)
				if err1 != nil {
					return fmt.Errorf("%v\n%v", err, err1)
				}
				return err
			}
			succeeded[dir] = out
		}
	}

	msg = fmt.Sprintf("deployment of %s succeeded", keys(succeeded))
	comment := fmt.Sprintf("Deployment succeeded!\n%s", formatResults(succeeded))
	err1 := update("success", msg, comment)
	if err1 != nil {
		return
	}

	_, _, err = client.PullRequests.Merge(ctx, req.Owner, req.Repo, int(req.Number), msg, nil)
	if err != nil {
		return
	}

	_, err = client.Git.DeleteRef(ctx, req.Owner, req.Repo, fmt.Sprintf("heads/%s", headRef))
	if err != nil {
		return
	}

	return
}

// By prepending a default namespace template we will loose any metadata
// on existing namespaces. We need to find a solution to this when it
// becomes a problem.
func joinManifests(namespace string, manifests []string) string {
	namespaceManifest := fmt.Sprintf(namespaceTemplate, namespace)
	manifests = append([]string{namespaceManifest}, manifests...)
	return strings.Join(manifests, "\n---\n")
}

func apply(namespace, manifest string) (out string, err error) {
	out, err = kubectl.Apply(namespace, manifest, true)
	if err == nil {
		out, err = kubectl.Apply(namespace, manifest, false)
	}
	return
}

func keys(m map[string]string) (keys []string) {
	keys = make([]string, len(m))

	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return
}

var sub = regexp.MustCompile("\n")

func formatResults(in map[string]string) (out string) {
	out = ""
	for _, k := range keys(in) {
		out += fmt.Sprintf("* %s:\n\t%v\n\n", k, sub.ReplaceAllString(in[k], "\n\t"))
	}
	return
}
