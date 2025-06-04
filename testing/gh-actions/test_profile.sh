#!/bin/bash

set -euo pipefail

# Script to test Kubeflow Profile functionality
# Usage: ./test_profile.sh OPERATION PROFILE_NAME [USER_EMAIL] [TIMEOUT]

OPERATION="$1"
PROFILE_NAME="$2"
USER_EMAIL="${3:-${PROFILE_NAME}@example.com}"
TIMEOUT="${4:-300}"

case "$OPERATION" in
    "create")
        cat <<EOF | kubectl apply -f -
apiVersion: kubeflow.org/v1
kind: Profile
metadata:
  name: ${PROFILE_NAME}
spec:
  owner:
    kind: User
    name: ${USER_EMAIL}
  resourceQuotaSpec:
    hard:
      cpu: "2"
      memory: 2Gi
      requests.nvidia.com/gpu: "1"
EOF
        kubectl wait --for=condition=Ready profile "${PROFILE_NAME}" --timeout="${TIMEOUT}s"
        ;;

    "create-simple")
        cat <<EOF | kubectl apply -f -
apiVersion: kubeflow.org/v1
kind: Profile
metadata:
  name: ${PROFILE_NAME}
spec:
  owner:
    kind: User
    name: ${USER_EMAIL}
EOF
        kubectl wait --for=condition=Ready profile "${PROFILE_NAME}" --timeout="${TIMEOUT}s"
        ;;

    "validate")
        kubectl get namespace "${PROFILE_NAME}"
        kubectl get serviceaccount default-editor -n "${PROFILE_NAME}"
        kubectl get serviceaccount default-viewer -n "${PROFILE_NAME}"
        kubectl get rolebinding -n "${PROFILE_NAME}"
        kubectl get resourcequota -n "${PROFILE_NAME}"
        kubectl get profile "${PROFILE_NAME}" -o yaml
        ;;

    "update")
        kubectl patch profile "${PROFILE_NAME}" --type='merge' -p='{"spec":{"resourceQuotaSpec":{"hard":{"cpu":"3","memory":"3Gi"}}}}'
        sleep 10
        kubectl get resourcequota -n "${PROFILE_NAME}" -o yaml | grep -A5 "hard:" 
        ;;

    "delete")
        kubectl delete profile "${PROFILE_NAME}" 
        kubectl wait --for=delete namespace "${PROFILE_NAME}" --timeout="${TIMEOUT}s" 
        ;;

    "list")
        kubectl get profiles
        ;;

    *)
        echo "Invalid operation: ${OPERATION}"
        echo "Valid operations: create, create-simple, validate, update, delete, list"
        exit 1
        ;;
esac 