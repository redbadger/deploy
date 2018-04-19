package agent

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"gopkg.in/go-playground/webhooks.v3"
	webhook "gopkg.in/go-playground/webhooks.v3/github"
	"gopkg.in/src-d/go-billy.v4"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/redbadger/deploy/filesystem"
	gh "github.com/redbadger/deploy/github"
	"github.com/redbadger/deploy/kubectl"
	"github.com/redbadger/deploy/model"
)

// Agent runs deploy as a bot
func Agent(port uint16, path, token, secret string) {
	hook := webhook.New(&webhook.Config{Secret: secret})
	hook.RegisterEvents(createPullRequestHandler(token), webhook.PullRequestEvent)

	err := webhooks.Run(hook, ":"+strconv.FormatUint(uint64(port), 10), path)
	if err != nil {
		log.Fatalln(fmt.Errorf("cannot listen for webhook: %v", err))
	}
}

var patterns = []string{"*.yml", "*.yaml"}

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

func statusUpdater(
	ctx context.Context, client *github.Client, context, login, repo, ref string,
) func(state, desc string) error {
	return func(state, desc string) error {
		log.Printf("%s: %s", state, desc)
		status := github.RepoStatus{
			State:       &state,
			Description: &desc,
			Context:     &context,
		}
		_, _, err := client.Repositories.CreateStatus(ctx, login, repo, ref, &status)

		if err != nil {
			return fmt.Errorf("error updating status %v", err)
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

	updateStatus := statusUpdater(
		ctx, client, "deploy", req.Owner, req.Repo, req.HeadSHA,
	)

	err = updateStatus("pending", "deployment started")
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
	commit, _, err := client.Repositories.Merge(
		ctx, req.Owner, req.Repo,
		&github.RepositoryMergeRequest{
			Base:          &req.HeadRef, // this PR HEAD
			Head:          &head,
			CommitMessage: nil,
		})
	if err != nil {
		return fmt.Errorf("error merging master: %v", err)
	}
	if commit.SHA != nil {
		// we merged master, so abandon this request after updating status
		err = updateStatus(
			"success",
			fmt.Sprintf("master was merged so deployment will occur on commit %s", *commit.SHA),
		)
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

	var succeeded []string
	for _, dir := range changedDirs {
		log.Printf("Walking %s\n", dir)
		var contents []string
		err = filesystem.Walk(w.Filesystem, dir, visit(&contents))
		if err != nil {
			return fmt.Errorf("error walking filesystem %v", err)
		}
		if len(contents) > 0 {
			err = kubectl.Apply(dir, strings.Join(contents, "---\n"))
			if err != nil {
				err1 := updateStatus("error", fmt.Sprintf("deployment of %s failed: %v", dir, err))
				if err1 != nil {
					return
				}
			} else {
				succeeded = append(succeeded, dir)
			}
		}
	}
	msg := fmt.Sprintf("deployment of %s succeeded", succeeded)
	err1 := updateStatus("success", msg)
	if err1 != nil {
		return
	}
	_, _, err = client.PullRequests.Merge(ctx, req.Owner, req.Repo, int(req.Number), msg, nil)
	if err != nil {
		return
	}
	return
}

func consume(ch chan *model.DeploymentRequest) {
	for {
		err := deploy(<-ch)
		if err != nil {
			log.Printf("error executing deployment request %v", err)
		}
	}
}

func createPullRequestHandler(token string) func(interface{}, webhooks.Header) {
	ch := make(chan *model.DeploymentRequest, 100)
	go consume(ch)
	return func(payload interface{}, header webhooks.Header) {
		pl := payload.(webhook.PullRequestPayload)
		switch pl.Action {
		case "opened", "synchronize":
			pr := pl.PullRequest
			log.Printf("Received %s on PR #%d, SHA %s", pl.Action, pl.PullRequest.Number, pr.Head.Sha)
			ch <- &model.DeploymentRequest{
				URL:      pl.Repository.URL,
				CloneURL: pl.Repository.CloneURL,
				Token:    token,
				Owner:    pl.Repository.Owner.Login,
				Repo:     pl.Repository.Name,
				Number:   pl.PullRequest.Number,
				HeadRef:  pr.Head.Ref,
				HeadSHA:  pr.Head.Sha,
				BaseSHA:  pr.Base.Sha,
			}
		default:
			log.Printf("Ignore %s on PR #%d", pl.Action, pl.PullRequest.Number)
		}
	}
}
