#!/usr/bin/env bash

set -euxo pipefail

KIND_VERSION="${KIND_VERSION:-0.29.0}"
KUSTOMIZE_VERSION="${KUSTOMIZE_VERSION:-5.5.0}"

# fail if not linux
if [[ "$(uname -s)" != "Linux" ]]; then
  echo "ERROR: this script is only supported on Linux."
  exit 1
fi

# fail if not amd64
if [[ "$(uname -m)" != "x86_64" ]]; then
  echo "ERROR: this script is only supported on amd64 architecture."
  exit 1
fi

# install kustomize binary
command -v kustomize >/dev/null 2>&1 || {
  curl -sL -o kustomize.tar.gz "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v${KUSTOMIZE_VERSION}/kustomize_v${KUSTOMIZE_VERSION}_linux_amd64.tar.gz"
  tar -xzf kustomize.tar.gz
  chmod +x kustomize
  sudo mv kustomize /usr/local/bin
}

# install kind binary
command -v kind >/dev/null 2>&1 || {
  curl -sL -o kind "https://kind.sigs.k8s.io/dl/v${KIND_VERSION}/kind-linux-amd64"
  chmod +x kind
  sudo mv kind /usr/local/bin
}

# create kind cluster
kind create cluster --wait 5m --config testing/gh-actions/kind-1-33.yaml || true

# install cert manager
./testing/gh-actions/install_cert_manager.sh

# install istio
./testing/gh-actions/install_istio.sh

# create kubeflow namespace
kubectl create namespace kubeflow || true

# install kubeflow roles
kubectl apply -k "https://github.com/kubeflow/manifests/common/kubeflow-roles/base?ref=v1.11.0"
