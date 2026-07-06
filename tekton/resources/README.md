# Tekton Resources for CI/CD

This folder includes `Tasks`, `Pipelines`, `TriggerTemplates` and other shared
resources used to setup CD pipelines for all repositories in the tektoncd
org. It also includes `tektoncd/plumbing` specific tasks and pipelines.

Resources are organised in folders:

- The [cd](cd) folder contains the event listener and tasks used to providde
  Tekton based CD services in the infra cluster
- The [images](images) fodler contains `Tasks` and trigger resources used
  to build container images
- The [org-permissions](org-permissions/README.md) folder contains Tekton
  resource used to manage the GitHub organisation
- The [release](release/README.md) folder contains Tekton resources used to
  create and verify releases
- The [nightly-release](nightly-release) folder contains trigger resources used
  to trigger nightly releases for the tektoncd projects
- The [nightly-tests](nightly-tests) folder contains tasks and trigger resources
   used to trigger nightly e2e tests for the tektoncd projects
