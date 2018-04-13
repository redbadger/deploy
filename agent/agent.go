package agent

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
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
)

// Agent runs deploy as a bot
func Agent(port uint16, path, token, secret string) {
	hook := webhook.New(&webhook.Config{Secret: secret})
	hook.RegisterEvents(handlePullRequest(token), webhook.PullRequestEvent)

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

func statusUpdater(ctx context.Context, client *github.Client, context, login, repo, ref string) func(state, desc string) error {
	return func(state, desc string) error {
		log.Printf("%s: %s", state, desc)
		status := github.RepoStatus{
			State:       &state,
			Description: &desc,
			Context:     &context,
		}
		_, _, err := client.Repositories.CreateStatus(ctx, login, repo, ref, &status)

		return err
	}
}

func handlePullRequest(token string) func(interface{}, webhooks.Header) {
	return func(payload interface{}, header webhooks.Header) {
		pl := payload.(webhook.PullRequestPayload)
		pr := pl.PullRequest
		log.Printf("\nPR #%d, SHA %s\n", pl.PullRequest.Number, pl.PullRequest.Head.Sha)

		ctx := context.Background()
		apiURL, err := url.Parse(pl.Repository.StatusesURL)
		if err != nil {
			return
		}
		apiURL.Path = ""
		client, err := gh.NewClient(ctx, apiURL.String(), token)
		if err != nil {
			return
		}

		owner := pl.Repository.Owner.Login
		repo := pl.Repository.Name
		headRef := pr.Head.Sha
		baseRef := pr.Base.Sha
		updateStatus := statusUpdater(ctx, client, "deploy", owner, repo, headRef)

		err = updateStatus("pending", "deployment started")
		if err != nil {
			log.Fatalf("error updating status %v\n", err)
		}

		r, err := gh.GetRepo(pl.Repository.CloneURL, owner, repo, token, headRef, baseRef)
		if err != nil {
			log.Fatalf("error getting repo %v\n", err)
		}

		w, err := r.Worktree()
		if err != nil {
			err = fmt.Errorf("error getting work tree: %v", err)
			return
		}
		err = w.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(headRef),
		})
		if err != nil {
			err = fmt.Errorf("error checking out %s: %v", headRef, err)
			return
		}

		changedDirs, err := gh.GetChangedDirectories(r, headRef, baseRef)
		if err != nil {
			err = fmt.Errorf("error identifying changed top level directories: %v", err)
			return
		}

		for _, dir := range changedDirs {
			log.Printf("Walking %s\n", dir)
			var contents []string
			err = filesystem.Walk(w.Filesystem, dir, visit(&contents))
			if err != nil {
				log.Fatalf("error walking filesystem %v\n", err)
			}
			if len(contents) > 0 {
				err = kubectl.Apply(dir, strings.Join(contents, "---\n"))
				if err != nil {
					err1 := updateStatus("error", fmt.Sprintf("deployment of %s failed: %v", dir, err))
					if err1 != nil {
						log.Fatalf("error updating status %v\n", err1)
					}
				} else {
					err1 := updateStatus("success", fmt.Sprintf("deployment of %s succeeded", dir))
					if err1 != nil {
						log.Fatalf("error updating status %v\n", err1)
					}
				}
			}
		}
	}
}
