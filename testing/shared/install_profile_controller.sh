#!/usr/bin/env bash
set -euxo pipefail

TAG="integration-test"
PROFILE_IMG="ghcr.io/kubeflow/dashboard/profile-controller"
KFAM_IMG="ghcr.io/kubeflow/dashboard/access-management"

make -C components/profile-controller docker-build-multi-arch IMG="${PROFILE_IMG}" TAG="${TAG}"
make -C components/access-management docker-build-multi-arch IMG="${KFAM_IMG}" TAG="${TAG}"

kind load docker-image "${PROFILE_IMG}:${TAG}"
kind load docker-image "${KFAM_IMG}:${TAG}"

NEW_PROFILE_IMAGE="${PROFILE_IMG}:${TAG}"
NEW_KFAM_IMAGE="${KFAM_IMG}:${TAG}"

# Escape "." in the image names, as it is a special character in sed
CURRENT_PROFILE_IMAGE_ESCAPED=$(echo "$PROFILE_IMG" | sed 's|\.|\\.|g')
NEW_PROFILE_IMAGE_ESCAPED=$(echo "$NEW_PROFILE_IMAGE" | sed 's|\.|\\.|g')
CURRENT_KFAM_IMAGE_ESCAPED=$(echo "$KFAM_IMG" | sed 's|\.|\\.|g')
NEW_KFAM_IMAGE_ESCAPED=$(echo "$NEW_KFAM_IMAGE" | sed 's|\.|\\.|g')

echo "Deploying Profile Controller and KFAM to kubeflow namespace"
kustomize build components/profile-controller/config/overlays/kubeflow \
    | sed "s|${CURRENT_PROFILE_IMAGE_ESCAPED}:[a-zA-Z0-9_.-]*|${NEW_PROFILE_IMAGE_ESCAPED}|g" \
    | sed "s|${CURRENT_KFAM_IMAGE_ESCAPED}:[a-zA-Z0-9_.-]*|${NEW_KFAM_IMAGE_ESCAPED}|g" \
    | kubectl apply -f -

kubectl wait --for=condition=Available deployment -n kubeflow profiles-deployment --timeout=300s
kubectl wait pods -n kubeflow -l kustomize.component=profiles --for=condition=Ready --timeout=300s
