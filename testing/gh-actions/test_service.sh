#!/bin/bash

set -euo pipefail

# Script to test Kubeflow dashboard services
# Usage: ./test_service.sh OPERATION SERVICE_NAME [NAMESPACE] [PORT] [TARGET_PORT]

OPERATION="$1"
SERVICE_NAME="$2"
NAMESPACE="${3:-kubeflow}"
PORT="${4:-8080}"
TARGET_PORT="${5:-80}"

case "$OPERATION" in
    "port-forward")
        kubectl port-forward -n "${NAMESPACE}" service/"${SERVICE_NAME}" "${PORT}:${TARGET_PORT}" &
        PF_PID=$!
        sleep 10
        echo "${PF_PID}" > /tmp/portforward_${SERVICE_NAME}_${PORT}.pid
        ;;

    "stop-port-forward")
        if [ -f "/tmp/portforward_${SERVICE_NAME}_${PORT}.pid" ]; then
            PF_PID=$(cat /tmp/portforward_${SERVICE_NAME}_${PORT}.pid)
            kill "${PF_PID}" 
            rm -f /tmp/portforward_${SERVICE_NAME}_${PORT}.pid
        fi
        ;;

    "test-health")
        curl -f "http://localhost:${PORT}/healthz" 2>/dev/null || curl -f "http://localhost:${PORT}/" 2>/dev/null
        ;;

    "test-dashboard")
        curl -f "http://localhost:${PORT}/" >/dev/null 2>&1
        curl -f "http://localhost:${PORT}/assets/dashboard.js" >/dev/null 2>&1 
        curl -f "http://localhost:${PORT}/api/v1/namespaces" >/dev/null 2>&1 
        curl -L "http://localhost:${PORT}/jupyter" >/dev/null 2>&1 
        curl -L "http://localhost:${PORT}/pipeline" >/dev/null 2>&1 
        curl -L "http://localhost:${PORT}/katib" >/dev/null 2>&1 
        ;;

    "test-kfam")
        curl -f "http://localhost:${PORT}/healthz" >/dev/null 2>&1
        curl "http://localhost:${PORT}/version" 2>/dev/null 
        ;;

    "test-api-with-user")
        USER_EMAIL="${6:-test-user@example.com}"
        PROFILE_NAMESPACE="${7:-test-profile}"
        curl -H "kubeflow-userid: ${USER_EMAIL}" \
             "http://localhost:${PORT}/kfam/v1/bindings?namespace=${PROFILE_NAMESPACE}" \
             2>/dev/null 
        ;;

    "performance-test")
        REQUESTS="${6:-10}"
        for i in $(seq 1 "${REQUESTS}"); do
            curl -s "http://localhost:${PORT}/" >/dev/null &
        done
        wait
        ;;

    "test-metrics")
        curl "http://localhost:${PORT}/metrics" 2>/dev/null 
        ;;

    "validate-service")
        kubectl get service "${SERVICE_NAME}" -n "${NAMESPACE}"
        kubectl describe service "${SERVICE_NAME}" -n "${NAMESPACE}"
        ;;

    "check-logs")
        LINES="${6:-50}"
        case "${SERVICE_NAME}" in
            "centraldashboard") kubectl logs -n "${NAMESPACE}" -l app=centraldashboard --tail="${LINES}" ;;
            "centraldashboard-angular") kubectl logs -n "${NAMESPACE}" -l app=centraldashboard-angular --tail="${LINES}" ;;
            "kfam") kubectl logs -n "${NAMESPACE}" -l app=kfam --tail="${LINES}" ;;
            "poddefaults-webhook") kubectl logs -n "${NAMESPACE}" -l app=poddefaults-webhook --tail="${LINES}" ;;
            *) kubectl logs -n "${NAMESPACE}" -l app="${SERVICE_NAME}" --tail="${LINES}" ;;
        esac
        ;;

    "check-errors")
        case "${SERVICE_NAME}" in
            "centraldashboard") kubectl logs -n "${NAMESPACE}" -l app=centraldashboard --tail=100 | grep -i error || echo "No errors found" ;;
            "centraldashboard-angular") kubectl logs -n "${NAMESPACE}" -l app=centraldashboard-angular --tail=100 | grep -i error || echo "No errors found" ;;
            "kfam") kubectl logs -n "${NAMESPACE}" -l app=kfam --tail=100 | grep -i error || echo "No errors found" ;;
            "poddefaults-webhook") kubectl logs -n "${NAMESPACE}" -l app=poddefaults-webhook --tail=100 | grep -i error || echo "No errors found" ;;
            *) kubectl logs -n "${NAMESPACE}" -l app="${SERVICE_NAME}" --tail=100 | grep -i error || echo "No errors found" ;;
        esac
        ;;

    *)
        echo "Invalid operation: ${OPERATION}"
        echo "Valid operations: port-forward, stop-port-forward, test-health, test-dashboard, test-kfam, test-api-with-user, performance-test, test-metrics, validate-service, check-logs, check-errors"
        exit 1
        ;;
esac 