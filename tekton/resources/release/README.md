# Tekton Resources for Release Automation

This folder contains various Tekton resource used to automate Tekton's own
release management. These components are written so that they can be used by
all the projects in the `tektoncd` GitHub org.
The core release pipelines are still owned by the specific project.

## Tasks

### Verify Tekton Release

The task `verify-tekton-release-github` compares the YAML of the release
stored in the GitHub release assets, with the YAML of the release stored in the bucket.

Inputs are:
- Param `projectName`: the name of the project (pipeline, trigger,
dashboard, experimental)
- Param `version`: the version to be installed, e.g. "v0.7.0"
- A storage resource, that should point to the release bucket. The release file
is expected to be at `<bucket>/<projectName>/previous/<version>/release.yaml`

### Install Tekton Release

The task `install-tekton-release` installs a release of a Tekton project from the YAML file in the
release bucket.

Inputs are:
- Param `projectName`: the name of the project (pipeline, trigger,
dashboard, experimental)
- Param `version`: the version to be installed, e.g. "v0.7.0"
- A storage resource, that should point to the release bucket. The release file
is expected to be at `<bucket>/<projectName>/previous/<version>/release.yaml`
- A cluster resource, that points to the credentials for the target cluster

An example using `tkn`:

```
export TEKTON_BUCKET_RESOURCE=tekton-bucket
export TEKTON_CLUSTER_RESOURCE=k8s-cluster
export TEKTON_PROJECT=pipeline
export TEKTON_VERSION=v0.9.0

tkn task start \
  -i release-bucket=$TEKTON_BUCKET_RESOURCE \
  -i k8s-cluser=$TEKTON_CLUSTER_RESOURCE \
  -p projectName=$TEKTON_PROJECT \
  -p version=$TEKTON_VERSION \
  install-tekton-release
```

The release task can use a `kustomize` overlay if available. The name of the
ovelay folder is specified via the `environment` parameter.
The overlay folder must contain a `kustomize.yaml` configuration file. It may
also contain a `pre` folder. Any `*.sh` script found in the folder will be
executed before the release is installed.

```
export TEKTON_BUCKET_RESOURCE=tekton-bucket
export TEKTON_CLUSTER_RESOURCE=k8s-cluster
export TEKTON_PROJECT=pipeline
export TEKTON_VERSION=v0.9.0

tkn task start \
  -i release-bucket=$TEKTON_BUCKET_RESOURCE \
  -i k8s-cluser=$TEKTON_CLUSTER_RESOURCE \
  -p projectName=$TEKTON_PROJECT \
  -p version=$TEKTON_VERSION \
  -p environment=robocat \
  install-tekton-release
```

## Save Release Logs

