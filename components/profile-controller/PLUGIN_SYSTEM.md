# Profile Controller Plugin System

## Overview

The Profile Controller now supports a **ConfigMap-based plugin system** that allows operators to configure plugins dynamically without rebuilding the controller image. This enhancement addresses [Issue #176](https://github.com/kubeflow/dashboard/issues/176).

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Plugin Configuration](#plugin-configuration)
- [Available Plugins](#available-plugins)
- [Migration Guide](#migration-guide)
- [Troubleshooting](#troubleshooting)

## Features

✅ **Dynamic Configuration**: Enable/disable plugins via ConfigMap without redeploying  
✅ **Backward Compatible**: Existing Profile CRs and controller flags still work  
✅ **Security**: Read-only ConfigMap access with proper RBAC  
✅ **Easy to Manage**: Standard Kubernetes ConfigMap workflow  

## Architecture

### Before (Hardcoded)

```go
// Plugins were hardcoded in GetPluginSpec()
switch p.Kind {
case "WorkloadIdentity":
    plugin = &GcpWorkloadIdentity{}
case "AwsIamForServiceAccount":
    plugin = &AwsIAMForServiceAccount{}
}
```

**Problem**: Required code changes and image rebuilds to modify plugin behavior.

### After (ConfigMap-Based)

```yaml
# ConfigMap: profile-controller-plugins-config
apiVersion: v1
kind: ConfigMap
metadata:
  name: profile-controller-plugins-config
  namespace: kubeflow
data:
  plugins.yaml: |
    plugins:
      - kind: WorkloadIdentity
        enabled: true
        config:
          gcpServiceAccount: "kubeflow@project.iam.gserviceaccount.com"
```

**Solution**: Controller reads plugin configuration from ConfigMap at runtime.

## Quick Start

### 1. Create the Plugin ConfigMap

```bash
kubectl apply -f components/profile-controller/manifests/profile-controller-plugins-config.yaml
```

### 2. Configure Your Plugins

Edit the ConfigMap to enable/configure plugins:

```bash
kubectl edit configmap profile-controller-plugins-config -n kubeflow
```

**Example: Enable GCP Workload Identity**

```yaml
data:
  plugins.yaml: |
    plugins:
      - kind: WorkloadIdentity
        enabled: true
        config:
          gcpServiceAccount: "my-sa@my-project.iam.gserviceaccount.com"
      
      - kind: AwsIamForServiceAccount
        enabled: false
        config:
          awsIamRole: ""
          annotateOnly: false
```

### 3. Verify Plugin Configuration

Check the controller logs:

```bash
kubectl logs -n kubeflow deployment/profile-controller -f | grep -i "plugin"
```

You should see:
```
Using plugin configuration from ConfigMap pluginCount=2
Successfully created plugin instance from ConfigMap config
```

### 4. Create a Profile

```bash
kubectl apply -f - <<EOF
apiVersion: kubeflow.org/v1
kind: Profile
metadata:
  name: test-profile
spec:
  owner:
    kind: User
    name: user@example.com
EOF
```

The plugins configured in the ConfigMap will be automatically applied!

## Plugin Configuration

### Configuration Priority

The controller follows this order:

1. **ConfigMap** (highest priority) - If ConfigMap exists with plugins configured
2. **Profile CR** - If no ConfigMap, reads from Profile.Spec.Plugins (legacy)
3. **Controller Flags** - If no ConfigMap, uses `--workload-identity` flag (legacy)

### ConfigMap Structure

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: profile-controller-plugins-config  # Must match this name
  namespace: kubeflow                       # Must be in kubeflow namespace
data:
  plugins.yaml: |                          # Must use this key
    plugins:                               # List of plugin configurations
      - kind: <PluginKind>                 # Plugin type
        enabled: <true|false>              # Enable/disable
        config:                            # Plugin-specific config
          <key>: <value>
```

## Available Plugins

### 1. WorkloadIdentity (GCP)

Grants GCP service account access to Kubernetes service accounts in profile namespaces.

**Kind**: `WorkloadIdentity`

**Configuration**:
```yaml
- kind: WorkloadIdentity
  enabled: true
  config:
    gcpServiceAccount: "my-sa@my-project.iam.gserviceaccount.com"
```

**What it does**:
- Annotates the `default-editor` service account with `iam.gke.io/gcp-service-account`
- Updates the GCP IAM policy to allow workload identity binding
- Enables pods in the profile namespace to access GCP resources

**Requirements**:
- GKE cluster with Workload Identity enabled
- GCP service account created with appropriate permissions
- IAM binding: `gcloud iam service-accounts add-iam-policy-binding`

### 2. AwsIamForServiceAccount (AWS)

Grants AWS IAM role access to Kubernetes service accounts in profile namespaces (IRSA).

**Kind**: `AwsIamForServiceAccount`

**Configuration**:
```yaml
- kind: AwsIamForServiceAccount
  enabled: true
  config:
    awsIamRole: "arn:aws:iam::123456789012:role/my-role"
    annotateOnly: false  # Set to true if IAM trust policy is managed externally
```

**What it does**:
- Annotates the `default-editor` service account with `eks.amazonaws.com/role-arn`
- Updates the AWS IAM role's trust policy (unless `annotateOnly: true`)
- Enables pods in the profile namespace to assume the AWS IAM role

**Requirements**:
- EKS cluster with IRSA (IAM Roles for Service Accounts) enabled
- AWS IAM role created with appropriate trust policy
- OIDC provider configured for the cluster

**annotateOnly mode**:
- Set `annotateOnly: true` if you manage IAM trust policies externally (e.g., via Terraform)
- The controller will only annotate the service account without modifying AWS IAM

## Migration Guide

### From Controller Flags to ConfigMap

**Old way** (controller deployment):
```yaml
containers:
- name: manager
  args:
  - --workload-identity=my-sa@project.iam.gserviceaccount.com
```

**New way** (ConfigMap):
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: profile-controller-plugins-config
  namespace: kubeflow
data:
  plugins.yaml: |
    plugins:
      - kind: WorkloadIdentity
        enabled: true
        config:
          gcpServiceAccount: "my-sa@project.iam.gserviceaccount.com"
```

**Benefits**:
- No need to restart controller pods
- Configuration visible in ConfigMap
- Can be managed via GitOps
- Easier to audit

### From Profile CR to ConfigMap

**Old way** (per-profile config):
```yaml
apiVersion: kubeflow.org/v1
kind: Profile
metadata:
  name: user-profile
spec:
  owner:
    kind: User
    name: user@example.com
  plugins:
  - kind: WorkloadIdentity
    spec:
      gcpServiceAccount: "user-sa@project.iam.gserviceaccount.com"
```

**New way** (centralized config):
```yaml
# ConfigMap applies to ALL profiles
apiVersion: v1
kind: ConfigMap
metadata:
  name: profile-controller-plugins-config
  namespace: kubeflow
data:
  plugins.yaml: |
    plugins:
      - kind: WorkloadIdentity
        enabled: true
        config:
          gcpServiceAccount: "shared-sa@project.iam.gserviceaccount.com"
```

**Note**: ConfigMap configuration is cluster-wide. All profiles will use the same plugin configuration.

### Backward Compatibility

✅ **Existing deployments continue to work**:
- If no ConfigMap exists, controller uses Profile CR plugins
- Controller flags (`--workload-identity`) still work as fallback
- No breaking changes

⚠️ **Migration path**:
1. Create ConfigMap with desired configuration
2. Test with a new profile
3. Once validated, existing profiles will automatically use ConfigMap config
4. Remove per-profile plugin specs from Profile CRs (optional cleanup)

## Troubleshooting

### ConfigMap not being loaded

**Check 1: ConfigMap exists in correct namespace**
```bash
kubectl get configmap profile-controller-plugins-config -n kubeflow
```

**Check 2: ConfigMap has correct data key**
```bash
kubectl get configmap profile-controller-plugins-config -n kubeflow -o yaml | grep plugins.yaml
```

**Check 3: Controller has RBAC permissions**
```bash
kubectl auth can-i get configmaps --as=system:serviceaccount:kubeflow:profile-controller -n kubeflow
```

### Plugin not working

**Check 1: Plugin is enabled**
```bash
kubectl get configmap profile-controller-plugins-config -n kubeflow -o yaml
# Verify "enabled: true" for your plugin
```

**Check 2: Controller logs**
```bash
kubectl logs -n kubeflow deployment/profile-controller -f | grep -i plugin
```

Look for:
- `Using plugin configuration from ConfigMap`
- `Successfully created plugin instance`
- Any error messages

**Check 3: Service account annotations**
```bash
kubectl get sa default-editor -n <profile-namespace> -o yaml
```

Should have annotations like:
- `iam.gke.io/gcp-service-account` (for GCP)
- `eks.amazonaws.com/role-arn` (for AWS)

### YAML parsing errors

**Symptom**: Controller logs show `Failed to unmarshal plugin configuration`

**Solution**: Validate your YAML syntax:
```bash
kubectl get configmap profile-controller-plugins-config -n kubeflow -o yaml | grep -A 20 plugins.yaml | yq eval -
```

Common issues:
- Incorrect indentation
- Missing quotes around values
- Invalid YAML syntax

### Plugins not applied to existing profiles

**Symptom**: New profiles work, but existing profiles don't have plugin configuration

**Solution**: Trigger reconciliation by updating the Profile:
```bash
kubectl annotate profile <profile-name> kubeflow.org/reconcile="$(date +%s)"
```

Or delete and recreate the Profile (data will be preserved).

## Advanced Topics

### Adding Custom Plugins

To add a new plugin type:

1. Implement the `Plugin` interface in a new file:
```go
type MyCustomPlugin struct {
    Config string `json:"config"`
}

func (p *MyCustomPlugin) ApplyPlugin(r *ProfileReconciler, profile *Profile) error {
    // Implementation
}

func (p *MyCustomPlugin) RevokePlugin(r *ProfileReconciler, profile *Profile) error {
    // Implementation
}
```

2. Register in `plugin_config.go`:
```go
const KIND_MY_CUSTOM_PLUGIN = "MyCustomPlugin"

// In GetPluginInstanceFromConfig()
case KIND_MY_CUSTOM_PLUGIN:
    pluginIns = &MyCustomPlugin{}
```

3. Add to ConfigMap:
```yaml
plugins:
  - kind: MyCustomPlugin
    enabled: true
    config:
      config: "custom-value"
```

### Monitoring Plugin Status

The controller logs plugin operations with structured logging:

```bash
# Watch plugin application
kubectl logs -n kubeflow deployment/profile-controller -f | grep ApplyPlugin

# Watch plugin revocation
kubectl logs -n kubeflow deployment/profile-controller -f | grep RevokePlugin
```

## References

- [Issue #176](https://github.com/kubeflow/dashboard/issues/176) - Original issue
- [Kubeflow Pipelines Metacontroller](https://github.com/kubeflow/pipelines) - Design inspiration
- [GKE Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
- [EKS IAM Roles for Service Accounts](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html)
