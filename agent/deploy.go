package agent

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/google/go-github/github"
	"gopkg.in/src-d/go-billy.v4"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/redbadger/deploy/filesystem"
	gh "github.com/redbadger/deploy/github"
	"github.com/redbadger/deploy/kubectl"
	"github.com/redbadger/deploy/model"
)

var patterns = []string{"*.yml", "*.yaml"}
var namespaceTemplate = `---
apiVersion: 1
kind: Namespace
metadata:
  name: %s`

func visit(files *[]string) filesystem.WalkFunc {
	return func(fs billy.Filesystem, path string, info os.FileInfo, err error) error {
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
				f, err := fs.Open(path)
				if err != nil {
					log.Fatalf("error opening file %v\n", err)
				}
				t, err := ioutil.ReadAll(f)
				if err != nil {
					log.Fatalf("cannot read file %v\n", err)
				}
				ts := string(t)
				if !strings.HasSuffix(ts, "\n") {
					ts += "\n"
				}
				*files = append(*files, ts)
			}
		}
		return nil
	}
}

func updater(
	ctx context.Context, client *github.Client, context, owner, repo string, number int, ref string,
) func(state, msg, comment string) (err error) {
	return func(state, msg, comment string) (err error) {
		log.Printf("%s: %s", state, msg)
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

func deploy(req *model.DeploymentRequest) (err error) {
	ctx := context.Background()
	apiURL, err := APIRoot(req.URL)
	if err != nil {
		return
	}
	client, err := gh.NewClient(ctx, apiURL, req.Token)
	if err != nil {
		return
	}

	update := updater(ctx, client, "deploy", req.Owner, req.Repo, int(req.Number), req.HeadSHA)

	msg := "Deployment started!"
	err = update("pending", msg, msg)
	if err != nil {
		return
	}

	r, err := gh.GetRepo(ctx, req.CloneURL, req.Token)
	if err != nil {
		return fmt.Errorf("error getting repo %v", err)
	}

	// merge master
	log.Println("merging master")
	head := "master"
	mergeReq := github.RepositoryMergeRequest{
		Base:          &req.HeadRef, // this PR HEAD
		Head:          &head,
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

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("error getting work tree: %v", err)
	}

	err = w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(req.HeadSHA),
	})
	if err != nil {
		return fmt.Errorf("error checking out %s: %v", req.HeadSHA, err)
	}

	changedDirs, err := gh.GetChangedDirectories(r, req.HeadSHA, req.BaseSHA)
	if err != nil {
		return fmt.Errorf("error identifying changed top level directories: %v", err)
	}

	succeeded := make(map[string]string)
	for _, dir := range changedDirs {
		log.Printf("Walking %s\n", dir)
		var contents []string
		err = filesystem.Walk(w.Filesystem, dir, visit(&contents))
		if err != nil {
			return fmt.Errorf("error walking filesystem %v", err)
		}
		if len(contents) > 0 {
			manifests := joinManifests(contents, dir)
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
	_, err = client.Git.DeleteRef(ctx, req.Owner, req.Repo, fmt.Sprintf("heads/%s", req.HeadRef))
	if err != nil {
		return
	}
	return
}

// By prepending a default namespace template we will loose any metadata
// on existing namespaces. We need to find a solution to this when it
// becomes a problem.
func joinManifests(manifests []string, namespace string) string {
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
