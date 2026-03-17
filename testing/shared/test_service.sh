#!/bin/bash

set -euxo pipefail

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
        # test communication between dashboard and access-management
        curl -f \
            -H "kubeflow-userid: test-user" \
            "http://localhost:${PORT}/api/workgroup/exists" \
            >/dev/null 2>&1

        # test the NetworkPolicy, by ensuring other Pods timeout talking to the dashboard
        OUTPUT=$(kubectl run \
                    netshoot-test --rm -i \
                    --restart=Never \
                    --image nicolaka/netshoot \
                    -- curl dashboard.kubeflow.svc --connect-timeout 5 \
                    2>&1 || true)
        echo $OUTPUT | grep "Connection timed out after"
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
        kubectl get service      "${SERVICE_NAME}" -n "${NAMESPACE}"
        kubectl describe service "${SERVICE_NAME}" -n "${NAMESPACE}"
        ;;

    "check-errors")
        # get the app_label from the service selector
        APP_LABEL=$(kubectl get service "${SERVICE_NAME}" -n "${NAMESPACE}" -o jsonpath='{.spec.selector.app}')
        if [ -z "${APP_LABEL}" ]; then
          echo "ERROR: service ${SERVICE_NAME} in namespace ${NAMESPACE} does not have an 'app' label in its selector"
          exit 1
        fi

        # ensure there are pods with the correct label before trying to get logs
        NUM_PODS=$(kubectl get pods -n "${NAMESPACE}" -l app="${APP_LABEL}" -o name | wc -l)
        if [ "${NUM_PODS}" -eq 0 ]; then
          echo "ERROR: no pods with label app=${SERVICE_NAME} found in namespace ${NAMESPACE}"
          exit 1
        fi

        # read logs from default container of all pods with the correct label
        LOGS_RAW=$(kubectl logs --tail=100 --prefix -n "${NAMESPACE}" -l app="${APP_LABEL}")
        if [ -z "${LOGS_RAW}" ]; then
          echo "WARN: no logs found for service ${SERVICE_NAME} in namespace ${NAMESPACE}"
          exit 0
        fi

        # check for errors in logs (case-insensitive)
        LOGS_ERRORS=$(echo "${LOGS_RAW}" | grep -i error || true)
        if [ -n "${LOGS_ERRORS}" ]; then
          echo "ERROR: found errors in logs for service ${SERVICE_NAME} in namespace ${NAMESPACE}:"
          echo "${LOGS_ERRORS}"
          exit 1
        else
          echo "INFO: no errors found in logs for service ${SERVICE_NAME} in namespace ${NAMESPACE}"
        fi
        ;;

    *)
        echo "Invalid operation: ${OPERATION}"
        echo "Valid operations: port-forward, stop-port-forward, test-health, test-dashboard, performance-test, test-metrics, validate-service, check-errors"
        exit 1
        ;;
esac
