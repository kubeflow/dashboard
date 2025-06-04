#!/bin/bash

set -euo pipefail

# Script to build and deploy a Kubeflow dashboard component
# Usage: ./deploy_component.sh COMPONENT_NAME COMPONENT_PATH IMAGE_NAME TAG [MANIFESTS_PATH] [OVERLAY]

COMPONENT_NAME="$1"
COMPONENT_PATH="$2"
IMAGE_NAME="$3"
TAG="$4"
MANIFESTS_PATH="${5:-manifests}"
OVERLAY="${6:-overlays/kubeflow}"

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

for overlay_path in "${OVERLAY}" "overlays/kserve" "overlays/cert-manager"; do
    if [ -d "$overlay_path" ]; then
        kustomize build "$overlay_path" \
          | sed "s|${CURRENT_IMAGE_ESCAPED}:[a-zA-Z0-9_.-]*|${PR_IMAGE_ESCAPED}|g" \
          | kubectl apply -f -
        exit 0
    fi
done

exit 1 