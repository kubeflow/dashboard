#!/usr/bin/env bash
set -euxo pipefail

# Deploy notebook-controller from upstream manifests (recommended for integration testing)
kustomize build https://github.com/kubeflow/kubeflow//components/notebook-controller/config/overlays/kubeflow?ref=master | kubectl apply -f -

kubectl wait pods -n kubeflow -l app=notebook-controller --for=condition=Ready --timeout=300s
