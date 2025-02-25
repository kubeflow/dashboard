# Dashboard

This repository is being migrated from kubeflow/kubeflow. Fell free to help us moving it according to the Option 3 in https://github.com/kubeflow/kubeflow/issues/7549.

- We need to set up the new kubeflow/dashboard repository readme, owners file, and issue templates.
- We should merge (or close) any obvious PRs for the dashboard components, and write down links for any we aren't ready for (so we don't loose track after migration)
- The commit PR (#xxx) suffix in the new repository should be rewritten as (kubeflow/kubeflow#xxx) so we don't break the links
- We should only move the folders under components/ that actually correspond to the new dashboard components (using git-filter-repo)
- We should only migrate the necessary GitHub actions

