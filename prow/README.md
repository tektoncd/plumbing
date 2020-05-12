# Prow

`tektoncd` uses
[`Prow`](https://github.com/kubernetes/test-infra/tree/master/prow)
for CI automation, though we are moving this over to
[use our own dogfooding](../README.md#the-dogfooding-cluster).

- Prow runs in [the tektoncd GCP project](../README.md#gcp-projects)
- [Ingress is configured to `prow.tekton.dev`](#ingress)
- Prow results are displayed via [gubernator](../gubernator/README.md)
- [Instructions for creating the Prow cluster](#creating-the-prow-cluster)
- [Instructions for updating Prow](#updating-prow-itself) and [Prow's Tekton Pipelines instance](#tekton-pipelines-with-prow)
- [Instructions for updating Prow configuration](#updating-prow-configuration)

_[Prow docs](https://github.com/kubernetes/test-infra/tree/master/prow)._
_[See the community docs](../CONTRIBUTING.md#pull-request-process) for more on
Prow and the PR process._

## Creating the Prow cluster

If you need to re-create the Prow cluster (which includes [the boskos](../boskos/README.md)
running inside), you will need to:

1. [Create a new cluster](#creating-the-cluster)
2. [Create the necessary secrets](#creating-the-secrets)
3. [Apply the new Prow and Boskos](#start-it)
4. [Setup ingress](#ingress)
4. [Update GitHub webhook(s)](#update-github-webhook)

### Creating the cluster

To create a cluster of the right size, using [the same GCP project](../README.md#gcp-projects):

```bash
export PROJECT_ID=tekton-releases
export CLUSTER_NAME=tekton-plumbing

gcloud container clusters create $CLUSTER_NAME \
 --scopes=cloud-platform \
 --enable-basic-auth \
 --issue-client-certificate \
 --project=$PROJECT_ID \
 --region=us-central1-a \
 --machine-type=n1-standard-4 \
 --image-type=cos \
 --num-nodes=8 \
 --cluster-version=latest
```

### Creating the secrets

In order to operate, Prow needs the following secrets, which are referred
to by the following names in our config:

- `GCP` secret: `test-account` is a token for the service account
  `prow-account@tekton-releases.iam.gserviceaccount.com`. This account can
   interact with GCP resources such as uploading Prow results to GCS
   (which is done directly from the containers started by Prow, configured in [config.yaml](config.yaml)) and
   [interacting with boskos clusters](../boskos/README.md).
- `Github` secrets: `hmac-token` for authenticating GitHub and `oauth-token` which is a
   GitHub access token for [`tekton-robot`](https://github.com/tekton-robot),
   used by Prow itself as well as by containers started by Prow via [the Prow config](config.yaml).
   See [the GitHub secret Prow docs](https://github.com/kubernetes/test-infra/blob/068e83ba2f8e9261c0af4cee598c70b92775945f/prow/getting_started_deploy.md#create-the-github-secrets).
- Nightly release secret: `nightly-account` a token for the nightly-release GCP service account

```bash
kubectl apply -f oauth-token.yaml
kubectl apply -f hmac-token.yaml
kubectl apply -f gcp-token.yaml
kubectl apply -f nightly.yaml
```


_To verify that you have gotten all the secrets, you can look for referenced secrets
and service accounts in [the Prow setup](prow.yaml), [the Prow config](config.yaml)
and [the boskos config](../boskos)._

### Start it

Apply the Prow and boskos configuration:

```bash
# Deploy boskos
kubectl apply -f boskos/boskos.yaml # Must be applied first to create the namespace
kubectl apply -f boskos/boskos-config.yaml
kubectl apply -f boskos/storage-class.yaml

# Deploy Prow
kubectl apply -f prow/prow.yaml

# Update Prow with the right configuration
kubectl create configmap config --from-file=config.yaml=prow/config.yaml --dry-run -o yaml | kubectl replace configmap config -f -
kubectl create configmap plugins --from-file=plugins.yaml=prow/plugins.yaml --dry-run -o yaml | kubectl replace configmap plugins -f -
```

### Ingress

To get ingress working properly, you must:

- Install and configure [cert-manager](https://github.com/jetstack/cert-manager/).
  `cert-manager` can be installed via `Helm` using this
  [guide](https://docs.cert-manager.io/en/latest/getting-started/)
- Apply the ingress resource and update the `prow.tekton.dev` DNS configuration.

To apply the ingress resource:

```bash
# Apply the ingress resource, configured to use `prow.tekton.dev`
kubectl apply -f prow/ingress.yaml
```

To see the IP of the ingress in the new cluster:

```bash
kubectl get ingres ing
```

You should be able to navigate to this endpoint in your browser and see the Prow landing page.

Then you can update https://prow.tekton.dev to point at the Cluster ingress address.
(Not sure who has access to this domain name registration, someone in the Linux Foundation?
[dlorenc@](http://github.com/dlorenc) can provide more info.)

### Update GitHub webhook

You will need to configure [GitHubs's webhook(s)](https://developer.github.com/webhooks/)
to point at [the ingress](#ingress) of the new Prow cluster. (Or you can use the domain name.)

For `tektoncd` this is configured at the Org level.

* github.com/tektoncd -> Settings -> Webhooks -> `http://some-ingress-ip/hook`

Update the value of the webhook with `http://ingress-address/hook`
(see [kicking the tires](#kicking-the-tires) to get the ingress IP).

## Updating Prow itself

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
  [config.yaml](config.yaml) and [plugin](plugins.yaml)
- The `Services` which were manually configured with a `ClusterIP` and other routing
  information (`deck`, `tide`, `hook`)
- The `Ingress` `ing` - Configuration for this is in [ingress.yaml](ingress.yaml)
- The `statusreconciler` Deployment, etc. - Created #54 to investigate adding this.
- The `Role` values give `pod` permissions in the `default` namespace as well as `test-pods` -
  The intention seems to be that `test-pods` be used to run the pods themselves, but we
  don't currently have that configured in our [config.yaml](config.yaml).

### Tekton Pipelines with Prow

[Tekton Pipelines](https://github.com/tektoncd/pipelines) is also installed in the `prow`
cluster so that Prow can trigger the execution of
[`PipelineRuns`](https://github.com/tektoncd/pipeline/blob/master/docs/pipelineruns.md).

[Since Prow only works with select versions of Tekton Pipelines](https://github.com/kubernetes/test-infra/issues/13948)
the version currently installed in the cluster is v0.3.1:

```bash
kubectl apply --filename  https://storage.googleapis.com/tekton-releases/previous/v0.3.1/release.yaml
```

_See also [Tekton Pipelines installation instructions](https://github.com/tektoncd/pipeline/blob/master/docs/install.md)._

#### Nightly Tekton Pipelines release

The prow configuration includes a `periodic` job which invokes
[the Tekton Pipelines nightly release Pipeline](https://github.com/tektoncd/pipeline/tree/master/tekton#nightly-releases).

#### Hello World Pipeline

Since Prow + Pipelines in this org are a WIP (see
[#922](https://github.com/tektoncd/pipeline/issues/922)),
the only job (besides [nightly releases](#nightly-tekton-pipelines-release))
that is currently configured is
[the hello scott Pipeline](prow/helloscott.yaml).

This `Pipeline` (`special-hi-scott-pipeline`) is executed on every PR to this repo
(`plumbing`) via the `try-out-prow-plus-tekton` Prow job.

## Updating Prow configuration

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
