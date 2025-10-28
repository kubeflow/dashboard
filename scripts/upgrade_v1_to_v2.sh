#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

echo "--------------------------------------------------------------------------------------"
echo "Running the upgrade script to migrate the Kubeflow Dashboard components to V2 release."
echo "--------------------------------------------------------------------------------------"

PROFILES_LABELS=kustomize.component=profiles
DASHBOARD_LABELS=app.kubernetes.io/component=centraldashboard
PODDEFAULT_LABELS=app.kubernetes.io/component=poddefaults

# Helper function for removing all K8s resources of a Kubeflow Component. This will include all resources
# relevant to the Deployment, but not CRDs. This is done to ensure we don't accidentally delete CRs
# (like PodDefaults) in user namespace.
remove-component() {
  label=$1
  namespace_resources="deployment service role rolebinding configmap serviceaccount virtualservice authorizationpolicy certificate secret"
  cluster_resources="clusterrole clusterrolebinding mutatingwebhookconfigurations"

  echo -e "\nWill remove namespaced resources with labels: $label"
  for resource in $namespace_resources; do
    echo "Removing all $resource objects..."
    kubectl delete -n kubeflow -l $label $resource
    echo "Successfully removed all $resource objects"
  done

  for resource in $cluster_resources; do
    echo "Removing all $resource objects..."
    kubectl delete -l $label $resource
    echo "Successfully removed all $resource objects"
  done
}


echo -e "\nRemoving PodDefaults component..."
remove-component $PODDEFAULT_LABELS
echo -e "\nSuccessfully removed PodDefaults!"

echo -e "\nRemoving Profiles component..."
remove-component $PROFILES_LABELS
echo -e "\nSuccessfully removed Profiles!\n"

echo -e "\nRemoving Centraldashboard component..."
remove-component $DASHBOARD_LABELS
echo "Removing NetworkPolicy created by kubeflow/manifests repo..."
kubectl delete networkpolicy -n kubeflow centraldashboard || echo "No NetworkPolicy from manifests repo found. Continuing..."
echo -e "\nSuccessfully removed Centraldashboard!"

echo "-----------------------------------------------"
echo "Installing the updated Dashboard V2 components."
echo "-----------------------------------------------"

echo -e "\nApplying Dashboard component..."
echo -e "--------------------------------"
kustomize build \
  $SCRIPT_DIR/../components/centraldashboard/manifests/overlays/istio \
  | kubectl apply -f -

echo "Waiting for Dashboard Deployment to become available..."
kubectl wait -n kubeflow \
  --for=condition=Available \
  deployment \
  dashboard \
  --timeout=5m
echo -e "Successfully applied the Dashboard component!"

echo -e "\nApplying PodDefaults component..."
echo -e "----------------------------------"
kustomize build \
  $SCRIPT_DIR/../components/poddefaults-webhooks/manifests/overlays/cert-manager \
  | kubectl apply -f -

echo "Waiting for PodDefaults Webhook Deployment to become available..."
kubectl wait -n kubeflow \
  --for=condition=Available \
  deployment \
  poddefaults-webhook-deployment \
  --timeout=5m
echo -e "Successfully applied the PodDefaults component!"

echo -e "\nApplying Profile Controller component..."
echo -e "-----------------------------------------"
kustomize build \
  $SCRIPT_DIR/../components/profile-controller/config/overlays/kubeflow/ \
  | kubectl apply -f -

echo "Waiting for Profiles Controller Deployment to become available..."
kubectl wait -n kubeflow \
  --for=condition=Available \
  deployment \
  profiles-deployment \
  --timeout=5m
echo -e "Successfully applied the Profile Controller component!"

echo -e "\nSuccessfully applied Dashboard V2 components!\n"
