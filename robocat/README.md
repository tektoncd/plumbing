# Setting up the Robocat cluster from scratch

These are step by step instructions on how to setup the `robocat` cluster using
the available automation. The automation is based on Tekton, so it requires
a "driver" cluster with Tekton deployed, which will run the tasks required to setup
the `robocat` cluster. This "driver" cluster is the `dogfooding` cluster.

# Point your cluster config to `robocat`

The initial step is done on the `robocat` cluster directly, so point your
configuration to it:

```
kubectl config use-context gke_tekton-nightly_europe-north1-a_robocat
```

# Create a Cluster Admin service account

To setup the [cluster admin](root/README.md) service account, authenticate to
the cluster with an admin user, and apply the content of the `root` folder:

```
kubectl apply -f robocat/root
```

Create a secret `robocat-tektoncd-cadmin-token` in the `dogfooding` cluster,
that holds the token for the `cadmin`. This secret is used by the
`robocat-cadmin` pipeline resource, which is used by the automation to drive
deployments in the `robotcat` cluster.

```
# Fetch the secret data from robocat
CADMIN_SECRET=$(kubectl get -n tektoncd sa/cadmin -o jsonpath='{.secrets[0].name}')
CA_CRT=$(kubectl get -n tektoncd secret/$CADMIN_SECRET -o jsonpath='{.data.ca\.crt}')
TOKEN=$(kubectl get -n tektoncd secret/$CADMIN_SECRET -o jsonpath='{.data.token}')

# Create the secret on dogfooding
cat <<EOF | kubectl --cluster gke_tekton-releases_us-central1-a_dogfooding create -f -
apiVersion: v1
kind: Secret
metadata:
  name: robocat-tektoncd-cadmin-token
type: Opaque
data:
  ca.crt: $CA_CRT
  token: $TOKEN
EOF
```

# Verify the robocat cluster resources

Obtain the URL of the cluster:

```
kubectl cluster-info
```

