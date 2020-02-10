# Dogfooding Cluster

The dogfooding runs the instance of Tekton that is used for all the CI/CD needs
of Tekton itself.

- `tekton` runs in [the tektoncd GCP project](./gcp.md)
- [Ingress is configured to `prow.tekton.dev`](#ingress)
- Prow results are displayed via [gubernator](../gubernator/README.md)
- [Instructions for creating the Prow cluster](#creating-the-prow-cluster)
- [Instructions for updating Prow](#updating-prow-itself) and [Prow's Tekton Pipelines instance](#tekton-pipelines-with-prow)
- [Instructions for updating Prow configuration](#updating-prow-configuration)

## Secrets

Some of the resources require secrets to operate.
- `GitHub` secrets: `bot-token-github` used for syncing label configuration and
  org configuration requires, `github-token` used to create a draft release
- `GCP` secrets: `nightly-account` is used by nightly releases to push releases
  to the nightly bucket. It's a token for service account
  `release-right-meow@tekton-releases.iam.gserviceaccount.com`.
  `release-secret` is used by Tekton Pipeline to push pipeline artifacts to a
  GCS bucket. It's also used to push images built by cron trigger (or Mario)
  to the image registry on GCP.

### Ingresses

Ingress resources use the GCP Load Balancers for HTTPS termination and offload
to kubernetes services in the `dogfooding` cluster.

SSL Certificate are generated automatically using a `ClusterIssuer` managed by
[cert-manager](https://github.com/jetstack/cert-manager/).

- To install `cert-manager` follow this
  [guide](https://docs.cert-manager.io/en/latest/getting-started/)
- To deploy the `ClusterIssuer`:
```
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
