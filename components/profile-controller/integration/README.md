# Profile Controller Integration Tests

This directory contains integration tests for the Kubeflow Profile Controller and KFAM (Kubeflow Access Management) components.

## 1. Required Tools

- Docker
- kubectl
- kind (Kubernetes in Docker)
- kustomize
- Istio CLI (istioctl)

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

## 4. Install Required CRDs

```bash
# Apply Kubeflow roles and CRDs
kustomize build https://github.com/kubeflow/manifests//common/kubeflow-roles/base?ref=master | kubectl apply -f -
```

## Running the Tests Locally

### 1. Build and Deploy Profile Controller

From the repository root:

```bash
# Set environment variables
export PROFILE_IMG="profile-controller"
export KFAM_IMG="kfam"
export TAG="integration-test"

# Build Profile Controller image
cd components/profile-controller
make docker-build IMG="${PROFILE_IMG}" TAG="${TAG}"
kind load docker-image "${PROFILE_IMG}:${TAG}"

# Build KFAM image
cd ../access-management
make docker-build IMG="${KFAM_IMG}" TAG="${TAG}"
kind load docker-image "${KFAM_IMG}:${TAG}"

# Deploy Profile Controller
cd ../profile-controller
kustomize build config/overlays/kubeflow \
  | sed "s|gcr.io/kubeflow-images-public/profile-controller:[a-zA-Z0-9_.-]*|${PROFILE_IMG}:${TAG}|g" \
  | sed "s|gcr.io/kubeflow-images-public/kfam:[a-zA-Z0-9_.-]*|${KFAM_IMG}:${TAG}|g" \
  | kubectl apply -f -
```

### 2. Wait for Deployment

```bash
# Wait for pods to be ready
kubectl wait --for=condition=Ready pods -n kubeflow -l kustomize.component=profiles --timeout=300s
kubectl wait --for=condition=Available deployment -n kubeflow profiles-deployment --timeout=300s
```

### 3. Run Integration Tests

Navigate to the integration test directory:

```bash
cd components/profile-controller/integration
```

#### Create Test Profile with Resource Quotas

```bash
./test_profile.sh create test-profile-user test-user@example.com
```

#### Validate Profile Resources

```bash
./test_profile.sh validate test-profile-user
```

#### Test Profile Update

```bash
./test_profile.sh update test-profile-user
```

#### Create Simple Profile (without quotas)

```bash
./test_profile.sh create-simple simple-profile simple-user@example.com
```

#### Validate Simple Profile

```bash
./test_profile.sh validate simple-profile
```

#### List All Profiles

```bash
./test_profile.sh list
```

### 6. Cleanup

```bash
# Clean up test profiles
./test_profile.sh delete test-profile-user
./test_profile.sh delete simple-profile

# Delete the kind cluster
kind delete cluster
```
