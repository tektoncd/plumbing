# Nightly Releases

This folder contains resources used for Tekton nightly releases.
The releases are triggered via [cronjobs](../../cronjobs/dogfooding/releases).

## Shared Resources

All resources stored directly in this folder are shared across the release
jobs for the different projects.

Triggers and trigger templates have a shared part that is stored under [base](./base). The trigger template shared fragment defines the input parameters that
are available to be passed down to the release pipelines:

```yaml
  - name: buildID
    description: The ID of the build. This is used to build artifact tracking.
  - name: gitrevision
    description: The Git revision to be used for the release.
  - name: gitrepository
    description: The Git repository to be used for the release.
  - name: versionTag
    description: The version tag to be applied to published images.
  - name: imageRegistry
    description: Registry where the images will be published to.
    default: gcr.io/tekton-nightly
  - name: projectName
    description: Name of the Tekton project to release (e.g. pipeline, triggers, etc).
```

## Project Specific Resources

Trigger templates bind the input parameters to the release pipeline. Since the
release pipeline is project specific, that part of the trigger template is
defined in project specified [overlays](./overlays).

Triggers are customized with the project name in the CEL filter, to drive cron
triggers to the correct release pipeline.

The pipeline, several tasks and pipeline resources are hosted in the repository
of the specific project, and pulled-in using `kustomize` remote resource abilities, for instance:

```yaml
resources:
  - github.com/tektoncd/dashboard/tekton/?ref=master
```

## LimitRange

The namespace where nightly builds are executed may require a `LimitRange`,
depending on the version of containerd, as a consequence of this issue: https://github.com/containerd/containerd/issues/4837#issuecomment-772840232.

```
apiVersion: v1
kind: LimitRange
metadata:
  name: limits
spec:
  limits:
  - defaultRequest:
      cpu: 100m
    type: Container
```