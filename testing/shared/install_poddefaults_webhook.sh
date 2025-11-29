#!/usr/bin/env bash
set -euxo pipefail

IMG="${IMG:-ghcr.io/kubeflow/dashboard/poddefaults-webhook}"
TAG="${TAG:-integration-test}"

./testing/shared/deploy_component.sh components/poddefaults-webhooks "${IMG}" "${TAG}" manifests overlays/cert-manager

kubectl wait --for=condition=Ready pods -n kubeflow -l app=poddefaults --timeout=300s
kubectl wait --for=condition=Available deployment -n kubeflow poddefaults-webhook-deployment --timeout=300s
