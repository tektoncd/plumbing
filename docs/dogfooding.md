# Dogfooding Cluster

The `dogfooding` cluster runs the instance of Tekton that is used for all the CI/CD needs
of Tekton itself.

- Configuration for the CI is in [tekton](../tekton)
- The cluster has [two node pools](#node-pools)

## Secrets

Secrets which have been applied to the `dogfooding` cluster but are not committed here are:

- `GitHub` personal access tokens:
  - In the default namespace:
    - `bot-token-github` used for syncing label configuration and org configuration
    - `github-token` used to create a draft release
  - In the `tekton-ci` namespace:
    - `bot-token-github` used for custom interceptors and CI jobs
    - `ci-webhook` contains the secret used to verify pull request webhook requests for
      plumbing CI.
  - In the [mario](../mariobot) namespace:
    - `mario-github-secret` contains the secret used to verify comment webhook requests to
      the mario service are coming from github
    - `mario-github-token` used for updating PRs
  - In the bastion-z namespace:
    - `s390x-k8s-ssh` used to ssh access s390x remote machine
  - In the bastion-p namespace:
    - `ppc64le-cluster` headless service & endpoint to resolve remote machine address
    - `ppc64le-k8s-ssh` used to ssh access ppc64le remote machine
- `GCP` secrets:
  - `nightly-account` is used by nightly releases to push releases
  to the nightly bucket. It's a token for service account
  `release-right-meow@tekton-releases.iam.gserviceaccount.com`.
  - `release-secret` is used by Tekton Pipeline to push pipeline artifacts to a
    GCS bucket. It's also used to push images built by cron trigger (or [Mario](../mariobot])
    to the image registry on GCP.
- K8s service account secrets. These secrets are used in pipeline resources of type cluster, to
  give enabled Tekton pipelines to deploy to target clusters with specific service accounts:
  - dogfooding-tekton-cd-token
  - dogfooding-tekton-cleaner-token
  - dogfooding-tektonci-default-token
  - robocat-tekton-deployer-token
  - robocat-tektoncd-cadmin-token
- Netlify API Token, in the `dns-manager` namespace, named `netlify-credentials`
- Lots of other secrets, hopefully we can add more documentation on them
  here as we go.

## Ingresses

Ingress resources use the GCP Load Balancers for HTTPS termination and offload
to kubernetes services in the `dogfooding` cluster.

SSL Certificate are generated automatically using a `ClusterIssuer` managed by
[cert-manager](https://github.com/jetstack/cert-manager/).

- To install `cert-manager` follow this
  [guide](https://docs.cert-manager.io/en/latest/getting-started/)
- To deploy the `ClusterIssuer`:

```bash
kubectl apply -f https://github.com/tektoncd/plumbing/blob/main/tekton/certificates/clusterissuer.yaml
```

The [DNS names](#dns-names) are automatically provisioned through annotations on the
ingresses themselves.

To see the IP of an ingress in the cluster:

```bash
kubectl get ingress <ingress-name>
```

A full example of an ingress with HTTPS certificate and DNS name provisioning:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    acme.cert-manager.io/http01-edit-in-place: "true"
    cert-manager.io/cluster-issuer: letsencrypt-prod
    dns.gardener.cloud/dnsnames: 'dashboard.dogfooding.tekton.dev'
    dns.gardener.cloud/ttl: "3600"
  name: ing
  namespace: tekton-pipelines
spec:
  tls:
  - secretName: dashboard-dogfooding-tekton-dev-tls
    hosts:
    - dashboard.dogfooding.tekton.dev
  rules:
  - host: dashboard.dogfooding.tekton.dev
    http:
      paths:
      - backend:
          service:
            name: tekton-dashboard
            port:
              number: 9097
        path: /*
        pathType: ImplementationSpecific
```

## Node Pools

The `dogfooding` cluster  is comprised of two node pools. One is used for workloads that operate with
Workload Identity, a feature of GKE which maps Kubernetes Service Accounts to Google Cloud IAM Service
Accounts. The other is used for workloads that don't use Workload Identity and rely instead on mechanisms
like mounted Secrets or that run unauthenticated.
Choosing the correct pool for a workload should really only depend on whether it utilizes the
Workload Identity feature or not.

- `default-pool` is used for most workloads. It doesn't have the GKE Metadata Server enabled
and therefore doesn't support workloads running with Workload Identity.
- `workload-id` has the GKE Metadata Server enabled and is used for workloads operating with
Workload Identity. The only workload that currently requires Workload Identity is "pipelinerun-logs"
which shows Stackdriver log entries for PipelineRuns.

## Manifests

Manifests for various resources are deployed to the `dogfooding` clusters from different repositories.
For the plumbing repo, manifest are applied nightly through two cronjobs:

- [tekton](https://github.com/tektoncd/plumbing/tree/main/tekton/cronjobs/dogfooding/manifests/plumbing-tekton)
- [tekton-cronjobs](https://github.com/tektoncd/plumbing/tree/main/tekton/cronjobs/dogfooding/manifests/plumbing-tekton-cronjobs)

Manifests from other repos (pipeline, dashboard and triggers) are applied manually for now.

### Service Accounts

Service accounts definitions are stored in git and are applied as part of CD, expect for the case of
[Cluster Roles](https://github.com/tektoncd/plumbing/blob/main/tekton/resources/cd/serviceaccount.yaml)
and related bindings, as they would require giving too broad access to the CD service account.

## Tekton Services

Tekton services are deployed using the [`deploy-release.sh`](https://github.com/tektoncd/plumbing/blob/main/scripts/deploy-release.sh)
script, which submits a kubernets `Job` to the `robocat` cluster, to trigger a deployment on the
`dogfooding` cluster. The `Job` triggers and event listener on the `robocat` cluster, and triggers
a Tekton task that downloads a release from the release bucket, optionally applies overlays and
deploys the result to the `dogfooding` cluster using a dedicated service account.

## DNS Names

DNS records for the `tekton.dev` are hosted by Netlify. [Gardeners External DNS Manager](https://github.com/gardener/external-dns-management)
is installed in the `dogfooding` cluster in the `dns-manager` namespace, and it watches for `DNSEntries` and annotated
ingresses and services in all namespaces.

DNS Manager is installed from the v0.11.4 tag using helm as follows:

```shell
# From a cloned https://github.com/gardener/external-dns-management
helm install dns-manager charts/external-dns-management \
  --namespace=dns-manager \
  --set configuration.disableNamespaceRestriction=true \
  --set configuration.identifier=tekton-dogfooding-default \
  --set vpa.enabled=false \
  --set createCRDs=true \
  --set resources.requests.memory=256Mi \
  --set resources.limits.memory=512Mi \
  --set 'custom.volumes:' \
  --set 'custom.volumeMounts:'
```

The DNS Provider for Netlify is installed through the following resource:

```yaml
apiVersion: dns.gardener.cloud/v1alpha1
kind: DNSProvider
metadata:
  name: netlify
  namespace: dns-manager
spec:
  type: netlify-dns
  secretRef:
    name: netlify-credentials
  domains:
    include:
    - tekton.dev
```

## Eventing

Tekton Pipelines is configured in the `dogfooding` cluster to generate `CloudEvents`
which are sent every time a `TaskRun` or `PipelineRun` is executed.
`CloudEvents` are sent by Tekton Pipelines to an event broker. `Trigger` resources
can be defined to pick-up events from broken and have them delivered to consumers.

### CloudEvents Broker

The broker installed is based on Knative Eventing running on top of a Kafka backend.
Knative Eventing is installed following the [official guide](https://knative.dev/docs/install/eventing/install-eventing-with-yaml/)
from the Knative project:

```shell
# Install the CRDs
kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v1.0.0/eventing-crds.yaml

# Install the core components
kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v1.0.0/eventing-core.yaml

# Verify the installation
kubectl get pods -n knative-eventing
```

The Kafka backend is installed, as recommended in the Knative guide, using the [strimzi](https://strimzi.io/quickstarts/) operator:

```shell
# Create the namespace
kubectl create namespace kafka

# Install in the kafka namespace
kubectl create -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka

# Apply the `Kafka` Cluster CR file
kubectl apply -f https://strimzi.io/examples/latest/kafka/kafka-persistent-single.yaml -n kafka

# Verify the installation
kubectl wait kafka/my-cluster --for=condition=Ready --timeout=300s -n kafka
```

A [Knative Channel](https://github.com/knative-sandbox/eventing-kafka) is installed next:

```shell
# Install the Kafka "Consolidated" Channel
kubectl apply -f https://storage.googleapis.com/knative-nightly/eventing-kafka/latest/channel-consolidated.yaml

# Edit the "config-kafka" config-map in the "knative-eventing" namespace
# Replace "REPLACE_WITH_CLUSTER_URL" with my-cluster-kafka-bootstrap.kafka:9092/
kubectl edit cm/config-kafka -n knative-eventing
```

Install the [Knative Kafka Broken](https://knative.dev/docs/install/eventing/install-eventing-with-yaml/#optional-install-a-broker-layer)
following the official guide:

```shell
# Kafka Controller
kubectl apply -f https://github.com/knative-sandbox/eventing-kafka-broker/releases/download/knative-v1.0.0/eventing-kafka-controller.yaml

# Kafka Broken Data plane
kubectl apply -f https://github.com/knative-sandbox/eventing-kafka-broker/releases/download/knative-v1.0.0/eventing-kafka-broker.yaml
```

### Kafka UI

The [Kafka UI](https://github.com/provectus/kafka-ui) allows viewing and searching for events stored by Kafka.
Events are retained by Kafka for some time (but not indefinitely), which helps when debugging event based integrations.
The Kafka UI allows managing channels and creating new events, so it is not publicly accessible. To access
the Kafka UI, port-forward the service port:

```shell
# Set up port forwarding
export POD_NAME=$(kubectl get pods --namespace kafka -l "app.kubernetes.io/name=kafka-ui,app.kubernetes.io/instance=kafka-ui" -o jsonpath="{.items[0].metadata.name}")
kubectl --namespace kafka port-forward $POD_NAME 8080:8080

# Point the browser to http://localhost:8080
```

The Kafka UI is installed via an helm chart as recommended in the [Kubernetes installation guide](https://github.com/provectus/kafka-ui#running-in-kubernetes).

```shell
helm install kafka-ui kafka-ui/kafka-ui --set envs.config.KAFKA_CLUSTERS_0_NAME=my-cluster --set envs.config.KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS=my-cluster-kafka-bootstrap:9092 --set envs.config.KAFKA_CLUSTERS_0_ZOOKEEPER=my-cluster-zookeeper-nodes:2181 --namespace kafka
```

### CloudEvents Producer

Tekton Pipelines is the only `CloudEvents` producer in the cluster. It's [configured](../)tekton/cd/pipeline/overlays/dogfooding/config-defaults.yaml) to send all events to the broker:

```yaml
data:
  default-cloud-events-sink: http://kafka-broker-ingress.knative-eventing.svc.cluster.local/default/default
```

### CloudEvents Consumers

`CloudEvents` are consumed from the broker via a Knative Eventing CRD called `Trigger`.
The `dogfooding` cluster is setup so that all `TaskRun` start, running and finish events are forwarded from the
broker to the `tekton-events` event listener, in the `default` namespace.
This initial filtering of events allows to reduce the load on the event listener.

The following `Triggers` are defined in the cluster:

```yaml
apiVersion: eventing.knative.dev/v1
kind: Trigger
metadata:
  name: taskrun-start-events-to-tekton-events-el
  namespace: default
spec:
  broker: default
  filter:
    attributes:
      type: dev.tekton.event.taskrun.started.v1
  subscriber:
    uri: http://el-tekton-events.default.svc.cluster.local:8080
---
apiVersion: eventing.knative.dev/v1
kind: Trigger
metadata:
  name: taskrun-running-events-to-tekton-events-el
  namespace: default
spec:
  broker: default
  filter:
    attributes:
      type: dev.tekton.event.taskrun.running.v1
  subscriber:
    uri: http://el-tekton-events.default.svc.cluster.local:8080
---
apiVersion: eventing.knative.dev/v1
kind: Trigger
metadata:
  name: taskrun-successful-events-to-tekton-events-el
  namespace: default
spec:
  broker: default
  filter:
    attributes:
      type: dev.tekton.event.taskrun.successful.v1
  subscriber:
    uri: http://el-tekton-events.default.svc.cluster.local:8080
---
apiVersion: eventing.knative.dev/v1
kind: Trigger
metadata:
  name: taskrun-failed-events-to-tekton-events-el
  namespace: default
spec:
  broker: default
  filter:
    attributes:
      type: dev.tekton.event.taskrun.failed.v1
  subscriber:
    uri: http://el-tekton-events.default.svc.cluster.local:8080
```
