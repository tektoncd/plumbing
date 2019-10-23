# Tekton Deployment Config

This folder holds kustomize overlays, that can be used to deploy resources
defined in the tekton folder.

For now this is only used to maintain cron job configurations. To add a new
cron job to be deployed to the `dogfooding` cluster, create a folder and add
a kustomization.yaml into it, along with the cronjob overlay.

There are two base cronjbs available:
* `nightly-image-build-cron-base` which can be used to build container images
  on a regular basis and push them to a container repo (by default
  gcr.io/tekton-releases/dogfooding/myimage)
* `nighyly-release-cron-base` which is used to trigger the pipeline repo nightly
  builds

Example folders structure:
```
tekton
├── README.md
├── config
│   ├── README.md
│   ├── nightly-image-build-cron-base
│   │   ├── kustomization.yaml
│   │   └── trigger-image-build.yaml
│   ├── nightly-release-cron-base
│   │   ├── kustomization.yaml
│   │   └── trigger-with-uuid.yaml
│   ├── pipeline-nightly-release-cron
│   │   ├── cronjob.yaml
│   │   └── kustomization.yaml
│   └── tkn-image-nightly-build-cron
│       ├── cronjob.yaml
│       └── kustomization.yaml
├── images
│   └── tkn
│       └── Dockerfile
```

Example Kustomization configuration file:
```
# kustomization.yaml
bases:
- ../../cron
patchesStrategicMerge:
- cronjob.yaml
nameSuffix: "-pipeline-nightly-release"
```

Cronjob definition file:
```
# cronjob.yaml
apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: pipeline-cron-trigger
spec:
  schedule: "*/1 * * * *"  # <-- Change the schedule here
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: trigger
            env: SINK_URL
            value: [URL of the event-binding sink] # <-- Change the URL here
          initContainers:
          - name: git
            env: GIT_REPO
            value: [URL of the git repo - no protocol] # <-- Change the REPO here
```

To generate the YAML for a cron configuration, run the following:
```
kustomize build tekton/config/pipeline-nighty-release/
```

To apply the cron configuration directly, run the following:
```
kustomize build tekton/config/pipeline-nightly-release-cron/ | kubectl apply -f -
```

# Adding a daily build for a new image

To build daily a docker image, follow these steps:

1. Create a new context folder with the Dockerfile in it:
```
tekton
├── images
│   └── myimage
│       └── Dockerfile
```

1. Create a new kustomize folder with `kustomization.yaml` and `cronjob.yaml`.
   Copy the content from an existing one, e.g. `tkn-image-nightly-build-cron`.
```
tekton
├── config
│   └── myimage-image-nightly-build-cron
│       ├── cronjob.yaml
│       └── kustomization.yaml
```

1. Edit `cronjob.yaml`. Configure at least CONTEXT_PATH and TARGET_IMAGE.
```
apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: image-build-cron-trigger
spec:
  schedule: "0 2 * * *"
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: trigger
            env:
            - name: SINK_URL
              value: el-image-builder.default.svc.cluster.local:8080
            - name: GIT_REPOSITORY
              value: github.com/tekton/plumbing
            - name: GIT_REVISION
              value: master
            - name: TARGET_IMAGE
              value: gcr.io/tekton-releases/dogfooding/myimage:latest
            - name: CONTEXT_PATH
              value: tekton/images/myimage
```

1. Edit `kustomization.yaml`. Set the nameSuffix to identify the new cron job.
```
bases:
- ../nightly-image-build-cron-base
patchesStrategicMerge:
- cronjob.yaml
nameSuffix: "-myimage"
```

1. Apply your new job:
```
kustomize build tekton/config/myimage-image-nightly-build-cron/ | kubectl apply -f -
```

1. Check the result:
```
kubectl get cronjobs
```

1. Run the build:
```
kubectl create job --from=cronjob/image-build-cron-trigger-myimage build-myimage-$(date +"%Y%m%d-%H%M")
```
