# Poddefaults Webhook Integration Tests

This directory contains integration tests for the Kubeflow PodDefaults Webhook component.

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
./testing/shared/install_poddefaults_webhook.sh
```

## Run Integration Tests

From the repository root:

```bash
cd components/poddefaults/integration
# validate webhook setup
./test_poddefault.sh validate-webhook kubeflow
# namespace and mutation tests
./test_poddefault.sh create-namespace test-poddefaults
./test_poddefault.sh create-poddefault test-poddefaults test-poddefault
./test_poddefault.sh test-mutation test-poddefaults test-poddefault test-pod
./test_poddefault.sh create-multi-poddefault test-poddefaults test-poddefault
./test_poddefault.sh test-multi-mutation test-poddefaults test-poddefault test-pod
./test_poddefault.sh test-error-handling test-poddefaults
# cleanup test namespace
./test_poddefault.sh cleanup test-poddefaults
```

## Cleanup (optional)

```bash
# delete KinD cluster
kind delete cluster
```
