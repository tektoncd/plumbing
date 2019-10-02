# Tekton Resources for CI/CD

This folder includes `Tasks`, `Pipelines` and other shared Tekton
resource used to setup CI/CD pipelines for all repositories in the
tektoncd org. It also includes `tektoncd/plumbing` specific tasks
and pipelines.

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

An example `TaskRun`:

```
apiVersion: tekton.dev/v1alpha1
kind: TaskRun
metadata:
  generateName: install-tekton-pipeline-
spec:
  taskRef:
    name: install-tekton-release
  inputs:
    resources:
      - name: release-bucket
        resourceRef:
          name: tekton-bucket
      - name: k8s-cluster
        resourceRef:
          name: <cluster-name-DNS-valid>
    params:
      - name: projectName
        value: pipeline
      - name: version
        value: v0.7.0
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
  name: <cluster-name-DNS-valid>
spec:
  type: cluster
  params:
    - name: name
      value: <cluster-name>
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