Ensure that the [`robocat-tekton-deployer`](https://github.com/tektoncd/plumbing/blob/5f9cb51b8530f9bfc5e97e235980767ae53cdec9/tekton/resources/cd/clusters.yaml#L57)
and the [`robocat-cadmin`](https://github.com/tektoncd/plumbing/blob/5f9cb51b8530f9bfc5e97e235980767ae53cdec9/tekton/resources/cd/clusters.yaml#L77) resources point to the correct URL of the cluster.
If not fix them in git and re-apply them to the `dogfooding` cluster.

# Wait... or not

Almost everything else is setup automatically via cronjobs scheduled in the
`dogfooding` cluster. Since the setup of the DNS entry and the creation of the
`ClusterIssuer` at the right time are still not automated, it's best for now
to run through the setup "manually" by triggering the various cronjobs
one by one, at least for the initial setup.
Future hanges to the resources will be deployed nightly from git.

## Point your cluster config to `dogfooding`

From this point on, most of the work will be done on the `dogfooding` cluster,
so switch your configuration to point to it:

```
kubectl config use-context gke_tekton-releases_us-central1-a_dogfooding
```

## Run the cronjobs

Cronjobs can be used to deploy a folder or resources, a config map, an Helm
chart or a Tekton services from a release.

The generic command to run a cronjob is:
```
kubectl create job --from=cronjob/$JOB_NAME $JOB_NAME-$(date +%s)
```

JOB_NAME | Details | Definition | Type
---------|---------|:----------:|:----:
`JOB_NAME=folder-cd-trigger-robotcat-cadmin`| [Namespaces and RBAC](cadmin/README.md) | [Cronjob](../tekton/cronjobs/robocat-cadmin-cron) | [Folder](../tekton/cronjobs/folder-cd-cron-base)
`JOB_NAME=helm-cd-trigger-cert-manager-helm` | [Cert Manager](https://github.com/jetstack/cert-manager) | [Cronjob](../tekton/cronjobs/robocat-cert-manager-helm-cron) | [Helm Chart](../tekton/cronjobs/helm-cd-cron-base)
`JOB_NAME=folder-cd-trigger-robotcat-cluster-issuer` | [`ClusterIssuer`](./certificates/README.md) | [Cronjob](../tekton/cronjobs/robocat-certificates-on-demand) | [Folder](../tekton/cronjobs/folder-cd-cron-base)
`JOB_NAME=helm-cd-trigger-minio-helm` | [Minio S3 Buckets](certificates/README.md) | [Cronjob](../tekton/cronjobs/minio-helm-cron) | [Helm Chart](../tekton/cronjobs/helm-cd-cron-base)


The following jobs are executed with the `tekton-deployer` service account,
instead of `cadmin`. The `tekton-deployer` service account is created during
of the first cronjob `folder-cd-trigger-robotcat-cadmin`.
Before running the next jobs, make sure the secret token for `tekton-deployer`
in the `dogfooding` cluster is up to date:

```
# Fetch the secret data from robocat
TD_SECRET=$(kubectl --cluster gke_tekton-nightly_europe-north1-a_robocat \
  get -n tekton-pipelines sa/tekton-deployer -o jsonpath='{.secrets[0].name}')
CA_CRT=$(kubectl --cluster gke_tekton-nightly_europe-north1-a_robocat \
  get -n tekton-pipelines secret/$TD_SECRET -o jsonpath='{.data.ca\.crt}')
TOKEN=$(kubectl --cluster gke_tekton-nightly_europe-north1-a_robocat \
  get -n tekton-pipelines secret/$TD_SECRET -o jsonpath='{.data.token}')

# Create the secret on dogfooding
cat <<EOF | kubectl --cluster gke_tekton-releases_us-central1-a_dogfooding create -f -
apiVersion: v1
kind: Secret
metadata:
  name: robocat-tekton-deployer-token
type: Opaque
data:
  ca.crt: $CA_CRT
  token: $TOKEN
EOF
```

JOB_NAME | Details | Definition | Type
---------|---------|:----------:|:----:
`JOB_NAME=tekton-release-cd-trigger-robotcat-pipeline` | [Tekton Pipeline](https://github.com/tektoncd/pipeline) | [Cronjob](../tekton/cronjobs/robocat-pipeline-deploy-latest-cron) | [Tekton Release](../tekton/cronjobs/tekton-service-cd-cron-base)
`JOB_NAME=tekton-release-cd-trigger-robotcat-triggers` | [Tekton Triggers](https://github.com/tektoncd/triggers) | [Cronjob](../tekton/cronjobs/robocat-triggers-deploy-latest-cron) | [Tekton Release](../tekton/cronjobs/tekton-service-cd-cron-base)
`JOB_NAME=tekton-release-cd-trigger-robotcat-dashboard` | [Tekton Triggers](https://github.com/tektoncd/dashboard) | [Cronjob](../tekton/cronjobs/robocat-dashboard-deploy-latest-cron) | [Tekton Release](../tekton/cronjobs/tekton-service-cd-cron-base)
`JOB_NAME=folder-cd-trigger-robotcat-tekton-resources`| [Tekton Resources](cadmin/README.md) | [Cronjob](../tekton/cronjobs/robocat-plumbing-tekton-resources-cron) | [Folder](../tekton/cronjobs/folder-cd-cron-base)

Monitor the progress by looking at the logs of recent `TaskRuns`:

```
tkn tr logs -f
```

# Set up the DNS name for the dashboard

The Tekton dashboard is publicly available, to that end an ingress it attached
to it. To get the public IP check the ingress:

```
echo "Public IP: $(kubectl get ing/ing -n tekton-pipelines -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"
```

It may take a couple of minutes before the IP is assigned.
Login to Netifly and create a new DNS `A` record:

```
dashboard.robotcat.tekton.dev   3600   IN   A   <PLUBLIC_IP>
```

# Set up `robocat` to drive deployments to the `dogfooding` cluster

Prerequisite for this step is that the `tekton-deployer` service account has
been created in the `dogfooding` cluster as well. Once that is in place, create
the secret in the `robocat` cluster that holds the service account credentials
need to use `tekton-deployer` on `dogfooding`:

```
# Fetch the secret data from robocat
TD_SECRET=$(kubectl --cluster gke_tekton-releases_us-central1-a_dogfooding \
  get -n tekton-pipelines sa/tekton-deployer -o jsonpath='{.secrets[0].name}')
CA_CRT=$(kubectl --cluster gke_tekton-releases_us-central1-a_dogfooding \
  get -n tekton-pipelines secret/$TD_SECRET -o jsonpath='{.data.ca\.crt}')
TOKEN=$(kubectl --cluster gke_tekton-releases_us-central1-a_dogfooding \
  get -n tekton-pipelines secret/$TD_SECRET -o jsonpath='{.data.token}')

# Create the secret on robocat
cat <<EOF | kubectl --cluster gke_tekton-nightly_europe-north1-a_robocat create -f -
apiVersion: v1
kind: Secret
metadata:
  name: dogfooding-tekton-deployer-token
type: Opaque
data:
  ca.crt: $CA_CRT
  token: $TOKEN
EOF
```

The `cluster` type `PipelineResource` is already deployed on `robocat` and it
uses the secret `dogfooding-tekton-deployer-token`.
