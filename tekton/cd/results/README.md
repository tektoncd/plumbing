# Previewing changes

To see the generated config, run:

```sh
$ kustomize build overlays/dogfooding
```

To diff the changes against the remote cluster:

```sh
$ kustomize build overlays/dogfooding | kubectl diff -f -
```