# Previewing changes

To see the generated config, run:

```sh
$ kustomize build overlays/oci-ci-cd
```

To diff the changes against the remote cluster:

```sh
$ kustomize build overlays/oci-ci-cd | kubectl diff -f -
```