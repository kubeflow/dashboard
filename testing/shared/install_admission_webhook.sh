#!/usr/bin/env bash
set -euo pipefail

IMG="${IMG:-admission-webhook}"
TAG="${TAG:-integration-test}"

./testing/shared/deploy_component.sh components/admission-webhook "${IMG}" "${TAG}" manifests overlays/cert-manager

kubectl wait --for=condition=Ready pods -n kubeflow -l app=poddefaults --timeout=300s
kubectl wait --for=condition=Available deployment -n kubeflow admission-webhook-deployment --timeout=300s
