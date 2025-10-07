# Profile Controller Integration Tests

This directory contains integration tests for the Kubeflow Profile Controller and KFAM (Kubeflow Access Management) components.

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
```

## Run Integration Tests

From the repository root:

```bash
cd components/profile-controller/integration
# create and validate a profile with quotas
./test_profile.sh create test-profile-user test-user@example.com
./test_profile.sh validate test-profile-user
./test_profile.sh update test-profile-user
# create and validate a simple profile
./test_profile.sh create-simple simple-profile simple-user@example.com
./test_profile.sh validate simple-profile
./test_profile.sh list
# cleanup
./test_profile.sh delete test-profile-user
./test_profile.sh delete simple-profile
```

## Cleanup (optional)

```bash
# delete KinD cluster
kind delete cluster
```
