# Central Dashboard Angular Integration Tests

This directory contains integration tests for the Kubeflow Central Dashboard Angular component.

## Prerequisites

- Docker
- kubectl
- kind
- kustomize
- istioctl
- Node.js 18+ and npm (for UI tests)
- Xvfb (for headless UI tests)

## Setup

From the repository root:

```bash
./testing/shared/setup_env.sh
```

## Install

From the repository root:

```bash
./testing/shared/install_profile_controller.sh
./testing/shared/install_centraldashboard_angular.sh
```

## Backend Integration Tests

```bash
./testing/shared/test_service.sh validate-service dashboard-angular kubeflow
./testing/shared/test_service.sh port-forward dashboard-angular kubeflow 8080 80
./testing/shared/test_service.sh test-health dashboard-angular kubeflow 8080
./testing/shared/test_service.sh performance-test dashboard-angular kubeflow 8080 80 8
./testing/shared/test_service.sh test-metrics dashboard-angular kubeflow 8080
./testing/shared/test_service.sh check-errors dashboard-angular kubeflow
```

# Apply necessary CRs

```bash
kubectl apply -k "https://github.com/kubeflow/manifests/common/kubeflow-roles/base?ref=v1.11.0"
kubectl apply -f components/profile-controller/integration/resources/user-profile.yaml
while ! kubectl get ns kubeflow-user; do sleep 1; done
kubectl apply -f components/centraldashboard-angular/integration/resources/test-notebook.yaml
kubectl wait notebooks -n kubeflow-user -l app=test-notebook --for=condition=Ready --timeout=300s
```

## Frontend UI Tests (optional)

From the repository root:

```bash
# install dependencies
cd components/centraldashboard-angular/frontend
npm install
cd .. && make build-common-lib
cd kubeflow-common-lib/dist/kubeflow && sudo npm link

# link kubeflow package in frontend
cd ../../../frontend
npm link kubeflow
# run UI tests headlessly
npm run serve &
npx wait-on http://localhost:4200
DISPLAY=:99 xvfb-run -a npm run ui-test-ci-all

kill %1 #kill the background serve process
```

## Cleanup (optional)

```bash
# stop port-forward if running
./testing/shared/test_service.sh stop-port-forward dashboard-angular kubeflow 8080
# delete KinD cluster
kind delete cluster
```
