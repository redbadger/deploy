package agent

import (
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
	"gopkg.in/go-playground/webhooks.v5/github"

	"github.com/redbadger/deploy/model"
)

// Agent runs deploy as a bot
func Agent(port uint16, path, token, secret string) {
	http.HandleFunc("/webhooks", createWebhookHandler(secret, token))
	http.HandleFunc("/healthz", createHealthHandler())

	address := ":" + strconv.FormatUint(uint64(port), 10)
	log.Infof("Listening on address %s", address)
	err := http.ListenAndServe(address, nil)
	if err != nil {
		log.WithError(err).Fatal("listening for webhook")
	}
}

func consume(ch chan *model.DeploymentRequest) {
	for {
		err := handleDeploymentRequest(<-ch)
		if err != nil {
			log.WithError(err).Error("executing deployment request")
		}
	}
}

func createWebhookHandler(secret, token string) http.HandlerFunc {
	ch := make(chan *model.DeploymentRequest, 100)
	go consume(ch)

	hook, _ := github.New(github.Options.Secret(secret))

	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := hook.Parse(r, github.PullRequestEvent)
		if err != nil {
			if err == github.ErrEventNotFound {
				http.Error(w, "only 'pull_request' events are supported", http.StatusNotAcceptable)
			} else {
				http.Error(w, err.Error(), http.StatusBadRequest)
			}

			return
		}

		switch pl := payload.(type) {
		case github.PullRequestPayload:
			pr := pl.PullRequest
			myLog := log.WithFields(log.Fields{
				"action":      pl.Action,
				"pullRequest": pr.Number,
			})

			switch pl.Action {
			case "opened", "synchronize":
				myLog.WithField("sha", pr.Head.Sha).Info("actioning webhook")
				ch <- &model.DeploymentRequest{
					URL:      pl.Repository.URL,
					CloneURL: pl.Repository.CloneURL,
					Token:    token,
					Owner:    pl.Repository.Owner.Login,
					Repo:     pl.Repository.Name,
					Number:   pr.Number,
				}
			default:
				myLog.Info("webhook ignored")
			}
		default:
			log.Info("payload not supported")
		}
	}
}

func createHealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}
