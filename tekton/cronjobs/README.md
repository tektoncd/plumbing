# Tekton Cron Jobs

This folder holds kustomize overlays, that are used to maintain cron job
configurations. To add a new cron job to be deployed to the `dogfooding`
cluster, create a folder and add a `kustomization.yaml` into it, along with the
cronjob overlay.

There are three base cron jobs available:
* `nightly-image-build-cron-base` which can be used to build container images
  on a regular basis and push them to a container repo (by default
  gcr.io/tekton-releases/dogfooding/myimage)
* `nighyly-release-cron-base` which is used to trigger the pipeline repo nightly
  builds
* `resource-cd-cron-base` which is used to trigger deployment of a resource

## Existing cron jobs

### Container Images

The following images are built nightly:
* [hub](hub-image-nightly-build-cron/README.md)
* [ko](ko-image-nightly-build-cron/README.md)
* [ko + gcloud](ko-gcloud-image-nightly-build-cron/README.md)
* [kubectl](kubectl-image-nightly-build-cron/README.md)
* [skopeo](skopeo-image-nightly-build-cron/README.md)
* [tkn](tkn-image-nightly-build-cron/README.md)
* [testrunner](pipeline-test-runner-build-cron/README.md)

### Nightly Releases

The following projects are released nightly:
* [pipeline](pipeline-nightly-release-cron/README.md)
* [triggers](triggers-nightly-release-cron/README.md)
* [dashboard](dashboard-nightly-release-cron/README.md)

### Continuous Deployments

The following resources are deployed continuously:
* [prow config](prow-config-cd-hourly-cron/README.md)
* [labels sync](labels-sync-cron/README.md)

## Adding a new cron job

To add a new cronjob for an existing base, follow these steps.

Example folders structure:
```
cronjobs
├── README.md
├── resource-cd-cron-base (existing)
│   ├── kustomization.yaml
│   └── trigger.yaml
└── myresource-cd-daily-cron (added)
    ├── cronjob.yaml
    └── kustomization.yaml
```

Example Kustomization configuration file:
```
# kustomization.yaml
bases:
- ../../resource-cd-cron-base
patchesStrategicMerge:
- cronjob.yaml
nameSuffix: "-myresource"
```

Example Cronjob definition file. This file must give enough context for
`kustomize` to match the correct resource to be patched - in this case to match
`resource-cd-trigger` defined in `resource-cd-cron-base/trigger.yaml`.
It allows cron jobs to override relevant parts of the template trigger, like
schedule or the value of environment variables.
```
# cronjob.yaml
apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: resource-cd-trigger # <-- This should not be changed!
spec:
  schedule: "*/1 * * * *"  # <-- Change the schedule here
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: trigger
            env:
              - name: SINK_URL
                value: "http://el-myconfig-deployer.default.svc.cluster.local:8080"
              - name: GIT_REPOSITORY
                value: "github.com/tektoncd/plumbing"
              - name: GIT_REVISION
                value: "master"
              - name: CONFIG_PATH
                value: "my-great-resource"
              - name: NAMESPACE
                value: "default"
              - name: CLUSTER_RESOURCE
                value: "prow-cluster-config-bot"
```

To generate the YAML for the newly defined cron configuration, run the following:
```
kustomize build tekton/cronjobs/myresource-cd-daily-cron/
```

To apply the cron configuration directly, run the following:
```
kustomize build tekton/cronjobs/myresource-cd-daily-cron/ | kubectl apply -f -
# or
kubectl -k tekton/cronjobs/myresource-cd-daily-cron/
```

## Adding a daily build for a new image

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
├── cronjobs
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
              value: github.com/tektoncd/plumbing
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
kubectl -k tekton/config/myimage-image-nightly-build-cron/
```

1. Check the result:
```
kubectl get cronjobs
```

1. Run the build:
```
kubectl create job --from=cronjob/image-build-cron-trigger-myimage build-myimage-$(date +"%Y%m%d-%H%M")
```
