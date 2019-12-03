# Tekton Resources for CI/CD

This folder includes `Tasks`, `Pipelines` and other shared resources used to
setup CI/CD pipelines for all repositories in the tektoncd org. It also
includes `tektoncd/plumbing` specific tasks and pipelines.

Resources are organised in folders:
- The [config](config/README.md) folder holds `CronJobs` definition for regular
  tasks, like building images, deploying configuration, nightly releases
- The [images][images/README.md] folder contains the `Dockerfile` and context for
  all container images used by the Tekton project infrastructure.
- The [resources][resources/README.md] folder contains Tekton resources used for
  various automation tasks: building container images, doing releases,
  maintaining the GitHub org and more.
