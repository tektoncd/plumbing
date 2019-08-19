# Prow

`tektoncd` uses
[`Prow`](https://github.com/kubernetes/test-infra/tree/master/prow)
for CI automation.

- Prow runs in [the tektoncd GCP project](../gcp.md)
- [Ingress is configured to `prow.tekton.dev`](#ingress)
- Prow results are displayed via [gubernator](../gubernator/README.md)
- [Instructions for updating Prow](#updating-prow-itself) and [Prow's Tekton Pipelines instance](#tekton-pipelines-with-prow)
- [Instructions for updating Prow configuration](#updating-prow-configuration)

_[Prow docs](https://github.com/kubernetes/test-infra/tree/master/prow)._
_[See the community docs](../CONTRIBUTING.md#pull-request-process) for more on
Prow and the PR process._

### Secrets

- Prow uses the service account
  `prow-account@tekton-releases.iam.gserviceaccount.com`
- Secrets for this account are configured in [Prow's config.yaml](config.yaml) via
  `gcs_credentials_secret: "test-account"`

## Ingress

- Ingress for prow is configured using
  [cert-manager](https://github.com/jetstack/cert-manager/).
- `cert-manager` was installed via `Helm` using this
  [guide](https://docs.cert-manager.io/en/latest/getting-started/)
- `prow.tekton.dev` is configured as a host on the prow `Ingress` resource.
- https://prow.tekton.dev is pointed at the Cluster ingress address.
- The configuration is in [ingress.yaml](./ingress.yaml)

### Updating Prow itself

Prow has been installed by taking the
[starter.yaml](https://github.com/kubernetes/test-infra/blob/master/prow/cluster/starter.yaml)
and modifying it for our needs.

Updating (e.g. bumping the versions of the images being used) requires:

0. If you are feeling cautious and motivated, manually backup the config values by hand
   (see [prow.yaml](prow.yaml) to see what values will be changed).
1. Manually updating the `image` values and applying any other config changes found in the
   [starter.yaml](https://github.com/kubernetes/test-infra/blob/master/prow/cluster/starter.yaml)
   to our [prow.yaml](prow.yaml).
2. Updating the `utility_images` in our [config.yaml](config.yaml) if the version of
   the `plank` component is changed.
3. Applying the new configuration with:

   ```yaml
    # Step 1: Configure kubectl to use the cluster, doesn't have to be via gcloud but gcloud makes it easy
    gcloud container clusters get-credentials prow --zone us-central1-a --project tekton-releases

    # Step 2: Update Prow itself
    kubectl apply -f prow/prow.yaml

    # Step 2: Update the configuration used by Prow
    kubectl create configmap config --from-file=config.yaml=prow/config.yaml --dry-run -o yaml | kubectl replace configmap config -f -

    # Step 3: Remember to configure kubectl to connect to your regular cluster!
    gcloud container clusters get-credentials ...
   ```
4. Verify that the changes are working by opening a PR and **manually looking at the logs of each check**,
   in case Prow has gotten into a state where failures are being reported as successes.

These values have been removed from the original
[starter.yaml](https://github.com/kubernetes/test-infra/blob/master/prow/cluster/starter.yaml):

- The `ConfigMap` values `plugins` and `config` because they are generated from
  [config.yaml](config.yaml) and [plugins.yaml](plugins.yaml)
- The `Services` which were manually configured with a `ClusterIP` and other routing
  information (`deck`, `tide`, `hook`)
- The `Ingress` `ing` - Configuration for this is in [ingress.yaml](ingress.yaml)
- The `statusreconciler` Deployment, etc. - Created #54 to investigate adding this.
- The `Role` values give `pod` permissions in the `default` namespace as well as `test-pods` -
  The intention seems to be that `test-pods` be used to run the pods themselves, but we
  don't currently have that configured in our [config.yaml](config.yaml).

#### Tekton Pipelines with Prow

[Tekton Pipelines](https://github.com/tektoncd/pipelines) is also installed in the `prow`
cluster so that Prow can trigger the execution of
[`PipelineRuns`](https://github.com/tektoncd/pipeline/blob/master/docs/pipelineruns.md).

[Since Prow only works with select versions of Tekton Pipelines](https://github.com/kubernetes/test-infra/issues/13948)
the version currently installed in the cluster is v0.3.1:

```bash
kubectl apply --filename  https://storage.googleapis.com/tekton-releases/previous/v0.3.1/release.yaml
```

_See also [Tekton Pipelines installation instructions](https://github.com/tektoncd/pipeline/blob/master/docs/install.md)._

##### Hello World Pipeline

Since Prow + Pipelines in this org are a WIP (see
[#922](https://github.com/tektoncd/pipeline/issues/922)),
the only job that is currently configured is
[the hello scott Pipeline](prow/helloscott.yaml).

This `Pipeline` (`special-hi-scott-pipeline`) is executed on every PR to this repo
(`plumbing`) via the `try-out-prow-plus-tekton` Prow job.

### Updating Prow configuration

TODO(#1) Apply config.yaml changes automatically

Changes to [config.yaml](./config.yaml) are not automatically reflected in
the Prow cluster and must be manually applied.

```bash
# Step 1: Configure kubectl to use the cluster, doesn't have to be via gcloud but gcloud makes it easy
gcloud container clusters get-credentials prow --zone us-central1-a --project tekton-releases

# Step 2: Update the configuration used by Prow
kubectl create configmap config --from-file=config.yaml=prow/config.yaml --dry-run -o yaml | kubectl replace configmap config -f -

# Step 3: Remember to configure kubectl to connect to your regular cluster!
gcloud container clusters get-credentials ...
```