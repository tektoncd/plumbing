# (Continuous) Deployment of Tekton Services

This folder includes overlays used to maintain the configuration of Tekton
services in the Tekton infra clusters `dogfooding` and `prow`.

## Deploy Tekton service on-demand
Tekton services can be deployed on-demand using a Tekton task called
`install-tekton-release`. For example, Tekton Pipeline can be deployed as
follows using the `tkn` client:

```
# The releaseBucket is a parameter that points to where the 
# bucket where the release files are stored e.g. gs://tekton-releases/pipeline
export RELEASE_BUCKET=<release-bucket>

# The K8S_CLUSTER is a the name of a secret that contains the k8s configuration
# for k8s cluster where the Tekton service is being deployed to
export K8S_CLUSTER=<k8s-cluster>

# Create a workspace template file with the following content
cat <<EOF > workspace-template.yaml
spec:
 accessModes:
   - ReadWriteOnce
 resources:
   requests:
     storage: 1Gi
EOF

tkn pipeline start \
  -p releaseBucket=$RELEASE_BUCKET \
  -p projectName=pipeline \
  -p version=v0.9.2 \
  -p environment=dogfooding \
  -w name=targetCluster,secret=$K8S_CLUSTER \
  -w name=resources,volumeClaimTemplateFile=workspace-template.yaml
  -w name=credentials,emptyDir=
  install-tekton-release
```

## Make changes to Tekton resources
The Tekton resources to be deployed are built based on `kustomization.yaml` files in the corresponding directory. The [folder-cd-trigger](../cronjobs/dogfooding/manifests/plumbing-tekton/cronjob.yaml) cron job applies the latest resource under the `plumbing/tekton` directory to the cluster on an hourly basis. The `folder-cd-trigger` cronjob triggers the [deploy-from-folder](../resources/cd/folder-template.yaml) Task to build the final resource yaml files using the [kustomize tool](https://github.com/kubernetes-sigs/kustomize) and deploy. 

To validate your changes to the resources locally, please install [kustomize](https://github.com/kubernetes-sigs/kustomize) and run the following command from the `plumbing/tekton/` directory.

```bash
kustomize build
```

Please check the [kustomize documentation](https://github.com/kubernetes-sigs/kustomize#readme) for more information.