The task `save-release-logs` fetches the logs from a release pipelines and stores
them in the release bucket along with the release YAML.
This task is triggered automatically by the `pipeline-release-post-processing`
event listener, which can be triggered by sending a Cloud Event to it at the
end of the release task, like
[pipeline](https://github.com/tektoncd/pipeline/blob/883dd4d5df5e80f051d8f6b3b357ce5fa0354a70/tekton/publish.yaml#L44-L45)
does.

Parameters are `pipelinerun`, `namespace` and `versionTag`.
A resource that points to the project folder within the release bucket is
needed both as input and output of the task.

An example using `tkn`:

```
export TEKTON_BUCKET_RESOURCE=tekton-bucket-nightly-4csms
export TEKTON_PIPELINERUN=pipeline-release-nightly-zrgdp-6n44c
export TEKTON_VERSION=v20191203-883dd4d5df
export TEKTON_NAMESPACE=default

tkn task start \
  -i release-bucket=$TEKTON_BUCKET_RESOURCE \
  -o release-bucket=$TEKTON_BUCKET_RESOURCE \
  -p pipelinerun=$TEKTON_PIPELINERUN \
  -p namespace=$TEKTON_NAMESPACE \
  -p versionTag=$TEKTON_VERSION \
  -s tekton-logs \
  save-release-logs
```

The bucket resource:
```
$ tkn resource describe tekton-bucket-nightly-4csms
Name:                    tekton-bucket-nightly-4csms
Namespace:               default
PipelineResource Type:   storage

Params
NAME       VALUE
type       gcs
location   gs://tekton-release-nightly/pipeline/
dir        y

Secret Params
FIELDNAME                        SECRETNAME
GOOGLE_APPLICATION_CREDENTIALS   xyz
```

## Create Draft Release

The pipeline `release-draft` calculates the list of PRs merged between the
previous release and a specified revision. It also builds a list of authors and
uses PRs and authors to build a draft new release in GitHub. It also attaches
the `release.yaml` from the release bucket to the GitHub release.

This pipeline must be executed after the release images and YAML have been
produced. Running this task multiple times will create multiple drafts; old
drafts have to be pruned manually when needed.

Once the draft release is created, the release manager needs to edit the draft,
arranging PRs in the right category, highlighting important changes and creating
a release tag line.

Parameters are: `package`, `release-name`, `release-tag`, `previous-release-tag`
, `git-revision`, `bucket` and `rekor-uuid`.

An example using `tkn`. Start defining a few environment variables, obtain the
REKOR_UUID and then run the pipeline:

```shell
export TEKTON_RELEASE_GIT_SHA=9c884fb3d3bf35c0a251936626f2ca9f17b5c183
export TEKTON_PACKAGE=tektoncd/pipeline
export TEKTON_VERSION=v0.9.0
export TEKTON_OLD_VERSION=v0.8.0
export TEKTON_RELEASE_NAME="Bengal Bender"

RELEASE_FILE=https://storage.googleapis.com/tekton-releases/pipeline/previous/${TEKTON_VERSION}/release.yaml
CONTROLLER_IMAGE_SHA=$(curl $RELEASE_FILE | egrep 'gcr.io.*controller' | cut -d'@' -f2)
REKOR_UUID=$(rekor-cli search --sha $CONTROLLER_IMAGE_SHA | grep -v Found | head -1)
echo -e "CONTROLLER_IMAGE_SHA: ${CONTROLLER_IMAGE_SHA}\nREKOR_UUID: ${REKOR_UUID}"

tkn pipeline start \
  --workspace name=shared,volumeClaimTemplateFile=workspace-template.yaml \
  --workspace name=credentials,secret=release-secret \
  -p package="${TEKTON_PACKAGE}" \
  -p git-revision="$TEKTON_RELEASE_GIT_SHA" \
  -p release-tag="${TEKTON_VERSION}" \
  -p previous-release-tag="${TEKTON_OLD_VERSION}" \
  -p release-name="${TEKTON_RELEASE_NAME}" \
  -p bucket="gs://tekton-releases/pipeline" \
  -p rekor-uuid="$REKOR_UUID" \
  release-draft
```

## Pipelines

### Verify Release

The `verify-deploy-test-tekton-release` is a pipeline to verify the release
assets for a Tekton project.

This pipeline performs the following steps:
* validate the release YAML from the bucket against that in GitHub
* deploy the release against a test k8s cluster
* wait for all the deployments and pods to be up and running
* log the version of tools in the test running image
* run the e2e tests
* cleanup resources
* repeat deploy-to-log
* run the YAML tests (if available)
* check the overall results and log success or failure

Params are:
- Param `projectName`: the name of the project (pipeline, trigger,
dashboard, experimental)
- Param `version`: the version to be installed, e.g. "v0.7.0"

Resources are:
- A ro storage resource, that should point to the release bucket. The release file
is expected to be at `<bucket>/<projectName>/previous/<version>/release.yaml`
- A cluster resource, that points to the credentials for the target cluster
- A git resource for the plumbing repo
- A git resource for the repo that holds the tests
- A rw storage resource where the test results and logs are written

The cluster resource pulls the token and cadata from a kubernetes secret like the
following:

```
export CLUSTER_SECRET_NAME=<cluster-name>-secrets
export SERVICE_ACCOUNT_SECRET=<service-account-secret>

cat <<EOF > secret-for-the-cluster-resource.yaml
apiVersion: v1
kind: Secret
metadata:
  name: $CLUSTER_SECRET_NAME
type: Opaque
data:
  cadataKey: $(kubectl get secret/$SERVICE_ACCOUNT_SECRET -o jsonpath="{.data.ca\.crt}")
  tokenKey: $(kubectl get secret/$SERVICE_ACCOUNT_SECRET -o jsonpath="{.data.token}")
EOF
```

The cluster resource itself:
```
apiVersion: tekton.dev/v1alpha1
kind: PipelineResource
metadata:
  name: k8s-cluster
spec:
  type: cluster
    - name: url
      value: <master-username>
    - name: username
      value: <service-account-name>
  secrets:
    - fieldName: token
      secretKey: tokenKey
      secretName: $CLUSTER_SECRET_NAME
    - fieldName: cadata
      secretKey: cadataKey
      secretName: $CLUSTER_SECRET_NAME
```

The read-only release bucket resource:
```
apiVersion: tekton.dev/v1alpha1
kind: PipelineResource
metadata:
  name: tekton-bucket
spec:
  type: storage
  params:
   - name: type
     value: gcs
   - name: location
     value: gs://tekton-releases
   - name: dir
     value: "y"
```

The plumbing git resource:
```
apiVersion: tekton.dev/v1alpha1
kind: PipelineResource
metadata:
  name: plumbing-git-main
spec:
  type: git
  params:
    - name: revision
      value: main
    - name: url
      value: https://github.com/tektoncd/plumbing
```

The test git resource:
```
apiVersion: tekton.dev/v1alpha1
kind: PipelineResource
metadata:
  name: pipeline-git-v0-7-0
spec:
  type: git
  params:
    - name: revision
      value: v0.7.0
    - name: url
      value: https://github.com/tektoncd/pipeline
```

The read-write results bucket resource:
```
apiVersion: tekton.dev/v1alpha1
kind: PipelineResource
metadata:
  name: tekton-results-bucket
spec:
  type: storage
  params:
   - name: type
     value: gcs
   - name: location
     value: gs://tekton-test-results
   - name: dir
     value: "y"
```

The pipeline can be executed using the `tkn` client:
```
tkn pipeline start \
  --param=version=<version> \
  --param=projectName=<tekton-project> \
  --param=namespace=tekton-pipelines \
  --param=resources="pipelineresources tasks pipelines taskruns pipelineruns" \
  --param=container-registry=docker-registry.default.svc:5000 \
  --param=package=github.com/tektoncd/pipeline \
  --resource=bucket=<tekton-bucket-resource> \
  --resource=test-cluster=<test-cluster> \
  --resource=plumbing=plumbing-git-main \
  --resource=tests=pipeline-git-v0-7-0 \
  --resource=results-bucket=tekton-results-bucket \
  verify-deploy-test-tekton-release
```
