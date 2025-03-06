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

This script uses [`kind`](https://kind.sigs.k8s.io/) to create a local K8s cluster in Docker and then deploys [Tekton Pipeline](https://github.com/tektoncd/pipeline), [Tekton Triggers](https://github.com/tektoncd/triggers) and [Tekton Dashboard](https://github.com/tektoncd/dashboard) components, into it.

#### Installation and prerequisites

- `go`: go 1.14+
- `kubectl`: Install the K8s CLI *(see [Install tools](https://kubernetes.io/docs/tasks/tools/))*
- `podman`: Install Podman *(see [Podman Installation Instructions](https://podman.io/docs/installation))*
- or `docker`: Install Docker *(see [Get Docker](https://docs.docker.com/get-started/get-docker/))*
- `kind`: Install `kind` *(see ["quick start" documentation](https://kind.sigs.k8s.io/docs/user/quick-start/))*

#### Usage

```sh
tekton_in_kind.sh [-c cluster-name -p pipeline-version -t triggers-version -d dashboard-version -k container-runtime]
```

> **Note**: the default `cluster-name` is `'tekton'`

#### Internals

The script, after using `kind` to create the K8s cluster, will then use `kubectl` to install the `latest` released versions of Tekton components unless other versions are specified on the optional arguments. Here is a snippet of how the script does this using `kubectl`:

```sh
# Use `-p` arg. value or `latest`
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/previous/${TEKTON_PIPELINE_VERSION}/release.yaml
# Use `-t` arg. value or `latest`
kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/previous/${TEKTON_TRIGGERS_VERSION}/release.yaml
kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/previous/${TEKTON_TRIGGERS_VERSION}/interceptors.yaml || true
# Use `-d` arg. value or `latest`
kubectl apply -f https://storage.googleapis.com/tekton-releases/dashboard/previous/${TEKTON_DASHBOARD_VERSION}/release-full.yaml
```

> **Note**: The script issues `kind cluster create` which automatically creates a K8s context named `'kind-tekton'` and makes it the current for `kubectl` commands.

> **Note**: The `kind` tool automatically updates the `current-context` for the `kubectl` command. After deleting your local cluster, `kind` unsets the `current-context` and you must manually set it again (e.g., `kubectl config use-context <context-name>`.

> **Note**: This script also builds and deploys a `kind-registry` named `registry:2` to your Podman/Docker image registry and leaves it running on port `5000`. You may manually stop it and delete the image if you do not intend to use the script again. For that see the cleanup section.

#### Cleanup

If you wish to delete the cluster that the script created, use the following command:

```sh
kind delete cluster --name tekton
```

If you have podman and docker installed and you started the cluster with the podman runtime you need to tell `kind` to use podman when deleting the cluster by setting the `KIND_EXPERIMENTAL_PROVIDER` environment variable otherwise it will default to docker and wont be able to delete it:

```sh
export KIND_EXPERIMENTAL_PROVIDER=podman 
kind delete cluster --name tekton
```

To stop and delete the registry, use the following command:

```sh
podman stop kind-registry && podman rm kind-registry
```

If you are using docker, use:

```sh
docker stop kind-registry && docker rm kind-registry
```

The `kind` tool will also use the cluster name from the `KIND_CLUSTER_NAME` environment variable if set.

```sh
export KIND_CLUSTER_NAME=tekton
kind delete cluster
Deleting cluster "tekton" ...
```

#### Troubleshooting

Podman support for `kind` is in experimental state as described [here](https://github.com/kubernetes-sigs/kind/issues/1778). If you encounter issues you should first check the `kind` [known issues](https://kind.sigs.k8s.io/docs/user/known-issues) section. The next step would be the [closed](https://github.com/kubernetes-sigs/kind/issues?q=is%3Aissue%20state%3Aclosed%20podman) and then [open](https://github.com/kubernetes-sigs/kind/issues?q=is%3Aissue%20state%3Aopen%20podman) issues of the `kind` project on Github.

##### Podman on Fedora 40

If cluster start fails when `kind` tries to join your worker nodes, check the *kubelet* logs in the worker node container:

```sh
podman exec -it tekton-worker journalctl -u kubelet
```

If you see this error:

```sh
# removed timestamp and node name for brevity
kubelet[510]: E0319 16:49:49.963674     510 manager.go:294] Registration of the raw container factory failed: inotify_init: too many open files
kubelet[510]: E0319 16:49:49.964127     510 kubelet.go:1632] "Failed to start cAdvisor" err="inotify_init: too many open files"
systemd[1]: kubelet.service: Main process exited, code=exited, status=1/FAILURE
systemd[1]: kubelet.service: Failed with result 'exit-code'.
```

Increase the resources limits for *inotify*:

```sh
sudo sysctl fs.inotify.max_user_watches=524288
sudo sysctl fs.inotify.max_user_instances=512
```

Read more [here](https://kind.sigs.k8s.io/docs/user/known-issues/#pod-errors-due-to-too-many-open-files).

---

## tekton_ci.sh

This script creates webhooks triggered by a specified GitHub repository and forwards the resulting events to your local K8s cluster running Tekton.  By default, the script assumes it is a fork of the `tektoncd/plumbing` repository.

### Usage

```sh
tekton_ci.sh -u <github-user> -t <github-token> -o <github-org> -r <github-repo>

Options:
 -u <github-user>         Your GitHub username
 -t <github-token>        Your GitHub token
 -o <github-org>          The org or user where your fork is hosted
 -r <github-repo>         The name of the fork, typically "plumbing"
```
