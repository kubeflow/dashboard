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
        cat <<EOF | kubectl apply -f -
apiVersion: kubeflow.org/v1alpha1
kind: PodDefault
metadata:
  name: ${PODDEFAULT_NAME}
  namespace: ${NAMESPACE}
spec:
  selector:
    matchLabels:
      ${PODDEFAULT_NAME}: "true"
  desc: "Test PodDefault for integration testing"
  env:
  - name: TEST_ENV_VAR
    value: "test-value"
  volumes:
  - name: test-volume
    emptyDir: {}
  volumeMounts:
  - name: test-volume
    mountPath: /test-mount
EOF
        ;;

    "create-multi-poddefault")
        cat <<EOF | kubectl apply -f -
apiVersion: kubeflow.org/v1alpha1
kind: PodDefault
metadata:
  name: ${PODDEFAULT_NAME}-2
  namespace: ${NAMESPACE}
spec:
  selector:
    matchLabels:
      test-multi: "true"
  desc: "Second test PodDefault"
  env:
  - name: SECOND_ENV_VAR
    value: "second-value"
EOF
        ;;

    "test-mutation")
        cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: ${TEST_POD_NAME}
  namespace: ${NAMESPACE}
  labels:
    ${PODDEFAULT_NAME}: "true"
spec:
  containers:
  - name: test-container
    image: busybox:latest
    command: ["sleep", "3600"]
EOF
                
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
        cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: ${TEST_POD_NAME}-multi
  namespace: ${NAMESPACE}
  labels:
    ${PODDEFAULT_NAME}: "true"
    test-multi: "true"
spec:
  containers:
  - name: test-container
    image: busybox:latest
    command: ["sleep", "300"]
EOF

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
        cat <<EOF | kubectl apply -f - 
apiVersion: kubeflow.org/v1alpha1
kind: PodDefault
metadata:
  name: invalid-poddefault
  namespace: ${NAMESPACE}
spec:
  selector:
    matchLabels:
      invalid: "true"
  volumeMounts:
  - name: non-existent-volume
    mountPath: /invalid
EOF
        ;;

    "validate-webhook")
        kubectl get crd poddefaults.kubeflow.org
        kubectl describe crd poddefaults.kubeflow.org
        kubectl get mutatingwebhookconfiguration admission-webhook-mutating-webhook-configuration
        kubectl describe mutatingwebhookconfiguration admission-webhook-mutating-webhook-configuration
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