#!/usr/bin/env bash

set -euxo pipefail

# Deploy notebook-controller from upstream manifests (recommended for integration testing)
#
# TODO: replace with the `kubeflow/notebooks` repo, and new version when available.
#       remember to check for other tests that reference `kubeflow/kubeflow` when updating.
#
kubectl apply -k "https://github.com/kubeflow/kubeflow/components/notebook-controller/config/overlays/kubeflow?ref=v1.10.0"

kubectl wait pods -n kubeflow -l app=notebook-controller --for=condition=Ready --timeout=300s
