# Infrastructure Documentation

This folder contains our infrastructure documentation. These documents may be
of interest for the team that maintain the Tekton own CI/CD setup as well as
for anyone interested in using Tekton to run (part of) their own CI/CD
infrastructure.

* [Clusters](#clusters)
* [GCP Projects](#gcp-projects)
* [DNS](#dns)

## Clusters

The infra system relies on several different kubernetes clusters, three are
static and the rest are dynamic (provisioned on demand).

- [*prow*](prow.md): Prow, Boskos and Tekton run in this cluster.
  This cluster runs resources defined in the `prow` folder. CI Jobs that only
  require a container run in the `test-pods` namespace of this cluster.
- [*dogfooding*](dogfooding.md): Tekton runs in this cluster. This cluster is
  setup with [resources](../tekton/README.md#resources-for-cicd) from the
  `tekton` folder, plus a few [secrets](./dogfooding.md#secrets).
- [*robocat*](robocat.md): This cluster is our test bed for continuous
  deployment of services and resources. Everything that runs in this cluster is
  deployed automatically, which means it must be possible at any time to delete
  the cluster and recreate it from scratch.

## GCP projects

Automation for the `tektoncd` org runs in a GKE cluster which
[members of the governing board](https://github.com/tektoncd/community/blob/main/governance.md#permissions-and-access)
have access to.

There are several GCP projects used by Tekton:
- The GCP project that is used for GKE, storage, etc. is called
  [`tekton-releases`](http://console.cloud.google.com/home/dashboard?project=tekton-releases). It has several GKE clusters:
  - The GKE cluster that is used for [`Prow`](prow/README.md), `Tekton`, and [`boskos`](boskos/README.md) is called
    [`prow`](https://console.cloud.google.com/kubernetes/clusters/details/us-central1-a/prow?project=tekton-releases)
  - The GKE cluster that is used for nightly releases and other dogfooding is called
    [`dogfooding`](https://console.cloud.google.com/kubernetes/clusters/details/us-central1-a/dogfooding?project=tekton-releases)
- The GCP project
  [`tekton-nightly`](http://console.cloud.google.com/home/dashboard?project=tekton-nightly)
  is used to hold nightly release artifacts and [the robocat cluster](#the-robocat-cluster)

The script [adjustpermissions.py](../adjustpermissions.py) gives users access to these projects.

## DNS

DNS Names are managed via [Netlify](https://www.netlify.com/). The setup of DNS
record, for now, is manual only.
