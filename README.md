# Kubeflow Dashboard

[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/10673/badge)](https://www.bestpractices.dev/projects/10673)

[Kubeflow Dashboard](https://www.kubeflow.org/docs/components/central-dash/overview/) is the web-based hub of a Kubeflow Platform.
It exposes the access controlled web interfaces for Kubeflow components and more.

> ⚠️ __Note__ ⚠️
>
> We are currently moving the Kubeflow Dashboard codebase from [`kubeflow/kubeflow`](https://github.com/kubeflow/kubeflow) to this repository ([`kubeflow/dashboard`](https://github.com/kubeflow/dashboard)).
> Please see [`kubeflow/kubeflow#7549`](https://github.com/kubeflow/kubeflow/issues/7549) for more information.

## What is Kubeflow Dashboard?

Key features of Kubeflow Dashboard include:

- Access to the [web interfaces](https://www.kubeflow.org/docs/components/central-dash/overview/#navigation) of Kubeflow components.
- Authorization using [Kubeflow Profiles](https://www.kubeflow.org/docs/components/central-dash/profiles/) and Kubernetes Namespaces.
   - _Note, authentication depends on how you [install](https://www.kubeflow.org/docs/started/installing-kubeflow/#kubeflow-platform) your Kubeflow Platform, and is not directly handled by Kubeflow Dashboard._
- Ability to [Customize](https://www.kubeflow.org/docs/components/central-dash/customize/) and include links to third-party applications.

## Components

In this repository, there are multiple components which are versioned and released together:

- `access-management` - Kubeflow Access Management
- `poddefaults-webhooks` - Kubeflow Admission Webhook (PodDefaults)
- `centraldashboard` - Central Dashboard
- `profile-controller` - Kubeflow Profile Controller

## Installation

Kubeflow Dashboard is designed to be deployed as part of a [Kubeflow Platform](https://www.kubeflow.org/docs/started/introduction/#what-is-kubeflow-platform) (not as a standalone component).

Please refer to the [Installing Kubeflow](https://www.kubeflow.org/docs/started/installing-kubeflow/) page for more information.

## Documentation

The official documentation for Kubeflow Dashboard can be found [here](https://www.kubeflow.org/docs/components/central-dash/).

## Migrate from V1

In the past, up until the Kubeflow 1.10 release, the components hosed in this repo were living in the
[`kubeflow/kubeflow`](https://github.com/kubeflow/kubeflow) repository.

To accomodate the migration, the first `v2.0.0` release of this repository contains the same artifacts
from the corresponding [`v1.10.0`](https://github.com/kubeflow/kubeflow/releases/tag/v1.10.0) tag from the `kubeflow/kubeflow` repo. This was done to ensure a smooth transition
between the components of the different repos. You can find more details about the migration in https://github.com/kubeflow/dashboard/issues/118.

### Prerequisites

1. A Kubeflow cluster with 1.10.0 or above
2. `kubectl` configured for access to the above cluster

### Script

You can use the following script to perform the migration from `v1.10.0` components to the `v2.0.0` ones from this repo. The script will
perform the following actions:
1. Remove the existing components of CentralDashboard, Profiles Controller and PodDefaults Webhook
    * The script will not remove any CR or CRD, to ensure no data loss
    * Only K8s resources relevant to the Deployments will be removed (Deployment, ServiceAccount, Service etc)
    * The [`NetworkPolicy`](https://github.com/kubeflow/manifests/blob/v1.10-branch/common/networkpolicies/base/centraldashboard.yaml) from the `kubeflow/manifests` repo, of the CentralDashboard, will be removed
2. Install the manifests from this repo for the Dashboard, Profiles Controller and PodDefaults webhook

```bash
./scripts/upgrade_v1_to_v1.sh
```

## Community

Kubeflow Dashboard is part of the Kubeflow project, refer to the [Kubeflow Community](https://www.kubeflow.org/docs/about/community/) page for more information.

Connect with _other users_ and the [Notebooks Working Group](https://github.com/kubeflow/community/tree/master/wg-notebooks) (maintainers of Kubeflow Dashboard) in the following places:

- [Kubeflow Slack](https://www.kubeflow.org/docs/about/community/#kubeflow-slack-channels) - Join the [`#kubeflow-platform`](https://cloud-native.slack.com/archives/C073W572LA2) channel.
- [Kubeflow Mailing List](https://groups.google.com/g/kubeflow-discuss)

## Contributing

Please see the [`CONTRIBUTING.md`](CONTRIBUTING.md) file for more information.
