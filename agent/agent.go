package agent

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/src-d/go-billy.v4"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/redbadger/deploy/fsWalker"
	gh "github.com/redbadger/deploy/github"
	"github.com/redbadger/deploy/kubectl"
	"gopkg.in/go-playground/webhooks.v3"
	"gopkg.in/go-playground/webhooks.v3/github"
)

// Agent runs deploy as a bot
func Agent(port uint16, path, token, secret string) {
	hook := github.New(&github.Config{Secret: secret})
	hook.RegisterEvents(handlePullRequest(token), github.PullRequestEvent)

	err := webhooks.Run(hook, ":"+strconv.FormatUint(uint64(port), 10), path)
	if err != nil {
		log.Fatalln(fmt.Errorf("cannot listen for webhook: %v", err))
	}
}

var patterns = []string{"*.yml", "*.yaml"}

func visit(files *[]string) fsWalker.WalkFunc {
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

func handlePullRequest(token string) func(interface{}, webhooks.Header) {
	return func(payload interface{}, header webhooks.Header) {
		pl := payload.(github.PullRequestPayload)
		pr := pl.PullRequest

		log.Printf("\nPR #%d, SHA %s\n", pl.PullRequest.Number, pl.PullRequest.Head.Sha)

		r, err := gh.GetRepo(
			pl.Repository.CloneURL,
			pl.Repository.Owner.Login,
			pl.Repository.Name,
			token,
			pr.Head.Sha,
			pr.Base.Sha,
		)
		if err != nil {
			log.Fatalf("error getting repo %v\n", err)
		}

		w, err := r.Worktree()
		if err != nil {
			err = fmt.Errorf("error getting work tree: %v", err)
			return
		}
		err = w.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(pr.Head.Sha),
		})
		if err != nil {
			err = fmt.Errorf("error checking out %s: %v", pr.Head.Sha, err)
			return
		}

		changedDirs, err := gh.GetChangedDirectories(r, pr.Head.Sha,
			pr.Base.Sha)
		if err != nil {
			err = fmt.Errorf("error identifying changed top level directories: %v", err)
			return
		}

		for _, dir := range changedDirs {
			log.Printf("Walking %s\n", dir)
			var contents []string
			err = fsWalker.Walk(w.Filesystem, dir, visit(&contents))
			if err != nil {
				log.Fatalf("error walking filesystem %v\n", err)
			}
			if len(contents) > 0 {
				err = kubectl.Apply(dir, strings.Join(contents, "---\n"))
				if err != nil {
					log.Fatalf("error applying manifests to the cluster: %v\n", err)
				}
			}
		}
	}
}
