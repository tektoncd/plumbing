# Hack

This directory includes convenience scripts for tools that assist developers in creating local Kubernetes clusters (using Docker) and then deploying/configuring Tekton components into them.

## Script overview

| Script | Description |
| :-- | :-- |
| [tekton_in_kind.sh](#tekton_in_kindsh) | Stands up a K8s cluster using the [kind](https://kind.sigs.k8s.io/) tool and deploys Tekton `pipeline`, `triggers` and `dashboard` components. |
| [tekton_ci.sh](#tekton_cish) | Sets up a GitHub webhook to a fork of the `tekton/plumbing` repo. using the `smee` tool. |

See [DEVELOPMENT.md](https://github.com/tektoncd/plumbing/blob/main/DEVELOPMENT.md) for complete usage examples.

---

## Script details

### tekton_in_kind.sh

This script uses the [`kind` tool](https://kind.sigs.k8s.io/) to create a local K8s cluster in Docker and then deploys [Tekton Pipeline](https://github.com/tektoncd/pipeline), [Tekton Triggers](https://github.com/tektoncd/triggers) and [Tekton Dashboard](https://github.com/tektoncd/dashboard) components, into it.

#### Installation and prerequisites

- `go`: go 1.14+
- `kubectl`: Install the K8s CLI *(see [Install tools](https://kubernetes.io/docs/tasks/tools/))*
- `docker`: Install Docker *(see [Get Docker](https://docs.docker.com/get-docker/))*
- `kind`: Install `kind` *(see ["quick start" documentation](https://kind.sigs.k8s.io/docs/user/quick-start/))*

#### Usage

```sh
tekton_in_kind.sh [-c cluster-name -p pipeline-version -t triggers-version -d dashboard-version]
```

> **Note**: the default `cluster-name` is `'tekton'`

#### Internals

The script, after using `kind` to create the K8s cluster, will then use `kubectl` to install the `latest` released versions of Tekton components unless other versions are specified on the optional arguments. Here is a snippet of how the script does this using `kubectl`:

```bash
# Use `-p` arg. value or `latest`
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/previous/${TEKTON_PIPELINE_VERSION}/release.yaml
# Use `-t` arg. value or `latest`
kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/previous/${TEKTON_TRIGGERS_VERSION}/release.yaml
kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/previous/${TEKTON_TRIGGERS_VERSION}/interceptors.yaml || true
# Use `-d` arg. value or `latest`
kubectl apply -f https://github.com/tektoncd/dashboard/releases/download/${TEKTON_DASHBOARD_VERSION}/tekton-dashboard-release.yaml
```

> **Note**: The script issues `kind cluster create` which automatically creates a K8s context named `'kind-tekton'` and makes it the current for `kubectl` commands.

#### Cleanup

If you wish to delete the cluster that the script created, use the following command:

```shell
kind delete cluster --name tekton
```

The `kind` tool will also use the cluster name from the `KIND_CLUSTER_NAME` environment variable if set.

```shell
export KIND_CLUSTER_NAME=tekton
kind delete cluster
Deleting cluster "tekton" ...
```

> **Note**: The `kind` tool automatically updates the `current-context` for the `kubectl` command. After deleting your local cluster, `kind` unsets the `current-context` and you must manually set it again (e.g., `kubectl config use-context <context-name>`.

> **Note**: This script also builds and deploys a `kind-registry` named `registry:2` to your Docker image registry and leaves it running on port `5000`. You may manually stop it and delete the image if you do not intend to use the script again.

---

## tekton_ci.sh

This script creates webhooks triggered by a specified GitHub repository and forwards the resulting events to your local K8s cluster running Tekton.  By default, the script assumes it is a fork of the `tektoncd/plumbing` repository.

### Usage

```shell
tekton_ci.sh -u <github-user> -t <github-token> -o <github-org> -r <github-repo>

Options:
 -u <github-user>         Your GitHub username
 -t <github-token>        Your GitHub token
 -o <github-org>          The org or user where your fork is hosted
 -r <github-repo>         The name of the fork, typically "plumbing"
```
