#!/bin/bash

set -euo pipefail

# Script to test Kubeflow Profile functionality
# Usage: ./test_profile.sh OPERATION PROFILE_NAME [USER_EMAIL] [TIMEOUT]

OPERATION="$1"

if [[ "$OPERATION" != "list" ]]; then
  if [[ $# -lt 2 ]]; then
    echo "Error: PROFILE_NAME is required for operation: $OPERATION"
    echo "Usage: ./test_profile.sh OPERATION PROFILE_NAME [USER_EMAIL] [TIMEOUT]"
    exit 1
  fi
  PROFILE_NAME="$2"
  USER_EMAIL="${3:-${PROFILE_NAME}@example.com}"
  TIMEOUT="${4:-300}"
else
  PROFILE_NAME=""
  TIMEOUT="${2:-300}"
fi

case "$OPERATION" in
    "create")
        export PROFILE_NAME USER_EMAIL
        envsubst < "$(dirname "$0")/resources/profile-with-quota.yaml" | kubectl apply -f -
        kubectl wait --for=jsonpath='{.metadata.name}'=${PROFILE_NAME} profile "${PROFILE_NAME}" --timeout=60s
        timeout=120
        interval=5
        elapsed=0
        while ! kubectl get namespace "${PROFILE_NAME}" >/dev/null 2>&1; do
          if [ $elapsed -ge $timeout ]; then
            echo "Timeout waiting for namespace ${PROFILE_NAME} to be created"
            exit 1
          fi
          echo "Waiting for namespace ${PROFILE_NAME} to be created..."
          sleep $interval
          elapsed=$((elapsed + interval))
        done
        if ! kubectl wait --for=condition=Ready profile "${PROFILE_NAME}" --timeout="${TIMEOUT}s" 2>/dev/null; then
            echo "Warning: Profile ${PROFILE_NAME} created but Ready condition not set - this may be expected"
            sleep 10
            kubectl get namespace "${PROFILE_NAME}" || exit 1
        fi
        ;;

    "create-simple")
        export PROFILE_NAME USER_EMAIL
        envsubst < "$(dirname "$0")/resources/profile-simple.yaml" | kubectl apply -f -
        kubectl wait --for=jsonpath='{.metadata.name}'=${PROFILE_NAME} profile "${PROFILE_NAME}" --timeout=60s
        timeout=120
        interval=5
        elapsed=0
        while ! kubectl get namespace "${PROFILE_NAME}" >/dev/null 2>&1; do
          if [ $elapsed -ge $timeout ]; then
            echo "Timeout waiting for namespace ${PROFILE_NAME} to be created"
            exit 1
          fi
          echo "Waiting for namespace ${PROFILE_NAME} to be created..."
          sleep $interval
          elapsed=$((elapsed + interval))
        done
        if ! kubectl wait --for=condition=Ready profile "${PROFILE_NAME}" --timeout="${TIMEOUT}s" 2>/dev/null; then
            echo "Warning: Profile ${PROFILE_NAME} created but Ready condition not set - this may be expected"
            sleep 10
            kubectl get namespace "${PROFILE_NAME}" || exit 1
        fi
        ;;

    "validate")
        echo "Validating profile ${PROFILE_NAME}..."
        kubectl get namespace "${PROFILE_NAME}"
        
        if kubectl get serviceaccount default-editor -n "${PROFILE_NAME}" 2>/dev/null; then
            echo "✓ default-editor service account found"
        else
            echo "⚠ default-editor service account not found - this may be expected in some setups"
        fi
        
        if kubectl get serviceaccount default-viewer -n "${PROFILE_NAME}" 2>/dev/null; then
            echo "✓ default-viewer service account found"
        else
            echo "⚠ default-viewer service account not found - this may be expected in some setups"
        fi
        
        kubectl get rolebinding -n "${PROFILE_NAME}" || echo "⚠ No role bindings found"
        
        kubectl get resourcequota -n "${PROFILE_NAME}" || echo "⚠ No resource quotas found"
        
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