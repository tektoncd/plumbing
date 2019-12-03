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
TEKTON_BUCKET_RESOURCE=tekton-bucket
TEKTON_CLUSTER_RESOURCE=k8s-cluster
TEKTON_PROJECT=pipeline
TEKTON_VERSION=v0.9.0

tkn task start \
  -i release-bucket=$TEKTON_BUCKET_RESOURCE \
  -i k8s-cluser=$TEKTON_CLUSTER_RESOURCE \
  -p projectName=$TEKTON_PROJECT \
  -p version=$TEKTON_VERSION
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
TEKTON_BUCKET_RESOURCE=tekton-bucket-nightly-4csms
TEKTON_PIPELINERUN=pipeline-release-nightly-zrgdp-6n44c
TEKTON_VERSION=v20191203-883dd4d5df
TEKTON_NAMESPACE=default

tkn task start \
  -i release-bucket=$TEKTON_BUCKET_RESOURCE \
  -o release-bucket=$TEKTON_BUCKET_RESOURCE \
  -p pipelinerun=$TEKTON_PIPELINERUN \
  -p namespace=$TEKTON_NAMESPACE \
  -p versionTag=$TEKTON_VERSION
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

The task `create-draft-release` calculates the list of PRs merged between the
previous release and a specified revision. It also builds a list of authors and
uses PRs and authors to build a draft new release in GitHub. It also attaches
the `release.yaml` from the release bucket to the GitHub release.

This task must be executed after the release images and YAML have been produced.
Running this task multiple times will create multiple drafts; old drafts have to
be pruned manually when needed.

Once the draft release is created, the release manager needs to edit the draft,
arranging PRs in the right category, highlighting important changes and creating
a release tag line.

Parameters are `package`, `release-name`, `release-tag` and
`previous-release-tag`.
Resources:
- A git resource that points to the release git revision
- A read-only resource that points to the project folder within the release
  bucket for the `release.yaml`.

This resources expects a secret named `github-token` to exists, with a GitHub
token in `GITHUB_TOKEN` with enough privileges to list PRs and create a draft
release.

An example using `tkn`:

```
TEKTON_RELEASE_GIT_RESOURCE=pipeline-git-v0-9-0
TEKTON_BUCKET_RESOURCE=tekton-bucket
TEKTON_PACKAGE=tektoncd/pipeline
TEKTON_VERSION=v0.9.0
TEKTON_OLD_VERSION=v0.8.0
TEKTON_RELEASE_NAME="Bengal Bender"

tkn task start \
  -i source=$TEKTON_RELEASE_GIT_RESOURCE \
  -i release-bucket=$TEKTON_BUCKET_RESOURCE \
  -p package=$TEKTON_PACKAGE \
  -p release-tag=$TEKTON_VERSION \
  -p previous-release-tag=$TEKTON_OLD_VERSION \
  -p release-name=$TEKTON_RELEASE_NAME
  create-draft-release
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
  name: plumbing-git-master
spec:
  type: git
  params:
    - name: revision
      value: master
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
  --param=resources="conditions pipelineresources tasks pipelines taskruns pipelineruns" \
  --param=container-registry=docker-registry.default.svc:5000 \
  --param=package=github.com/tektoncd/pipeline \
  --resource=bucket=<tekton-bucket-resource> \
  --resource=test-cluster=<test-cluster> \
  --resource=plumbing=plumbing-git-master \
  --resource=tests=pipeline-git-v0-7-0 \
  --resource=results-bucket=tekton-results-bucket \
  verify-deploy-test-tekton-release
```
