# Plumbing

This repo holds configuration for infrastructure used across the tektoncd org 🏗️:

- Automation runs [in the tektoncd GCP projects](#gcp-projects), including clusters such as dogfooding and
  [robocat](robocat/)
- The script [addpermissions.py](addpermissions.py) gives users access to
  [the GCP projects](#gcp-projects)
- [Prow](prow/README.md) is used for
  [pull request automation]((https://github.com/tektoncd/community/blob/master/process.md#reviews))
- [Tekton](tekton/README.md) is used to release projects, build docker images and run periodic jobs
- [Ingress](prow/README.md#ingress) configuration for access via `tekton.dev`
- [Gubernator](gubernator/README.md) is used for holding and displaying [Prow](prow/README.md) logs
- [Boskos](boskos/README.md) is used to control a pool of GCP projects which end to end tests can run against
- [Peribolos](tekton/resources/org-permissions/README.md) is used to control org and repo permissions

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
  is used to hold nightly release artifacts and [the robocat cluster](robocat/)

## Support

If you need support, reach out [in the tektoncd slack](https://github.com/tektoncd/community/blob/master/contact.md#slack)
via the `#plumbing` channel.

[Members of the Tekton governing board](goverance.md)
[have access to the underlying resources](https://github.com/tektoncd/community/blob/master/governance.md#permissions-and-access).
