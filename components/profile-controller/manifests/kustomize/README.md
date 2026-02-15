### Manifests

This folder contains manifests for installing `profile-controller`. The structure is the following:

```
.
├── base
│   ├── crd
│   ├── manager
│   ├── patches
│   └── rbac
├── components
│   ├── istio
│   └── prometheus
├── overlays
│   ├── kubeflow
│   └── standalone
└── samples
```

The breakdown is the following:
- `base`: main install target. It includes the kubebuilder-generated resources under `base/crd`, `base/manager`, and `base/rbac`.
- `components`: reusable kustomize components used by overlays.
- `overlays`: environment-specific install targets.
- `samples`: sample `Profile` custom resources.

Overlay behavior:
- `overlays/kubeflow`: installs `profile-controller` as part of Kubeflow, applies the KFAM patch, and includes Istio resources.
- `overlays/standalone`: installs only `profile-controller` in `profiles-system`.


### Settings

#### Namespace label injection

The Profile Controller applies several labels to every Profile namespace. These labels are configurable by editing the `namespace-labels` ConfigMap. Refer to the current value for usage instruction.
