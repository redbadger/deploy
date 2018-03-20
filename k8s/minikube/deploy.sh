#!/usr/bin/env bash

: "${DEPLOY_SECRET?Need to export DEPLOY_SECRET}"
: "${PERSONAL_ACCESS_TOKEN?Need to export PERSONAL_ACCESS_TOKEN}"

kubectl apply -f ./00-namespace.yaml

kubectl --namespace=deploy-robot create secret generic deploy-secret \
	--from-literal=webhookSecret="${DEPLOY_SECRET}" \
	--from-literal=personalAccessToken="${PERSONAL_ACCESS_TOKEN}"

kubectl apply -f .
