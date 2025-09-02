# Admission Webhook Integration Tests

This directory contains integration tests for the Kubeflow PodDefaults Admission Webhook component.

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
# Create KinD cluster
kind create cluster --config testing/gh-actions/kind-1-33.yaml

# Create kubeflow namespace
kubectl create namespace kubeflow
```

## 4. Install Cert Manager

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.8.0/cert-manager.yaml
kubectl wait --for=condition=Ready pods -n cert-manager -l app=cert-manager --timeout=300s
```

## Running the Tests Locally

### 1. Build and Deploy the Admission Webhook

From the repository root:

```bash
# Set environment variables
export IMG="admission-webhook"
export TAG="integration-test"

# Build and deploy the component
./testing/shared/deploy_component.sh \
  "admission-webhook" \
  "components/admission-webhook" \
  "${IMG}" \
  "${TAG}" \
  "manifests" \
  "overlays/cert-manager"
```

### 2. Wait for Deployment

```bash
# Wait for pods to be ready
kubectl wait --for=condition=Ready pods -n kubeflow -l app=poddefaults --timeout=300s
kubectl wait --for=condition=Available deployment -n kubeflow admission-webhook-deployment --timeout=300s
```

### 3. Run Integration Tests

Navigate to the integration test directory:

```bash
cd components/admission-webhook/integration
```

#### Validate Webhook Configuration

```bash
./test_poddefault.sh validate-webhook kubeflow
```

#### Create Test Namespace

```bash
./test_poddefault.sh create-namespace test-poddefaults
```

#### Create and Test Basic PodDefault

```bash
./test_poddefault.sh create-poddefault test-poddefaults test-poddefault
./test_poddefault.sh test-mutation test-poddefaults test-poddefault test-pod
```

#### Test Multiple PodDefaults

```bash
./test_poddefault.sh create-multi-poddefault test-poddefaults test-poddefault
./test_poddefault.sh test-multi-mutation test-poddefaults test-poddefault test-pod
```

#### Test Error Handling

```bash
./test_poddefault.sh test-error-handling test-poddefaults
```

### 4. Check Logs and Troubleshoot

```bash
# Check webhook logs
kubectl logs -n kubeflow -l app=poddefaults --tail=100

# Check for errors in webhook logs
kubectl logs -n kubeflow -l app=poddefaults --tail=100 | grep -i error || echo "No errors found in webhook logs"
```

### 5. Cleanup

```bash
# Clean up test resources
./test_poddefault.sh cleanup test-poddefaults

# Delete the kind cluster
kind delete cluster
```
