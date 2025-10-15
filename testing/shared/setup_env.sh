#!/usr/bin/env bash
set -euxo pipefail

KIND_VERSION="${KIND_VERSION:-0.29.0}"
KUSTOMIZE_VERSION="${KUSTOMIZE_VERSION:-5.4.1}"
ISTIO_VERSION="${ISTIO_VERSION:-1.17.8}"
CERT_MANAGER_VERSION="${CERT_MANAGER_VERSION:-v1.8.0}"

command -v kustomize >/dev/null 2>&1 || {
  curl -sL -o kustomize.tar.gz "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v${KUSTOMIZE_VERSION}/kustomize_v${KUSTOMIZE_VERSION}_linux_amd64.tar.gz"
  tar -xzf kustomize.tar.gz
  chmod +x kustomize
  sudo mv kustomize /usr/local/bin
}

command -v kind >/dev/null 2>&1 || {
  curl -sL -o kind "https://kind.sigs.k8s.io/dl/v${KIND_VERSION}/kind-linux-amd64"
  chmod +x kind
  sudo mv kind /usr/local/bin
}

kind create cluster --config testing/gh-actions/kind-1-33.yaml || true
kubectl create namespace kubeflow || true

kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml
kubectl wait --for=condition=Ready pods -n cert-manager -l app=cert-manager --timeout=300s

command -v istioctl >/dev/null 2>&1 || {
  tmpdir="$(mktemp -d)"
  (cd "$tmpdir" && curl -sL https://istio.io/downloadIstio | ISTIO_VERSION=${ISTIO_VERSION} sh - && sudo mv "istio-${ISTIO_VERSION}/bin/istioctl" /usr/local/bin/istioctl)
  rm -rf "$tmpdir"
}

istioctl install -y

kustomize build https://github.com/kubeflow/manifests//common/kubeflow-roles/base?ref=master | kubectl apply -f -
