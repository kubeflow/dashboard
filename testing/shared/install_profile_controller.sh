#!/usr/bin/env bash
set -euxo pipefail

IMG="${IMG:-profile-controller}"
TAG="${TAG:-integration-test}"

./testing/shared/deploy_component.sh components/profile-controller "${IMG}" "${TAG}"

kubectl wait --for=condition=Ready pods -n kubeflow -l kustomize.component=profiles --timeout=300s
kubectl wait --for=condition=Available deployment -n kubeflow profiles-deployment --timeout=300s
