# Tekton Cron Jobs

This folder holds kustomize overlays, that are used to maintain cron job
configurations. To add a new cron job to be deployed to a cluster,
create a folder and add a `kustomization.yaml` into it, along with the
cronjob overlay. Add the created folder to the `kustomization.yaml` of
the containg folder.

There are several [base cron jobs](bases/) available, each linked to a
dedicated trigger template:

* [`release`](../resources/nightly-releases): trigger the pipeline repo
  nightly builds
* [`image-build`](../resources/images/image-build-trigger.yaml): build
  container images and push them to a container repo (by default
  ghcr.io/tektoncd/dogfooding/myimage)
* [`folder`](../resources/cd/folder-template.yaml): trigger deployment
  of manifests or overlays in a folder
* [`configmap`](../resources/cd/configmap-template.yaml): trigger
  deployment a configmap from a YAML stored in git
* [`cleanup`](../resources/cd/cleanup-template.yaml): trigger cleanup
  of `*Run` resources from a namespace
* [`helm`](../resources/cd/helm-template.yaml): trigger deployment of
  an helm chart
* [`tekton-service`](../resources/cd/tekton-template.yaml): deploy a
  Tekton service from a release file with an optional overlay from git

Cronjobs are organized per cluster where they are deployed and run.
Note that a cronjob deployed to a cluster may act on a different one.

```bash
cronjobs
├── dogfooding
├── prow
├── robocat
├── kustomization.yaml
└── README.md
```

## Existing cron jobs

### Container Images

[Images](dogfooding/images) are build on the dogfooding cluster.

### Nightly Releases

The following projects are released nightly on the dogfooding cluster:

* [pipeline](dogfooding/releases/pipeline-nightly/README.md)
* [triggers](dogfooding/releases/triggers-nightly/README.md)
* [dashboard](dogfooding/releases/dashboard-nightly/README.md)

### Continuous Deployments

[Tekton services](dogfooding/tekton) in the Robocat cluster are deployed
nightly from the dogfooding cluster. Other deployments are performed from
the same cluster they target.

## Adding a new cron job

To add a new cronjob for an existing base, follow these steps.

Example folders structure:

```bash
cronjobs
├─── dogfooding
│   ├── cleanup
│   │   ├── kustomization.yaml
│   │   ├── default-nightly
│   │   │   ├── README.md
│   │   │   ├── cronjob.yaml
│   │   │   └── kustomization.yaml
│   │   └── *mynamespace-nightly*
│   │       ├── README.md
│   │       ├── cronjob.yaml
│   │       └── kustomization.yaml
```

Example Kustomization configuration file:

```yaml
# kustomization.yaml
bases:
- ../../../bases/cleanup
patchesStrategicMerge:
- cronjob.yaml
nameSuffix: "-dogfooding-mynamespace"
```

Example Cronjob definition file. This file must give enough context for
`kustomize` to match the correct resource to be patched - in this case to match
[`cleanup-trigger`](bases/cleanup/trigger-resource-cd.yaml).
It allows cron jobs to override relevant parts of the template trigger, like
schedule or the value of environment variables.

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: cleanup-trigger
spec:
  schedule: "0 11 * * *"
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: trigger
            env:
              - name: NAMESPACE
                value: "mynamespace"
              - name: CLUSTER_RESOURCE
                value: "dogfooding-tektoncd-cleaner"
              - name: CLEANUP_KEEP
                value: "200"
```

To generate the YAML for the newly defined cron configuration, run the following:

```bash
kustomize build tekton/cronjobs/dogfooding/cleanup/mynamespace-nightly
```

To apply the cron configuration directly, run the following:

```bash
kustomize build tekton/cronjobs/dogfooding/cleanup/mynamespace-nightly | kubectl apply -f -
# or
kubectl replace -k tekton/cronjobs/dogfooding/cleanup/mynamespace-nightly/
```

To reapply all cronjobs on dogfooding:

```bash
kubectl replace -k tekton/cronjobs/dogfooding
```

Make sure that RBAC is confifured properly to execute pipelinerun/taskrun
triggered by cronjob. For cleanup, new RoleBinding should be added to
[serviceaccount.yaml](../resources/cd/serviceaccount.yaml).

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: tektoncd-cleaner-delete-pr-tr-default
  namespace: mynamespace
subjects:
- kind: ServiceAccount
  name: tekton-cleaner
  namespace: default
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: delete-pr-tr
```

## Adding a daily build for a new image

To build daily a container image, follow these steps:

1. Create a new context folder with the Dockerfile in it:

```bash
tekton
├── images
│   └── myimage
│       └── Dockerfile
```

2. Create a new kustomize folder with `kustomization.yaml` and `cronjob.yaml`.
   Copy the content from an existing one, e.g. `tkn-image-nightly-build-cron`.

```bash
cronjobs
├─── dogfooding
│   ├── images
│   │   └── myimage-nightly
│   │       ├── README.md
│   │       ├── cronjob.yaml
│   │       └── kustomization.yaml
```

3. Edit `cronjob.yaml`. Configure at least CONTEXT_PATH and TARGET_IMAGE.

```yaml
apiVersion: batch/v1
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
              value: main
            - name: TARGET_IMAGE
              value: ghcr.io/tektoncd/plumbing/myimage:docker.io/latest
            - name: CONTEXT_PATH
              value: tekton/images/myimage
```

4. Add PLATFORMS and BUILD_TYPE to `cronjob.yaml` if you want to build multi-arch image. List target architectures for the image. For BUILD_TYPE use [docker](../resources/images/docker-multi-arch-template.yaml) or [ko](../resources/images/ko-multi-arch-template.yaml) value to choose the image build tool.
```
...
            - name: PLATFORMS
              value: "linux/amd64,linux/s390x,linux/ppc64le"
            - name: BUILD_TYPE
              value: docker
```
**Note**
Please, make sure that the image is buildable for listed architectures. For instance, base image in corresponding Dockerfile should support the same(or larger) list of architectures.

5. Edit `kustomization.yaml`. Set the nameSuffix to identify the new cron job.

```yaml
bases:
- ../../../bases/image-build
patchesStrategicMerge:
- cronjob.yaml
nameSuffix: "-myimage"
```

6. Edit `tekton/cronjobs/dogfooding/images/kustomization.yaml`. Add name of your newly created directory

```
- myimage-nightly
```

7. Apply your new job:

```yaml
kubectl -k tekton/cronjobs/dogfooding/images/myimage-nightly/
```

8. Check the result:

```yaml
kubectl get cronjobs
```

9. Run the build:

```yaml
kubectl create job --from=cronjob/image-build-cron-trigger-myimage build-myimage-$(date +"%Y%m%d-%H%M")
```
