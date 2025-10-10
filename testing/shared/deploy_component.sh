#!/bin/bash

set -euxo pipefail

# Script to build and deploy a Kubeflow dashboard component
# Usage: ./deploy_component.sh COMPONENT_NAME COMPONENT_PATH IMAGE_NAME TAG [MANIFESTS_PATH] [OVERLAY]

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

kind load docker-image "${IMAGE_NAME}:${TAG}" --name dashboard

if [ -d "${MANIFESTS_PATH}" ]; then
    cd "${MANIFESTS_PATH}"
elif [ -d "config" ]; then
    cd "config"
    OVERLAY="overlays/kubeflow"
else
    exit 1
fi

export PR_IMAGE="${IMAGE_NAME}:${TAG}"
export PR_IMAGE_ESCAPED=$(echo "$PR_IMAGE" | sed 's|\.|\\.|g')

for overlay_path in "${OVERLAY}" "overlays/kserve" "overlays/cert-manager"; do
    if [ -d "$overlay_path" ]; then
        KUSTOMIZE_OUTPUT=$(kustomize build "$overlay_path")
        IMAGE_BASENAME=$(basename "${IMAGE_NAME}")
        ACTUAL_IMAGE=$(echo "$KUSTOMIZE_OUTPUT" | grep -E "^[[:space:]]*image:[[:space:]]+\S" | grep -F "${IMAGE_BASENAME}" | head -1 | sed -E 's/^[[:space:]]*image:[[:space:]]*//' | sed -E 's/:[^:]*$//')
        if [ -z "$ACTUAL_IMAGE" ]; then
            echo "Could not detect image in manifests, using provided IMAGE_NAME"
            ACTUAL_IMAGE="${IMAGE_NAME}"
        fi
        echo "Detected image in manifests: ${ACTUAL_IMAGE}"
        echo "Will replace with: ${PR_IMAGE}"
        ACTUAL_IMAGE_ESCAPED=$(echo "$ACTUAL_IMAGE" | sed 's|\.|\\.|g')
        
        if [ "$overlay_path" = "overlays/cert-manager" ]; then
            echo "$KUSTOMIZE_OUTPUT" \
              | sed "s|${ACTUAL_IMAGE_ESCAPED}:[a-zA-Z0-9_.-]*|${PR_IMAGE_ESCAPED}|g" \
              | sed 's/$(podDefaultsServiceName)/poddefaults-webhook-service/g' \
              | sed 's/$(podDefaultsNamespace)/kubeflow/g' \
              | sed "s|\$(CD_NAMESPACE)|${CD_NAMESPACE:-kubeflow}|g" \
              | sed "s|\$(CD_CLUSTER_DOMAIN)|${CD_CLUSTER_DOMAIN:-cluster.local}|g" \
              | sed "s|CD_NAMESPACE_PLACEHOLDER|${CD_NAMESPACE_PLACEHOLDER:-kubeflow}|g" \
              | sed "s|CD_CLUSTER_DOMAIN_PLACEHOLDER|${CD_CLUSTER_DOMAIN_PLACEHOLDER:-cluster.local}|g" \
              | kubectl apply -f -
        else
            echo "$KUSTOMIZE_OUTPUT" \
              | sed "s|${ACTUAL_IMAGE_ESCAPED}:[a-zA-Z0-9_.-]*|${PR_IMAGE_ESCAPED}|g" \
              | sed "s|\$(CD_NAMESPACE)|${CD_NAMESPACE:-kubeflow}|g" \
              | sed "s|\$(CD_CLUSTER_DOMAIN)|${CD_CLUSTER_DOMAIN:-cluster.local}|g" \
              | sed "s|CD_NAMESPACE_PLACEHOLDER|${CD_NAMESPACE_PLACEHOLDER:-kubeflow}|g" \
              | sed "s|CD_CLUSTER_DOMAIN_PLACEHOLDER|${CD_CLUSTER_DOMAIN_PLACEHOLDER:-cluster.local}|g" \
              | kubectl apply -f -
        fi
        exit 0
    fi
done

exit 1 