#!/usr/bin/env bash
set -euxo pipefail

IMG="${IMG:-centraldashboard-angular}"
TAG="${TAG:-integration-test}"

./testing/shared/deploy_component.sh components/centraldashboard-angular "${IMG}" "${TAG}" manifests overlays/istio

kubectl wait --for=condition=Ready pods -n kubeflow -l app=centraldashboard-angular --timeout=300s
kubectl wait --for=condition=Available deployment -n kubeflow centraldashboard-angular --timeout=300s
