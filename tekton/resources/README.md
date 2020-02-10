# Tekton Resources for CI/CD

This folder includes `Tasks`, `Pipelines`, `TriggerTemplates` and other shared
resources used to setup CI/CD pipelines for all repositories in the tektoncd
org. It also includes `tektoncd/plumbing` specific tasks and pipelines.

Resources are organised in folders:
- The "images" contains `Tasks` and trigger resources used to build container
  images
- The [org-permissions](org-permissions/README.md) folder contains Tekton
  resource used to manage the GitHub organisation
- The [release](release/README.md) folder contains Tekton resources used to
  create and verify releases
- The "nightly-release" folder contains trigger resources used to
  trigger nightly releases for the tektoncd projects
