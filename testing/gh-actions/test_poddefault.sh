#!/bin/bash

set -euo pipefail

# Script to test PodDefault functionality
# Usage: ./test_poddefault.sh OPERATION NAMESPACE [PODDEFAULT_NAME] [TEST_POD_NAME]

OPERATION="$1"
NAMESPACE="$2"
PODDEFAULT_NAME="${3:-test-poddefault}"
TEST_POD_NAME="${4:-test-pod}"

case "$OPERATION" in
    "create-namespace")
        kubectl create namespace "${NAMESPACE}" 
        kubectl label namespace "${NAMESPACE}" katib.kubeflow.org/metrics-collector-injection=enabled 
        kubectl label namespace "${NAMESPACE}" app.kubernetes.io/part-of=kubeflow-profile
        ;;

    "create-poddefault")
        export PODDEFAULT_NAME NAMESPACE
        envsubst < "$(dirname "$0")/resources/poddefault-basic.yaml" | kubectl apply -f -
        ;;

    "create-multi-poddefault")
        export PODDEFAULT_NAME NAMESPACE
        envsubst < "$(dirname "$0")/resources/poddefault-multi-selector.yaml" | kubectl apply -f -
        ;;

    "test-mutation")
        export TEST_POD_NAME NAMESPACE PODDEFAULT_NAME
        envsubst < "$(dirname "$0")/resources/poddefault-test-pod.yaml" | kubectl apply -f -
                
        kubectl get pod "${TEST_POD_NAME}" -n "${NAMESPACE}" -o yaml | grep -q "TEST_ENV_VAR" || {
            echo "ERROR: TEST_ENV_VAR not found in pod spec"
            kubectl get pod "${TEST_POD_NAME}" -n "${NAMESPACE}" -o yaml
            exit 1
        }
        kubectl get pod "${TEST_POD_NAME}" -n "${NAMESPACE}" -o yaml | grep -q "test-volume" || {
            echo "ERROR: test-volume not found in pod spec"
            kubectl get pod "${TEST_POD_NAME}" -n "${NAMESPACE}" -o yaml
            exit 1
        }
        
        ;;

    "test-multi-mutation")
        export TEST_POD_NAME NAMESPACE PODDEFAULT_NAME
        envsubst < "$(dirname "$0")/resources/poddefault-test-pod-multi.yaml" | kubectl apply -f -

        kubectl get pod "${TEST_POD_NAME}-multi" -n "${NAMESPACE}" -o yaml | grep -q "TEST_ENV_VAR" || {
            echo "ERROR: TEST_ENV_VAR not found in pod spec"
            kubectl get pod "${TEST_POD_NAME}-multi" -n "${NAMESPACE}" -o yaml
            exit 1
        }
        kubectl get pod "${TEST_POD_NAME}-multi" -n "${NAMESPACE}" -o yaml | grep -q "SECOND_ENV_VAR" || {
            echo "ERROR: SECOND_ENV_VAR not found in pod spec"
            kubectl get pod "${TEST_POD_NAME}-multi" -n "${NAMESPACE}" -o yaml
            exit 1
        }
        
        ;;

    "test-error-handling")
        export NAMESPACE
        envsubst < "$(dirname "$0")/resources/poddefault-invalid.yaml" | kubectl apply -f -
        ;;

    "validate-webhook")
        kubectl get crd poddefaults.kubeflow.org
        kubectl describe crd poddefaults.kubeflow.org
        kubectl get mutatingwebhookconfiguration poddefaults-webhook-mutating-webhook-configuration
        kubectl describe mutatingwebhookconfiguration poddefaults-webhook-mutating-webhook-configuration
        ;;

    "cleanup")
        kubectl delete namespace "${NAMESPACE}" 
        ;;

    *)
        echo "Invalid operation: ${OPERATION}"
        echo "Valid operations: create-namespace, create-poddefault, create-multi-poddefault, test-mutation, test-multi-mutation, test-error-handling, validate-webhook, cleanup"
        exit 1
        ;;
esac 