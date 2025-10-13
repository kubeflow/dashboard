#!/usr/bin/env bash
set -euxo pipefail

IMG="${IMG:-ghcr.io/kubeflow/dashboard/dashboard}"
TAG="${TAG:-integration-test}"

./testing/shared/deploy_component.sh components/centraldashboard "${IMG}" "${TAG}" manifests overlays/istio

kubectl wait --for=condition=Ready pods -n kubeflow -l app=dashboard --timeout=300s
kubectl wait --for=condition=Available deployment -n kubeflow dashboard --timeout=300s
