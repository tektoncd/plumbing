# Plumbing

This repo holds configuration for infrastructure used across the tektoncd org 🏗️:

- [Prow](#prow) is used for
  [pull request automation]((https://github.com/tektoncd/community/blob/master/process.md#reviews))
- [Ingress](#ingress) configuration for access via `tekton.dev`
- [Gubernator](#guberator) is used for holding and displaying [Prow](#prow) logs
- [Boskos](#boskos) is used to control a pool of GCP projects which end to end tests can run against

## Support

If you need support, reach out [in the tektoncd slack](https://github.com/tektoncd/community/blob/master/contact.md#slack)
via the `#plumbing` channel.

[Members of the Tekton governing board](goverance.md)
[have access to the underlying resources](https://github.com/tektoncd/community/blob/master/governance.md#permissions-and-access).

## Prow

- Prow runs in
  [the GCP project `tekton-releases`](http://console.cloud.google.com/home/dashboard?project=tekton-releases)
- Prow runs in
  [the GKE cluster `prow`](https://console.cloud.google.com/kubernetes/clusters/details/us-central1-a/prow?project=tekton-releases)
- Prow uses the service account
  `prow-account@tekton-releases.iam.gserviceaccount.com`
  - Secrets for this account are configured in
    [Prow's config.yaml](prow/config.yaml) via
    `gcs_credentials_secret: "test-account"`
- Prow configuration is in [infra/prow](./prow)

_[Prow docs](https://github.com/kubernetes/test-infra/tree/master/prow)._
_[See the community docs](../CONTRIBUTING.md#pull-request-process) for more on
Prow and the PR process._

### Updating Prow

TODO(#1) Apply config.yaml changes automatically

Changes to [config.yaml](./prow/config.yaml) are not automatically reflected in
the Prow cluster and must be manually applied.

```bash
# Step 1: Configure kubectl to use the cluster, doesn't have to be via gcloud but gcloud makes it easy
gcloud container clusters get-credentials prow --zone us-central1-a --project tekton-releases

# Step 2: Update the configuration used by Prow
kubectl create configmap config --from-file=config.yaml=prow/config.yaml --dry-run -o yaml | kubectl replace configmap config -f -

# Step 3: Remember to configure kubectl to connect to your regular cluster!
gcloud container clusters get-credentials ...
```

## Ingress

- Ingress for prow is configured using
  [cert-manager](https://github.com/jetstack/cert-manager/).
- `cert-manager` was installed via `Helm` using this
  [guide](https://docs.cert-manager.io/en/latest/getting-started/)
- `prow.tekton.dev` is configured as a host on the prow `Ingress` resource.
- https://prow.tekton.dev is pointed at the Cluster ingress address.

## Gubernator

- Gubernator is configured on App Engine in the project `tekton-releases`.
- It was deployed using `gcloud app deploy .` with no configuration changes.
- The configruation lives in [the gubernator dir](./gubernator).

_[Gubernator docs](https://github.com/kubernetes/test-infra/tree/master/gubernator)._

## Boskos

We use Boskos to manage GCP projects which end to end tests are run against.

- Boskos configuration lives [in the `boskos` directory](./boskos)
- It runs [in the `prow` cluster of the `tekton-releases` project](#prow), in
  the namespace `test-pods`

_[Boskos docs](https://github.com/kubernetes/test-infra/tree/master/boskos)._

### Adding a project

Projects are created in GCP and added to the `boskos/boskos-config.yaml` file.

Make sure the IAM account:
`prow-account@tekton-releases.iam.gserviceaccount.com` has Editor permissions.

Also make sure the GKE API is enabled.
You can do this by browsing to the GKE tab in the Cloud Console.

After editing `boskos/boskos-config.yaml`,
[apply the updated ConfigMap](https://github.com/kubernetes/test-infra/tree/master/boskos#config-update)
to the `tekton-releases` `prow` cluster.
