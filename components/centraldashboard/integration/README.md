# Central Dashboard Integration Tests

This directory contains integration tests for the Kubeflow Central Dashboard component.

## 1. Required Tools

- Docker
- kubectl
- kind (Kubernetes in Docker)
- kustomize
- Istio CLI (istioctl)
- curl

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

### 4. Deploy Dependencies

#### Deploy Profile Controller and KFAM

```bash
# Set environment variables
export PROFILE_IMG="profile-controller"
export TAG="integration-test"

# Build profile controller image
cd components/profile-controller
make docker-build-multi-arch IMG="${PROFILE_IMG}" TAG="${TAG}"

# Load image into KinD cluster (auto-detect cluster name)
CLUSTER_NAME=$(kind get clusters | head -n 1)
echo "Loading image into KinD cluster: $CLUSTER_NAME"
kind load docker-image "${PROFILE_IMG}:${TAG}" --name "$CLUSTER_NAME"

# Deploy profile controller with locally built image
cd config
kustomize build overlays/kubeflow \
  | sed "s|ghcr.io/kubeflow/kubeflow/profile-controller:[a-zA-Z0-9_.-]*|${PROFILE_IMG}:${TAG}|g" \
  | sed "s|ghcr.io/kubeflow/kubeflow/kfam:[a-zA-Z0-9_.-]*|ghcr.io/kubeflow/kubeflow/kfam:latest|g" \
  | kubectl apply -f -

# Return to root directory
cd ../../..

# Wait for profile controller
kubectl wait --for=condition=Ready pods -n kubeflow -l kustomize.component=profiles --timeout=300s
kubectl wait --for=condition=Available deployment -n kubeflow profiles-deployment --timeout=300s
```

#### Apply Kubeflow Roles

```bash
kustomize build https://github.com/kubeflow/manifests//common/kubeflow-roles/base?ref=master | kubectl apply -f -
```

## Running the Tests Locally

### 1. Build and Deploy Central Dashboard

From the repository root:

```bash
# Set environment variables
export DASHBOARD_IMG="centraldashboard"
export TAG="integration-test"

# Build and deploy the component
./testing/shared/deploy_component.sh \
  "centraldashboard" \
  "components/centraldashboard" \
  "${DASHBOARD_IMG}" \
  "${TAG}" \
  "manifests" \
  "overlays/istio"

# Wait for pods to be ready
kubectl wait --for=condition=Ready pods -n kubeflow -l app=centraldashboard --timeout=300s
kubectl wait --for=condition=Available deployment -n kubeflow centraldashboard --timeout=300s
```

### 3. Create Test Profile

```bash
# Create test profile for dashboard testing
kubectl apply -f components/profile-controller/integration/resources/profile-dashboard-test.yaml

```

### 4. Run Integration Tests

#### Validate Service

```bash
./testing/shared/test_service.sh validate-service centraldashboard kubeflow
```

#### Start Port Forward for Testing

```bash
./testing/shared/test_service.sh port-forward centraldashboard kubeflow 8082 80
```

#### Test Dashboard Health

```bash
./testing/shared/test_service.sh test-health centraldashboard kubeflow 8082
```

#### Test Dashboard Performance

```bash
./testing/shared/test_service.sh performance-test centraldashboard kubeflow 8082 80 10
```

#### Test Dashboard Metrics

```bash
./testing/shared/test_service.sh test-metrics centraldashboard kubeflow 8082
```

#### Check Dashboard Logs

```bash
./testing/shared/test_service.sh check-logs centraldashboard kubeflow 50
```

#### Check for Errors in Logs

```bash
./testing/shared/test_service.sh check-errors centraldashboard kubeflow
```

### 5. Cleanup

```bash
# Clean up test resources
kubectl delete profile test-dashboard-profile --ignore-not-found=true

# Wait for namespace deletion
for i in {1..30}; do
  if ! kubectl get namespace test-dashboard-profile >/dev/null 2>&1; then
    break
  fi
  echo "Waiting for namespace deletion... (attempt $i/30)"
  sleep 5
done

# Delete the kind cluster
kind delete cluster
```
