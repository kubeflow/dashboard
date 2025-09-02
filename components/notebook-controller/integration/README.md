# Notebook Controller Integration Tests

This directory contains integration tests for the Kubeflow Notebook Controller component.

## 1. Required Tools

- Docker
- kubectl
- kind (Kubernetes in Docker)
- kustomize
- Istio CLI (istioctl)
- Go (version 1.19+)

## 2. Install Prerequisites

### Install KinD

```bash
#!/bin/bash
KIND_VERSION="0.29.0"
KIND_URL="https://kind.sigs.k8s.io/dl/v${KIND_VERSION}/kind-linux-amd64"

echo "Installing kind ${KIND_VERSION}..."
curl -sL -o kind "$KIND_URL"
chmod +x ./kind
sudo mv kind /usr/local/bin
```

### Install kustomize

```bash
#!/bin/bash
KUSTOMIZE_VERSION="5.4.1"
KUSTOMIZE_URL="https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v${KUSTOMIZE_VERSION}/kustomize_v${KUSTOMIZE_VERSION}_linux_amd64.tar.gz"

echo "Installing kustomize ${KUSTOMIZE_VERSION}..."
curl -sL -o kustomize.tar.gz "$KUSTOMIZE_URL"
tar -xzf kustomize.tar.gz
chmod +x kustomize
sudo mv kustomize /usr/local/bin
```

### Install Istio

```bash
#!/bin/bash
ISTIO_VERSION="1.17.8"
ISTIO_URL="https://istio.io/downloadIstio"

echo "Installing Istio ${ISTIO_VERSION}..."
mkdir istio_tmp
pushd istio_tmp >/dev/null
    curl -sL "$ISTIO_URL" | ISTIO_VERSION=${ISTIO_VERSION} sh -
    cd istio-${ISTIO_VERSION}
    export PATH=$PWD/bin:$PATH
    istioctl install -y
popd
```

## 3. Set up KinD Cluster

```bash
# Setup kind environment
sudo swapoff -a
sudo rm -f /swapfile
sudo mkdir -p /tmp/etcd
sudo mount -t tmpfs tmpfs /tmp/etcd

# Create KinD cluster
kind create cluster --config testing/gh-actions/kind-1-33.yaml

# Create kubeflow namespace
kubectl create namespace kubeflow
```

## 4. Deploy Dependencies

### Deploy Profile Controller (Required)

```bash
# Set environment variables
export PROFILE_CONTROLLER_IMG="profile-controller"
export KFAM_IMG="kfam"
export TAG="integration-test"

# Deploy profile controller
cd components
kustomize build profile-controller/config/overlays/kubeflow | kubectl apply -f -
kubectl wait pods -n kubeflow -l kustomize.component=profiles --for=condition=Ready --timeout=300s
```

### Apply Kubeflow Roles

```bash
kustomize build https://github.com/kubeflow/manifests//common/kubeflow-roles/base?ref=master | kubectl apply -f -
```

## Running the Tests Locally

### 1. Deploy Notebook Controller

#### Option 1: Deploy from Upstream (Recommended for Integration Testing)

```bash
# Deploy notebook controller from upstream Kubeflow manifests
kustomize build https://github.com/kubeflow/kubeflow//components/notebook-controller/config/overlays/kubeflow?ref=master | kubectl apply -f -

# Wait for controller to be ready
kubectl wait pods -n kubeflow -l app=notebook-controller --for=condition=Ready --timeout=300s
```

#### Option 2: Build and Deploy Local Version

```bash
cd components/notebook-controller

# Build and load Docker image
make docker-build IMG=notebook-controller TAG=integration-test
kind load docker-image notebook-controller:integration-test

# Deploy with local image
kustomize build config/overlays/kubeflow \
  | sed 's|kubeflownotebookswg/notebook-controller:[a-zA-Z0-9_.-]*|notebook-controller:integration-test|g' \
  | kubectl apply -f -

# Wait for controller to be ready
kubectl wait pods -n kubeflow -l app=notebook-controller --for=condition=Ready --timeout=300s
```

### 2. Create User Profile

```bash
# Apply user profile for testing
kubectl apply -f components/profile-controller/integration/resources/user-profile.yaml

# Wait for namespace creation
while ! kubectl get ns kubeflow-user; do 
  echo "Waiting for kubeflow-user namespace..."
  sleep 1
done

echo "Profile and namespace created successfully"
```

### 3. Run Integration Tests

#### Create Test Notebook

```bash
cd components/notebook-controller/integration

# Apply test notebook
kubectl apply -f resources/test-notebook.yaml

# Wait for notebook to be ready
kubectl wait notebooks -n kubeflow-user -l app=test-notebook --for=condition=Ready --timeout=300s
```

### 4. Controller Unit Tests

```bash
cd components/notebook-controller

# Run unit tests
make test

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run specific test suites
go test -v ./controllers/...
go test -v ./pkg/...
```

### 8. Cleanup

```bash
# Clean up test resources
kubectl delete namespace kubeflow-user

# Delete the kind cluster
kind delete cluster
```

## Integration with Dashboard Components

### Jupyter Web App Integration

```bash
# Deploy Jupyter Web App (JWA)
kustomize build https://github.com/kubeflow/kubeflow//components/crud-web-apps/jupyter/manifests/overlays/istio?ref=master | kubectl apply -f -
kubectl wait pods -n kubeflow -l app=jupyter-web-app --for=condition=Ready --timeout=300s

# Test JWA integration
kubectl port-forward -n kubeflow svc/jupyter-web-app-service 8085:80 &
curl -H "kubeflow-userid: user" localhost:8085
```

### Volumes Web App Integration

```bash
# Deploy Volumes Web App (VWA)
kustomize build https://github.com/kubeflow/kubeflow//components/crud-web-apps/volumes/manifests/overlays/istio?ref=master | kubectl apply -f -
kubectl wait pods -n kubeflow -l app=volumes-web-app --for=condition=Ready --timeout=300s

# Test VWA integration
kubectl port-forward -n kubeflow svc/volumes-web-app-service 8087:80 &
curl -H "kubeflow-userid: user" localhost:8087
```
