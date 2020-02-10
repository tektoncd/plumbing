# Clusters

The infra system relies on several different kubernetes clusters, three are
static and the rest are dynamic (provisioned on demand).

- [*prow*](../prow/README.md): Prow, Boskos and Tekton run in this cluster.
  This cluster runs resources defined in the `prow` folder. CI Jobs that only
  require a container run in the `test-pods` namespace of this cluster.
- [*dogfooding*](./dogfooding.md): Tekton runs in this cluster. This cluster is
  setup with [resources](../tekton/README.md#resources-for-cicd) from the
  `tekton` folder, plus a few [secrets](./dogfooding.md#secrets).
- [*robocat*](./robocat.md): This cluster is our test bed for continuous
  deployment of services and resources. Everything that runs in this cluster is
  deployed automatically, which means it must be possible at any time to delete
  the cluster and recreate it from scratch. 

# DNS

DNS Names are managed via [Netlify](https://www.netlify.com/). The setup of DNS
record, for now, is manual only.
