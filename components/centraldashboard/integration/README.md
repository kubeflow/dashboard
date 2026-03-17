# Central Dashboard Integration Tests

This directory contains integration tests for the Kubeflow Central Dashboard component.

## Prerequisites

- Docker
- kubectl
- kind
- kustomize
- istioctl

## Setup

From the repository root:

```bash
./testing/shared/setup_env.sh
```

## Install

From the repository root:

```bash
./testing/shared/install_profile_controller.sh
./testing/shared/install_centraldashboard.sh
```

## Create Test Profile

```bash
kubectl apply -f components/profile-controller/integration/resources/profile-dashboard-test.yaml
```

## Run Integration Tests

```bash
./testing/shared/test_service.sh validate-service dashboard kubeflow
./testing/shared/test_service.sh port-forward dashboard kubeflow 8082 80
./testing/shared/test_service.sh test-health dashboard kubeflow 8082
./testing/shared/test_service.sh performance-test dashboard kubeflow 8082 80 10
./testing/shared/test_service.sh test-metrics dashboard kubeflow 8082
./testing/shared/test_service.sh check-errors dashboard kubeflow
```

## Cleanup (optional)

```bash
# stop port-forward if running
./testing/shared/test_service.sh stop-port-forward dashboard kubeflow 8082
# delete KinD cluster
kind delete cluster
```
