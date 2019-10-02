<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Kubeflow CI with tektoncd pipelines](#kubeflow-ci-with-tektoncd-pipelines)
  - [Use Case](#use-case)
  - [Background information on TektonCD pipelineruns, pipelines and tasks](#background-information-on-tektoncd-pipelineruns-pipelines-and-tasks)
  - [Parameterization](#parameterization)
  - [Secrets](#secrets)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Kubeflow CI with tektoncd pipelines

### Use Case

This uses TektonCD [pipelinerun](https://github.com/tektoncd/pipeline/blob/master/docs/pipelineruns.md) to enable the following use case:

1. A PR is merged into kubeflow/kubeflow updating central dashboard
1. The merged commit is 1234
1. This tekton pipelinerun is triggered to build the central dashboard image from commit @1234 in kubeflow.
1. The pipelinerun edits manifests/common/centraldashboard/base/kustomization.yaml and adds the new image tag
1. The pipelinerun calls `make generate; make test` 
1. If successful then checks in the changes 
1. Opens a PR with the updated kubeflow/manifests that uses the newly built image
1. Approvers LGTM the PR to kubeflow/manifests and it gets merged

### Background information on TektonCD pipelineruns, pipelines and tasks

A TektonCD PipelineRun takes 1 Pipeline and N PipelineResources.
The PipelineResources can be git repos, git pull requests, docker images.
These resources are made available to the Pipeline via PipelineRun.

The general relationship between TektonCD resources is shown below:

```
── PipelineRun
   ├── PipelineResources
   └── Pipeline
       └── Tasks
```

In this use case the following instance is created:

```
── ci-centraldashboard-pipeline-run
   ├── resources
   │   ├── image
   │   │   └── centraldashboard+digest
   │   └── git 
   │       ├── kubeflow+revision
   │       └── manifests+revision 
   └── pipeline
       └── tasks
           ├── build-push
           └── update-manifests
```

The PipelineRun includes a Pipeline that has 2 tasks and 3 PipelineResources of type image (centraldashboard) and git (kubeflow, manifests). The Tasks reference these resources in their inputs or outputs. 

### Parameterization 

The PipelineRun specifies PipelineResources which are passed down to the the Pipeline and Tasks.
The Pipeline specifies Task References and their parameters. 
Reuse of centraldashboard requires changing the parameters in PipelineRun and Pipeline.
The PipelineRun, Pipeline, Tasks and PipelineResources are parameterized by kustomize vars.
Changing the values in params.env will allow a different component to be used.

The parameters are noted below, those with an asterix should change per component:
Those parameters without an asterix allow different gcr.io locations, namespace and pvc_mount_path.
This can be run locally (for example using a local cluster via `kind create cluster`)

```
  container_image=gcr.io/kubeflow-ci/test-worker:latest
* docker_target=serve
* image_name=centraldashboard
  image_url=gcr.io/kubeflow_public_images
* kubeflow_repo_revision=1234
* kubeflow_repo_url=git@github.com:kubeflow/kubeflow.git
* manifests_repo_revision=master
* manifests_repo_url=git@github.com:kubeflow/manifests.git
  namespace=kubeflow-test-infra
* path_to_context=components/centraldashboard
* path_to_docker_file=components/centraldashboard/Dockerfile
* path_to_manifests_dir=common/centraldashboard/base
  pvc_mount_path=/kubeflow
```

### Secrets

The secrets file has been supplied with no tokens and should have tokens generated. 
The file itself should not be checked in with valid tokens. 
- gcp-credentials
- kaniko-secret (same as gcp-credentials, use by kaniko)
- github-ssh
- github-token
