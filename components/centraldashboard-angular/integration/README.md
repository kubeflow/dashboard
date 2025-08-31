# Central Dashboard Angular Integration Tests

This directory contains integration tests for the Kubeflow Central Dashboard Angular component.

## Prerequisites

Before running these tests locally, you need:

### 1. Required Tools

- Docker
- kubectl
- kind (Kubernetes in Docker)
- kustomize
- Istio CLI (istioctl)
- Node.js (version 18+)
- npm
- curl
- Xvfb (for headless browser testing)

### 2. Install Prerequisites

#### Install KinD

```bash
#!/bin/bash
KIND_VERSION="0.29.0"
KIND_URL="https://kind.sigs.k8s.io/dl/v${KIND_VERSION}/kind-linux-amd64"

echo "Installing kind ${KIND_VERSION}..."
curl -sL -o kind "$KIND_URL"
chmod +x ./kind
sudo mv kind /usr/local/bin
```

#### Install kustomize

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

#### Install Istio

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

#### Install Node.js and npm

```bash
# Install Node.js 18+ (using NodeSource repository)
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# Verify installation
node --version
npm --version
```

#### Install Xvfb (for headless browser testing)

```bash
sudo apt-get update
sudo apt-get install -y xvfb
```

### 3. Set up KinD Cluster

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

#### Apply Kubeflow Roles and CRDs

```bash
kustomize build https://github.com/kubeflow/manifests//common/kubeflow-roles/base?ref=master | kubectl apply -f -

# Apply user profile for testing
kubectl apply -f components/profile-controller/integration/resources/user-profile.yaml
while ! kubectl get ns kubeflow-user; do sleep 1; done

# Apply test notebook
kubectl apply -f components/notebook-controller/integration/resources/test-notebook.yaml
kubectl wait notebooks -n kubeflow-user -l app=test-notebook --for=condition=Ready --timeout=300s
```

## Running the Tests Locally

### 1. Build and Deploy Central Dashboard Angular

From the repository root:

```bash
# Set environment variables
export DASHBOARD_IMG="centraldashboard-angular"
export TAG="integration-test"
export CD_NAMESPACE="kubeflow"
export CD_CLUSTER_DOMAIN="cluster.local"

# Build and deploy the component
./testing/shared/deploy_component.sh \
  "centraldashboard-angular" \
  "components/centraldashboard-angular" \
  "${DASHBOARD_IMG}" \
  "${TAG}" \
  "manifests" \
  "overlays/istio"
```

### 2. Wait for Deployment

```bash
# Wait for pods to be ready
kubectl wait --for=condition=Ready pods -n kubeflow -l app=centraldashboard-angular --timeout=300s
kubectl wait --for=condition=Available deployment -n kubeflow centraldashboard-angular --timeout=300s
```

### 3. Run Backend Integration Tests

#### Validate Service

```bash
./testing/shared/test_service.sh validate-service centraldashboard-angular kubeflow
```

#### Start Port Forward for Testing

```bash
./testing/shared/test_service.sh port-forward centraldashboard-angular kubeflow 8080 80
```

#### Test Dashboard Health

```bash
./testing/shared/test_service.sh test-health centraldashboard-angular kubeflow 8080
```

#### Test Dashboard Performance

```bash
./testing/shared/test_service.sh performance-test centraldashboard-angular kubeflow 8080 80 8
```

#### Test Dashboard Metrics

```bash
./testing/shared/test_service.sh test-metrics centraldashboard-angular kubeflow 8080
```

#### Check Dashboard Logs

```bash
./testing/shared/test_service.sh check-logs centraldashboard-angular kubeflow 50
```

#### Check for Errors in Logs

```bash
./testing/shared/test_service.sh check-errors centraldashboard-angular kubeflow
```

### 4. Run Frontend UI Tests

Navigate to the frontend directory:

```bash
cd components/centraldashboard-angular/frontend
```

#### Install Dependencies

```bash
# Install frontend dependencies
npm install

# Build common library
cd ..
make build-common-lib

# Link the common library
cd frontend
npm link kubeflow
```

#### Run UI Tests with Cypress

##### Option 1: Interactive Mode

```bash
# Start development server
npm run serve &

# Wait for server to be ready
npx wait-on http://localhost:4200

# Open Cypress UI for interactive testing
npm run ui-test
```

##### Option 2: Headless Mode (CI-style)

```bash
# Start development server
npm run serve &

# Wait for server to be ready
npx wait-on http://localhost:4200

# Run tests in Chrome (headless)
npm run ui-test-ci

# Run tests in both Chrome and Firefox (headless)
DISPLAY=:99 xvfb-run -a npm run ui-test-ci-all
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
