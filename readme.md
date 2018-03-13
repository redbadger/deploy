## Deploy to Kubernetes through a Github Repo

A small go application that:

1.  watches for PR updates on a webhook
1.  clones the repo to an in-memory filesystem
1.  checks out the commit SHA
1.  walks down any top-level directories that contain changes
1.  gathers yaml files (however they are nested)
1.  applies the manifests to a Kubernetes cluster using `kubctl`.

This is an experiment to demonstrate how a CI/CD system might deploy to environments by opening a Pull Request on a "cluster repo" with the intention "please will you accept this configuration". It's early days and not production ready.

TODO:

1.  Implement a mark-and-sweep garbage collector, similar to [`kubecfg`](https://github.com/ksonnet/kubecfg). Currently any removed manifests will not result in the resources being removed from the cluster.
1.  Implement an image resolver, similar to [`kubecfg`](https://github.com/ksonnet/kubecfg). This allows idempotent deploys even for images whose tags have changed but their content hasn't (e.g. if you're using a monorepo and the SHA is used as the tag).

Note: we can't currently use [`kubecfg`](https://github.com/ksonnet/kubecfg) as it stands, because it doesn't support accepting manifests from `stdin` (and as there are no file extensions to look at, it wouldn't know whether they were `yaml`, `json` or `jsonnet` anyway). We could raise a PR to add this functionality, or use it as a library. Jury is still out.

To install:

```bash
go get github.com/redbadger/deploy
```

To run:

```bash
export PERSONAL_ACCESS_TOKEN=<personal access token>
export DEPLOY_SECRET=<webhook secret>
deploy
```

Typical output:

```
> deploy
2018/03/13 15:52:21 INFO: Listening on addr: :3016 path: /webhooks
2018/03/13 15:52:40 INFO: Webhook received
2018/03/13 15:52:40 INFO: Parsing Payload...
2018/03/13 15:52:40 INFO: Checking secret
2018/03/13 15:52:40
PR #1, SHA 304b14faac3130bba0e8da4c3bd84af5754de7d5
2018/03/13 15:52:43 Walking guestbook
deployment "redis" configured
service "redis" unchanged
service "guestbook-ui" unchanged
deployment "guestbook-ui" unchanged
```
