# DEPRECATED: Please use Argo-CD instead :-) https://github.com/argoproj/argo-cd

# Deploy to Kubernetes through a Github Repo

A cli command, written in Go, that runs in Kubernetes as an agent (`deploy agent`) or raises a pull request against the cluster repo to request a deployment (`deploy request`).

### `deploy agent`

1.  watches for PR updates on a webhook
1.  clones the repo to a temporary directory
1.  checks out the commit SHA
1.  walks down any top-level directories that contain changes
1.  gathers yaml files (however they are nested)
1.  applies the manifests to a Kubernetes cluster using `kubctl`.

### `deploy request`

1.  checks out the cluster repo specified
1.  copies the specified manifests into a new branch (named with the commit sha)
1.  commits, pushes and raises a PR requesting deployment

### Note:

This is an experiment to demonstrate how a CI/CD system might deploy to environments by opening a Pull Request on a "cluster repo" with the intention "please will you accept this configuration". It's early days and not production ready.

TODO:

1.  Implement a mark-and-sweep garbage collector, similar to [`kubecfg`](https://github.com/ksonnet/kubecfg). Currently any removed manifests will not result in the resources being removed from the cluster.
1.  Implement an image resolver, similar to [`kubecfg`](https://github.com/ksonnet/kubecfg). This allows idempotent deploys even for images whose tags have changed but their content hasn't (e.g. if you're using a monorepo and the SHA is used as the tag).

Note: we can't currently use [`kubecfg`](https://github.com/ksonnet/kubecfg) as it stands, because it doesn't support accepting manifests from `stdin` (and as there are no file extensions to look at, it wouldn't know whether they were `yaml`, `json` or `jsonnet` anyway). We could raise a PR to add this functionality, or use it as a library. Jury is still out.

## To install locally:

```bash
go get github.com/redbadger/deploy

export PERSONAL_ACCESS_TOKEN=<personal access token>
export DEPLOY_SECRET=<webhook secret>
deploy help
deploy help agent
deploy help request
```

## To run the agent in Kubernetes:

`deploy agent` runs on the k8s cluster.
There is an example deployment (for minikube) in the [`k8s/minikube`](./k8s/minikube) directory.
The shell script `deploy.sh` does the following:

- creates a **namespace**
  - with name `deploy-robot`
- creates a **secret**
  - from exported `PERSONAL_ACCESS_TOKEN`
  - from exported `DEPLOY_SECRET`
- creates other resources:
  - **serviceAccount**
    - with name `deploy-robot` - useful for RBAC
  - **deployment**
  - **service**
  - **ingress**
    - backs onto the nginx ingressController which you can enable with `minikube addons enable ingress`
    - uses the hostname `deploy.internal` so you may need to add that to your hosts file so it resolves to `${minikube ip}`)

```bash
echo "$(minikube ip) deploy.internal" | sudo tee -a /etc/hosts
cd ./k8s/minikube
./deploy.sh
```

You may need to use `ngrok` if testing with a local minikube cluster in order to provide a public endpoint for your github webhook:

```bash
ngrok http -host-header=rewrite deploy.internal:80
```

This will give you an endpoint like `https://fb2e74de.ngrok.io`, so you can configure your webhook on the cluster repo to point to something like `https://fb2e74de.ngrok.io/webhooks`. The webhook should be triggered on Pull Request events.

When the webhook is configured, you should be able to use `deploy request` as shown below to trigger the whole process. Then you should be able to get the logs from the relevant pod:

```
> kubectl --namespace deploy-robot logs deploy-robot-6bb9fb46d6-ltljc
2018/03/20 18:52:33 INFO: Listening on addr: :3016 path: /webhooks
2018/03/20 18:55:09 INFO: Webhook received
2018/03/20 18:55:09 INFO: Parsing Payload...
2018/03/20 18:55:09 INFO: Checking secret
2018/03/20 18:55:09
PR #8, SHA 90a864e84cde99283cf9e2c4cc7cea93ee36308c
2018/03/20 18:55:11 Walking guestbook
deployment "redis" created
service "redis" created
deployment "guestbook-ui" created
service "guestbook-ui" created
```

## To make a deployment request:

`deploy request` runs in the CD pipeline, but you can test from the root directory of this repo. Modify the config in `/example/guestbook` and then:

```
> deploy request --namespace=guestbook --manifestDir=example/guestbook --sha=41e8650 --org=redbadger --repo=cluster-local
2018/03/17 13:50:22 copying from example/guestbook to /guestbook
2018/03/17 13:50:22 commit obj: commit cfb3da3c0b28f4bb731a13689ed0f994ba24b340
Author: Robot <robot>
Date:   Sat Mar 17 13:50:22 2018 +0000

    Commit message!
2018/03/17 13:50:26 Pull request #1 raised!
```
