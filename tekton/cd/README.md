# (Continuous) Deployment of Tekton Services

This folder includes overlays used to maintain the configuration of Tekton
services in the Tekton infra clusters `dogfooding` and `prow`.

Tekton services can be deployed on-demand using a Tekton task called
`install-tekton-release`. For example, Tekton Pipeline can be deployed as
follows using the `tkn` client:

```
# The RELEASE_BUCKET_RESOURCE is a storage PipelineResource that points to the
# bucket where the release files are stored e.g. gs://tekton-releases/pipeline
export RELEASE_BUCKET_RESOURCE=<release-bucket>

# The K8S_CLUSTER_RESOURCE is a cluster PipelineResource that points to the
# k8s cluster where the Tekton service is being deployed to
export K8S_CLUSTER_RESOURCE=<k8s-cluster>

# The PLUMBING_GIT_RESOURCE is a git PipelineResource that points to the git
# repo where shared plumbing scripts are (usually tektoncd/plumbing)
export PLUMBING_GIT_RESOURCE=<plumbing-git>

tkn task start \
  -i release-bucket=$RELEASE_BUCKET_RESOURCE \
  -i k8s-cluster=$K8S_CLUSTER_RESOURCE \
  -i plumbing-library=$PLUMBING_GIT_RESOURCE \
  -p projectName=pipeline \
  -p version=v0.9.2 \
  -p environment=dogfooding \
  install-tekton-release
```
