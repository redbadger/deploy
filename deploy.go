package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/src-d/go-billy.v4"

	"github.com/redbadger/deploy/fsWalker"
	gh "github.com/redbadger/deploy/github"
	"gopkg.in/go-playground/webhooks.v3"
	"gopkg.in/go-playground/webhooks.v3/github"
)

const (
	secretEnvVar = "DEPLOY_SECRET"
	path         = "/webhooks"
	port         = 3016
)

func main() {
	secret, ok := os.LookupEnv(secretEnvVar)
	if ok == false {
		log.Fatalf("Environment variable %s is not exported.", secretEnvVar)
	}

	hook := github.New(&github.Config{Secret: secret})
	hook.RegisterEvents(handlePullRequest, github.PullRequestEvent)

	err := webhooks.Run(hook, ":"+strconv.Itoa(port), path)
	if err != nil {
		fmt.Println(err)
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
					log.Fatalf("Error opening file %v", err)
				}
				t, err := ioutil.ReadAll(f)
				if err != nil {
					log.Fatalf("Cannot read file %v", err)
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

func handlePullRequest(payload interface{}, header webhooks.Header) {
	fmt.Println("Handling PR")
	pl := payload.(github.PullRequestPayload)

	fmt.Printf("PR #%d, SHA %s\n", pl.PullRequest.Number, pl.PullRequest.Head.Sha)
	baseEndpoint, err := url.Parse(pl.Repository.URL)
	if err != nil {
		log.Fatalf("Error parsing api URL %v", err)
	}
	if baseEndpoint.Hostname() == "api.github.com" {
		baseEndpoint.Path = ""
	} else {
		baseEndpoint.Path = "/api/v3"
	}

	fs, changedDirs, err := gh.GetRepo(
		baseEndpoint.String(),
		pl.Repository.Owner.Login,
		pl.Repository.Name,
		pl.PullRequest.Head.Sha,
		pl.PullRequest.Base.Sha,
	)
	if err != nil {
		log.Fatalf("Error getting repo %v", err)
	}

	for _, dir := range changedDirs {
		fmt.Printf("Walking %s\n", dir)
		var contents []string
		err = fsWalker.Walk(fs, dir, visit(&contents))
		if err != nil {
			log.Fatalf("Error walking filesystem %v", err)
		}
		fmt.Println(strings.Join(contents, "---\n"))
	}
}
