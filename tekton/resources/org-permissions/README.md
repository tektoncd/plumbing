## Peribolos

The peribolos configurations are automatically applied via a Github trigger on merges to master in
the community repo.

If something goes wrong and you must run the sync manually, use:

```shell
kubectl create -f tekton/resources/org-permissions/peribolos-run.yaml
```
