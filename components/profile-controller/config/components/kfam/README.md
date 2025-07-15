## KFAM Component

This kustomize component is aimed to provide the configuration for enabling KFAM in the Profile Controller.

Since the Profile Controller is expected to be deployed either standalone or alongside Kubeflow, we'll need to
be able to include this functionality only in specific cases.

Also, since now we have more than one flavour of the Kubeflow integration (Istio sidecar vs ambient) we need
to include this common functionality to both overlays.
