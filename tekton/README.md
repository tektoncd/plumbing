# Resources for CI/CD

This folder includes `Tasks`, `Pipelines` and other shared resources used to
setup CI/CD pipelines for all repositories in the tektoncd org. It also
includes `tektoncd/plumbing` specific tasks and pipelines.

Resources are organised in folders:
- The [config](config/README.md) folder holds `CronJobs` definition for regular
  tasks, like building images, deploying configuration, nightly releases
- The [images](images/README.md) folder contains the `Dockerfile` and context for
  all container images used by the Tekton project infrastructure.
- The [resources](resources/README.md) folder contains Tekton resources used for
  various automation tasks: building container images, doing releases,
  maintaining the GitHub org and more.
- The [cd](cd/README.md) folder contains kustomize overlays, used to deploy the
  various Tekton projects to the infra clusters.

# Secrets

Some of the resources require secrets to operate.
- `GitHub` secrets: `bot-token-github` used for syncing label configuration and
  org configuration requires, `github-token` used to create a draft release
- `GCP` secrets: `nightly-account` is used by nightly releases to push releases
  to the nightly bucket. It's a token for service account
  `release-right-meow@tekton-releases.iam.gserviceaccount.com`.
  `release-secret` is used by Tekton Pipeline to push pipeline artifacts to a
  GCS bucket. It's also used to push images built by cron trigger (or Mario)
  to the image registry on GCP.
