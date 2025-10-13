#!/usr/bin/env bash
set -euxo pipefail

IMG="${IMG:-ghcr.io/kubeflow/dashboard/dashboard-angular}"
TAG="${TAG:-integration-test}"

./testing/shared/deploy_component.sh components/centraldashboard-angular "${IMG}" "${TAG}" manifests overlays/istio

kubectl wait --for=condition=Ready pods -n kubeflow -l app=dashboard-angular --timeout=300s
kubectl wait --for=condition=Available deployment -n kubeflow dashboard-angular --timeout=300s
