## Remove System Namespace Component

This component is aimed to be consumed by the overlays targetted for the Kubeflow installation.

In this case, the manifests will need to install everything in the `kubeflow` namespace and not create
the `profile-controller-system` namespace. Thus the two kubeflow overlays (Istio sidecar and ambient)
will need to both include this common component.
