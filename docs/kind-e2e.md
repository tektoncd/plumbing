# Using `kind` for end-to-end/integration tests

Historically, our projects have used GKE clusters for end-to-end/integration testing of 
PRs. In the Pipeline project, we've discovered that using `kind`, with specially 
designated nodes in the Prow cluster, is measurably faster and more reliable. This document
is intended to help developers of Tekton projects move their tests to `kind`.

## In-repo configuration

A few changes will be needed within your project repository to keep the scripts from
creating GKE clusters and instead use `kind`. These changes can be made before you've
updated the presubmit job configuration in [the Prow configuration in the Plumbing repo](../prow/config.yaml).

### In `test/e2e-tests.sh`

For reference, [here is Pipeline's `e2e-tests.sh`](https://github.com/tektoncd/pipeline/blob/0741289847dc098f192ad9d75e040a50d9540cf6/test/e2e-tests.sh).

If you don't already have a way to skip invoking `initialize $@` via an environment
variable in your e2e script, that will need to be added. You can see at [this line](https://github.com/tektoncd/pipeline/blob/0741289847dc098f192ad9d75e040a50d9540cf6/test/e2e-tests.sh)
that the Pipeline script makes sure the environment variable `SKIP_INITIALIZE` is set to
`false` if it's not already set. We set that to `true` in the env file below.

[On these lines](https://github.com/tektoncd/pipeline/blob/0741289847dc098f192ad9d75e040a50d9540cf6/test/e2e-tests.sh#L34-L36),
we skip running `initialize $@` if `SKIP_INITIALIZE` is `true`:

```shell
if [ "${SKIP_INITIALIZE}" != "true" ]; then
  initialize $@
fi
```

`initialize` is what actually provisions a GKE cluster, so we don't want to run that when
we're using `kind`.

### In the env file

Finally, we need to make sure the environment is configured properly for running in `kind`.
We do this in Pipeline with files such as [this one](https://github.com/tektoncd/pipeline/blob/0741289847dc098f192ad9d75e040a50d9540cf6/test/e2e-tests-kind-prow.env),
which configures various environment variables for a specific job. There are two lines
here that are required for running in `kind` to work:

```
SKIP_INITIALIZE=true
KO_DOCKER_REPO=registry.local:5000
```

As mentioned above, `SKIP_INITIALIZE` is needed to prevent `e2e-tests.sh` from creating a
GKE cluster. `KO_DOCKER_REPO` needs to be set to `registry.local:5000`, because we're not
publishing the images built with `ko` to GCR, and we don't have credentials to do so.
Instead, the `kind-e2e` script spins up a local registry container, and we'll push to that.

Create a file in your repository, generally in `./test/`, with a name like `e2e-kind.env`.
Any additional environment variables you want to set for your tests can be added to the 
file as well.

## Prow configuration

Once you've made the needed updates in your project repository,  you'll need to 
reconfigure the relevant Prow job to something like [what's used for `pull-tekton-pipeline-integration-tests`](https://github.com/tektoncd/plumbing/blob/2666d73c8397c3d5b62805242886f4fa35373723/prow/config.yaml#L1229-L1283).

In [../prow/config.yaml](../prow/config.yaml), search for `name: [name of your CI job]`.
Changes will be needed on the job's configuration for `kind` to work and be used, as well
as ensuring that the job's pods will use the correct resources.

The `labels` on the job need to look like this:

```yaml
    labels:
      preset-presubmit-sh: "true"
      preset-dind-enabled: "true"
      preset-kind-volume-mounts: "true"
```

This ensures that Docker-in-Docker is enabled for the resulting pod, and that additional 
volumes required for `kind` are mounted.

Next, in the `spec:`, you will need to add these entries:

```yaml
      nodeSelector:
        cloud.google.com/gke-ephemeral-storage-local-ssd: "true"
        cloud.google.com/gke-nodepool: n2-standard-4-kind
      tolerations:
        - key: kind-only
          operator: Equal
          value: "true"
          effect: NoSchedule
```

This ensures that specific nodes are used for your `kind` job. These nodes have 4 CPUs 
and 16gb RAM, with local SSD-backed ephemeral storage. We have determined that this
configuration is the optimal one for `kind`. 

The image used in your container should be `gcr.io/tekton-releases/dogfooding/test-runner:v20220812-35d6c29808@sha256:b9010d2fe3d1da99c1735ad291271ca8ed56c8e0f3a16d4b0de7a1096fcf5a08`
or newer. You can use `:latest`, but we'd recommend against that.

The `args` for your job's container should look like this:

```yaml
  args:
    - "--service-account=/etc/test-account/service-account.json"
    - "--" # end bootstrap args, scenario args below
    - "--" # end kubernetes_execute_bazel flags (consider following flags as text)
    - "/usr/local/bin/kind-e2e"
    - "--k8s-version"
    - "v1.22.x"
    - "--nodes"
    - "3"
    - "--e2e-script"
    - "./test/e2e-tests.sh"
    - --e2e-env
    - "./test/e2e-tests-kind-prow.env"
```

The specific values you will need to pay attention to are:
* `--k8s-version`: This specifies the Kubernetes version used in the `kind` cluster. 
* `--e2e-script`: The path in your repo to the command which actually runs your tests.
    This will generally be `./test/e2e-tests.sh`, as [discussed above](#in-teste2e-testssh), 
    but if you're already using a non-standard location, use that here too.
* `--e2e-env`: The path in your repo to a file containing `FOO=bar` environment variable
    definitions to use when running `--e2e-script`. This should be the path to [the env file described above](#in-the-env-file).

Privileged mode is required to run Docker-in-Docker, add the following to your job's container:

```yaml
  securityContext:
    privileged: true
```

Finally, set the `resources` for the job's container to:
```yaml
  resources:
    requests:
      cpu: 3500m
      memory: 4Gi
    limits:
      cpu: 3500m
      memory: 8Gi
```

Please use these specific values. We want to ensure that no more than one `kind` job runs
on a single node at a time, and that we don't swamp the node's 4 CPUs.

Once you have configured the job as described above, please open a pull request to 
[the Plumbing repository](https://github.com/tektoncd/plumbing), and mention `@tektoncd/plumbing-maintainers`.
We'll review your change and merge it promptly.

After your PR has been merged, it could take up to an hour for the Prow configuration to be updated.

# Conclusion

This should be all the changes you need to make to change your e2e jobs from running 
against GKE clusters to running against local `kind` clusters. You should see some
improvement in runtime and reliability from this change, and the project as a whole saves
money, since a single `n2-standard-4` ends up being cheaper than the small GKE clusters
we have traditionally used.

If you have any questions or problems, please ask them in the `#plumbing` channel [in the tektoncd slack](https://github.com/tektoncd/community/blob/main/contact.md#slack).
