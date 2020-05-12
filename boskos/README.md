# Boskos

We use Boskos to manage GCP projects which end to end tests are run against.

- It runs [in the `prow` cluster of the `tekton-releases` project](../README.md#gcp-projects), in
  the namespace `test-pods`

_[Boskos docs](https://github.com/kubernetes/test-infra/tree/master/boskos)._

## Adding a project

* Projects are created in GCP and added to the `boskos/boskos-config.yaml` file.
* Make sure the IAM account:
`prow-account@tekton-releases.iam.gserviceaccount.com` has Editor permissions.
* Make sure the GKE API is enabled. You can do this by browsing to the GKE tab in the Cloud Console.
* After editing `boskos/boskos-config.yaml`,
[apply the updated ConfigMap](https://github.com/kubernetes/test-infra/tree/master/boskos#config-update)
to the `tekton-releases` `prow` cluster.