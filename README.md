# Plumbing

This repo holds configuration for infrastructure used across the tektoncd org 🏗️:

- Automation runs [in the tektoncd GCP projects](clusters/README.md#gcp-projects), including
  [clusters](#clusters)
- [Tekton](tekton/README.md) is used to release projects, build docker images and run periodic jobs
- [Ingress](prow/README.md#ingress) configuration for access via `tekton.dev`
- [Gubernator](gubernator/README.md) is used for holding and displaying [Prow](prow/README.md) logs
- [Boskos](boskos/README.md) is used to control a pool of GCP projects which end to end tests can run against
- [Peribolos](tekton/resources/org-permissions/README.md) is used to control org and repo permissions

## Support

If you need support, reach out [in the tektoncd slack](https://github.com/tektoncd/community/blob/master/contact.md#slack)
via the `#plumbing` channel.

[Members of the Tekton governing board](goverance.md)
[have access to the underlying resources](https://github.com/tektoncd/community/blob/master/governance.md#permissions-and-access).

## Clusters

Tekton uses several kubernetes clusters:

* [dogfooding](#the-dogfooding-cluster) which exists in [tekton-releases](#gcp-projects)
* [robocat](robocat/) which exists in [tekton-nightly](#gcp-projects)
* The cluster [prow](../prow) also exists in [tekton-releases](#gcp-projects)

## GCP projects

Automation for the `tektoncd` org runs in a GKE cluster which
[members of the governing board](https://github.com/tektoncd/community/blob/master/governance.md#permissions-and-access)
have access to.

There are several GCP projects used by Tekton:
- The GCP project that is used for GKE, storage, etc. is called
  [`tekton-releases`](http://console.cloud.google.com/home/dashboard?project=tekton-releases). It has several GKE clusters:
  - The GKE cluster that [`Prow`](prow/README.md), `Tekton`, and [`boskos`](boskos/README.md) run in is called
    [`prow`](https://console.cloud.google.com/kubernetes/clusters/details/us-central1-a/prow?project=tekton-releases) and is used
  - The GKE cluster that is used for nightly releases and other dogfooding is called
    [`dogfooding`](https://console.cloud.google.com/kubernetes/clusters/details/us-central1-a/dogfooding?project=tekton-releases)
- The GCP project
  [`tekton-nightly`](http://console.cloud.google.com/home/dashboard?project=tekton-nightly)
  is used to hold nightly release artifacts and [the robocat cluster](#the-robocat-cluster)

The script [addpermissions.py](addpermissions.py) gives users access to these projects.

### The prow cluster

[The prow cluster](prow) is where we run Prow, which currently does a lot of our CI, though
we are trying to [dogfood](#the-dogfooding-cluster) more and more.

#### Prow secrets

Secrets which have been applied to the prow cluster but are not committed here are:

- `GitHub` secrets:
 - `bot-token-github` in the default namespace
 - `bot-token-github` in the github-admin namespace

### The robocat cluster

[The robocat cluster](robocat) is where we test the nightly releases of all Tekton projects.

#### Robocat secrets

Secrets which have been applied to the robocat cluster but are not committed here are:

- [cluster admin secret](robocat/README.md#create-a-cluster-admin-service-account)
- [secrets used by cronjobs](robocat/README.md#run-the-cronjobs)
- [deployment secrets](robocat/README.md#set-up-robocat-to-drive-deployments-to-the-dogfooding-cluster)

### The Dogfooding cluster

The dogfooding cluster is where we run Tekton for CI. Configuration for the CI itself lives
in [the tekton folder](tekton). This cluster is part of
[the tekton-releases GCP project](#gcp-projects)

#### Dogfooding Secrets

Secrets which have been applied to the dogfooding cluster but are not committed here are:

- `GitHub` secrets:
  - In the default namespace:
    - `bot-token-github` used for syncing label configuration and org configuration
    - `github-token` used to create a draft release
  - In the `tektonci` namespace:
    - `bot-token-github` used for ?
  - In the [mario](../../mariobot) namespace:
    - `mario-github-secret` contains the secret used to verify requests are coming from github
    - `mario-github-token` used for updating PRs
- `GCP` secrets:
  - `nightly-account` is used by nightly releases to push releases
  to the nightly bucket. It's a token for service account
  `release-right-meow@tekton-releases.iam.gserviceaccount.com`.
  - `release-secret` is used by Tekton Pipeline to push pipeline artifacts to a
    GCS bucket. It's also used to push images built by cron trigger (or [Mario](../../mariobot])
    to the image registry on GCP.
- Lots of other secrets, hopefully we can add more documentation on them
  here as we go.