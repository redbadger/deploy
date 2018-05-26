package agent

import (
	"strconv"

	log "github.com/sirupsen/logrus"
	"gopkg.in/go-playground/webhooks.v3"
	webhook "gopkg.in/go-playground/webhooks.v3/github"

	"github.com/redbadger/deploy/model"
)

// Agent runs deploy as a bot
func Agent(port uint16, path, token, secret string) {
	hook := webhook.New(&webhook.Config{Secret: secret})
	hook.RegisterEvents(createPullRequestHandler(token), webhook.PullRequestEvent)

	err := webhooks.Run(hook, ":"+strconv.FormatUint(uint64(port), 10), path)
	if err != nil {
		log.WithError(err).Fatal("listening for webhook")
	}
}

func consume(ch chan *model.DeploymentRequest) {
	for {
		err := deploy(<-ch)
		if err != nil {
			log.WithError(err).Error("executing deployment request")
		}
	}
}

func createPullRequestHandler(token string) func(interface{}, webhooks.Header) {
	ch := make(chan *model.DeploymentRequest, 100)
	go consume(ch)
	return func(payload interface{}, header webhooks.Header) {
		pl := payload.(webhook.PullRequestPayload)
		myLog := log.WithFields(log.Fields{
			"action":      pl.Action,
			"pullRequest": pl.PullRequest.Number,
		})
		switch pl.Action {
		case "opened", "synchronize":
			pr := pl.PullRequest
			myLog.WithField("sha", pr.Head.Sha).Info("actioning webhook")
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
			myLog.Info("webhook ignored")
		}
	}
}
