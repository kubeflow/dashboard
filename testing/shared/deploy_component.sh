#!/bin/bash

set -euo pipefail

# Script to build and deploy a Kubeflow dashboard component
# Usage: ./deploy_component.sh COMPONENT_PATH IMAGE_NAME TAG [MANIFESTS_PATH] [OVERLAY]

COMPONENT_PATH="$1"
IMAGE_NAME="$2"
TAG="$3"
MANIFESTS_PATH="${4:-manifests}"
OVERLAY="${5:-overlays/kubeflow}"

cd "${COMPONENT_PATH}"
if [ -f "Makefile" ]; then
    if grep -q "docker-build-multi-arch" Makefile; then
        make docker-build-multi-arch IMG="${IMAGE_NAME}" TAG="${TAG}"
    else
        make docker-build IMG="${IMAGE_NAME}" TAG="${TAG}"
    fi
else
    exit 1
fi

kind load docker-image "${IMAGE_NAME}:${TAG}"

if [ -d "${MANIFESTS_PATH}" ]; then
    cd "${MANIFESTS_PATH}"
elif [ -d "config" ]; then
    cd "config"
    OVERLAY="overlays/kubeflow"
else
    exit 1
fi

export CURRENT_IMAGE="${IMAGE_NAME}"
export PR_IMAGE="${IMAGE_NAME}:${TAG}"
export CURRENT_IMAGE_ESCAPED=$(echo "$CURRENT_IMAGE" | sed 's|\.|\\.|g')
export PR_IMAGE_ESCAPED=$(echo "$PR_IMAGE" | sed 's|\.|\\.|g')


kustomize build "$OVERLAY" \
  | sed "s|${CURRENT_IMAGE_ESCAPED}:[a-zA-Z0-9_.-]*|${PR_IMAGE_ESCAPED}|g" \
  | sed "s|\$(CD_NAMESPACE)|${CD_NAMESPACE:-kubeflow}|g" \
  | sed "s|\$(CD_CLUSTER_DOMAIN)|${CD_CLUSTER_DOMAIN:-cluster.local}|g" \
  | sed "s|CD_NAMESPACE_PLACEHOLDER|${CD_NAMESPACE_PLACEHOLDER:-kubeflow}|g" \
  | sed "s|CD_CLUSTER_DOMAIN_PLACEHOLDER|${CD_CLUSTER_DOMAIN_PLACEHOLDER:-cluster.local}|g" \
  | kubectl apply -f -
