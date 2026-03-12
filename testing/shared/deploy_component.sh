#!/bin/bash

set -euo pipefail

# Script to build and deploy a Kubeflow dashboard component
# Usage: ./deploy_component.sh COMPONENT_PATH IMAGE_NAME TAG [MANIFESTS_PATH] [OVERLAY]

COMPONENT_PATH="$1"
IMAGE_NAME="$2"
TAG="$3"
MANIFESTS_PATH="${4:-manifests/kustomize}"
OVERLAY="${5:-overlays/kubeflow}"

cd "${COMPONENT_PATH}"
if [ -f "Makefile" ]; then
    make docker-build-multi-arch IMG="${IMAGE_NAME}" TAG="${TAG}"
else
    exit 1
fi

kind load docker-image "${IMAGE_NAME}:${TAG}"

if [ -d "${MANIFESTS_PATH}" ]; then
    cd "${MANIFESTS_PATH}"
else
    exit 1
fi

CURRENT_IMAGE="${IMAGE_NAME}"
PR_IMAGE="${IMAGE_NAME}:${TAG}"
CURRENT_IMAGE_ESCAPED=$(echo "$CURRENT_IMAGE" | sed 's|\.|\\.|g')
PR_IMAGE_ESCAPED=$(echo "$PR_IMAGE" | sed 's|\.|\\.|g')

kustomize build "$OVERLAY" \
  | sed "s|${CURRENT_IMAGE_ESCAPED}:[a-zA-Z0-9_.-]*|${PR_IMAGE_ESCAPED}|g" \
  | kubectl apply -f -
