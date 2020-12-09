# Dogfooding Cluster

The dogfooding runs the instance of Tekton that is used for all the CI/CD needs
of Tekton itself.

- Configuration for the CI is in [tekton](../tekton)
- The cluster has [two node pools](#node-pools)

## Secrets

Secrets which have been applied to the dogfooding cluster but are not committed here are:

- `GitHub` personal access tokens:
  - In the default namespace:
    - `bot-token-github` used for syncing label configuration and org configuration
    - `github-token` used to create a draft release
  - In the `tektonci` namespace:
    - `bot-token-github` used for ?
    - `ci-webhook` contains the secret used to verify pull request webhook requests for
      plumbing CI.
  - In the [mario](../mariobot) namespace:
    - `mario-github-secret` contains the secret used to verify comment webhook requests to
      the mario service are coming from github
    - `mario-github-token` used for updating PRs
  - In the bastion-z namespace:
    - `s390x-kubeconfig` used to access s390x remote k8s cluster to run s390x tests there
- `GCP` secrets:
  - `nightly-account` is used by nightly releases to push releases
  to the nightly bucket. It's a token for service account
  `release-right-meow@tekton-releases.iam.gserviceaccount.com`.
  - `release-secret` is used by Tekton Pipeline to push pipeline artifacts to a
    GCS bucket. It's also used to push images built by cron trigger (or [Mario](../mariobot])
    to the image registry on GCP.
- K8s service account secrets. These secrets are used in pipeline resources of type cluster, to
  give enabled Tekton pipelines to deploy to target clusters with specific service accounts:
  - dogfooding-tekton-cd-token
  - dogfooding-tekton-cleaner-token
  - dogfooding-tektonci-default-token
  - robocat-tekton-deployer-token
  - robocat-tektoncd-cadmin-token
- Lots of other secrets, hopefully we can add more documentation on them
  here as we go.

## Ingresses

Ingress resources use the GCP Load Balancers for HTTPS termination and offload
to kubernetes services in the `dogfooding` cluster.

SSL Certificate are generated automatically using a `ClusterIssuer` managed by
[cert-manager](https://github.com/jetstack/cert-manager/).

- To install `cert-manager` follow this
  [guide](https://docs.cert-manager.io/en/latest/getting-started/)
- To deploy the `ClusterIssuer`:

```bash
kubectl apply -f https://github.com/tektoncd/plumbing/blob/master/tekton/certificates/clusterissuer.yaml
```

- Apply the ingress resources and update the `*.tekton.dev` DNS configuration.
  Ingress resources are deployed along with the corresponding service.

The following DNS names and corresponding ingresses are defined:

- `dashboard.dogfooding.tekton.dev`: [ingress](https://github.com/tektoncd/plumbing/blob/master/tekton/cd/dashboard/overlays/dogfooding/ingress.yaml)

To see the IP of the ingress in the new cluster:

```bash
kubectl get ingress ing
```

## Node Pools

Dogfooding is comprised of two node pools. One is used for workloads that operate with Workload Identity,
a feature of GKE which maps Kubernetes Service Accounts to Google Cloud IAM Service Accounts. The other
is used for workloads that don't use Workload Identity and rely instead on mechanisms like mounted Secrets
or that run unauthenticated. Choosing the correct pool for a workload should really only depend on whether
it utilizes the Workload Identity feature or not.

- `default-pool` is used for most workloads. It doesn't have the GKE Metadata Server enabled
and therefore doesn't support workloads running with Workload Identity.
- `workload-id` has the GKE Metadata Server enabled and is used for workloads operating with
Workload Identity. The only workload that currently requires Workload Identity is "pipelinerun-logs"
which shows Stackdriver log entries for PipelineRuns.

## Manifests

Manifests for various resources are deployed to the dogfooding clusters from different repositories.
For the plumbing repo, manifest are applied nightly through two cronjobs:

- [tekton](https://github.com/tektoncd/plumbing/tree/master/tekton/cronjobs/dogfooding/manifests/plumbing-tekton-nightly)
- [tekton-cronjobs](https://github.com/tektoncd/plumbing/tree/master/tekton/cronjobs/dogfooding/manifests/plumbing-tekton-cronjobs-nightly)

Manifests from other repos (pipeline, dashboard and triggers) are applied manually for now.

### Service Accounts

Service accounts definitions are stored in git and are applied as part of CD, expect for the case of
[Cluster Roles](https://github.com/tektoncd/plumbing/blob/master/tekton/resources/cd/serviceaccount.yaml)
and related bindings, as they would require giving too broad access to the CD service account.

## Tekton Services

Tekton services are deployed using the [`deploy-release.sh`](https://github.com/tektoncd/plumbing/blob/master/scripts/deploy-release.sh)
script, which submits a kubernets `Job` to the `robocat` cluster, to trigger a deployment on the
`dogfooding` cluster. The `Job` triggers and event listener on the `robocat` cluster, and triggers
a Tekton task that downloads a release from the release bucket, optionally applies overlays and
deploys the result to the `dogfooding` cluster using a dedicated service account.
