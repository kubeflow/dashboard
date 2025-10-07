# Notebook Controller Integration Tests

This directory contains integration tests for the Kubeflow Notebook Controller component.

## Prerequisites

- Docker
- kubectl
- kind
- kustomize
- istioctl
- Go (for unit tests)

## Setup

From the repository root:

```bash
./testing/shared/setup_env.sh
```

## Install

From the repository root:

```bash
./testing/shared/install_profile_controller.sh
./testing/shared/install_notebook_controller.sh
```

## Create User Profile

```bash
kubectl apply -f components/profile-controller/integration/resources/user-profile.yaml
until kubectl get ns kubeflow-user >/dev/null 2>&1; do sleep 1; done
```

## Run Integration Tests

```bash
cd components/notebook-controller/integration
# create a sample notebook and wait until Ready
kubectl apply -f resources/test-notebook.yaml
kubectl wait notebooks -n kubeflow-user -l app=test-notebook --for=condition=Ready --timeout=300s
```

## Cleanup (optional)

```bash
# delete test namespace created by profile
kubectl delete namespace kubeflow-user --ignore-not-found=true
# delete KinD cluster
kind delete cluster
```

## Controller Unit Tests (optional)

```bash
cd components/notebook-controller
make test
```

## Optional: Web App Integrations

```bash
kustomize build https://github.com/kubeflow/kubeflow//components/crud-web-apps/jupyter/manifests/overlays/istio?ref=master | kubectl apply -f -
kubectl wait pods -n kubeflow -l app=jupyter-web-app --for=condition=Ready --timeout=300s
kubectl port-forward -n kubeflow svc/jupyter-web-app-service 8085:80 &
curl -H "kubeflow-userid: user" localhost:8085

kustomize build https://github.com/kubeflow/kubeflow//components/crud-web-apps/volumes/manifests/overlays/istio?ref=master | kubectl apply -f -
kubectl wait pods -n kubeflow -l app=volumes-web-app --for=condition=Ready --timeout=300s
kubectl port-forward -n kubeflow svc/volumes-web-app-service 8087:80 &
curl -H "kubeflow-userid: user" localhost:8087
```
