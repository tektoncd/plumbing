# GCP projects

Automation for the `tektoncd` org runs in a GKE cluster which
[members of the governing board](https://github.com/tektoncd/community/blob/master/governance.md#permissions-and-access)
have access to.

- The GCP project the is used for GKE, storage, etc. is called
  [`tekton-releases`](http://console.cloud.google.com/home/dashboard?project=tekton-releases)
  - The GKE cluster that [`Prow`](prow/README.md), `Tekton`, and
    [`boskos`](boskos/README.md) run in is called
    [`prow`](https://console.cloud.google.com/kubernetes/clusters/details/us-central1-a/prow?project=tekton-releases)
  - The GKE cluster that is used for nightly releases and other dogfooding is called
    [`dogfooding`](https://console.cloud.google.com/kubernetes/clusters/details/us-central1-a/dogfooding?project=tekton-releases)
- The GCP project
  [`tekton-nightly`](http://console.cloud.google.com/home/dashboard?project=tekton-nightly)
  is used for GKE and to hold nightly release artifacts
  - The GKE cluster that is used to continuously deploy nightly releases is called
    [`robotcat`](https://console.cloud.google.com/kubernetes/clusters/details/europe-north1-a/robocat?project=tekton-nightly)  

This projects and clusters are used for:

- [CI automation with Prow](../prow/prow.md)
- Release automation, e.g.
  [Tekton Pipelines releases](https://github.com/tektoncd/pipeline/tree/master/tekton#release-pipeline)
