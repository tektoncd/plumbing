# Dogfooding Job PR Status Updater

This folder contains a Custom Task that can be used to update check statuses on tektoncd PRs. It replaces the
`github-set-status` catalog task, and should be more efficient.

## Configuration

All configuration of the custom task is done via [environment variables on the deployment](./config/500-controller.yaml).
The `GITHUB_TOKEN` secret is the same GitHub OAuth token used in a number of other places in dogfooding.

## Example `Run`

```yaml
apiVersion: tekton.dev/v1beta1
kind: CustomRun
metadata:
  name: example-pr-status-update
  namespace: tekton-ci
spec:
  ref:
    apiVersion: custom.tekton.dev/v0
    kind: PRStatusUpdater
  params:
  - name: repo
    value: plumbing
  - name: prNumber
    value: 1234
  - name: sha
    value: abcd1234
  - name: jobName
    value: check-pr-has-kind-label
  - name: state
    value: pending
  - name: targetURL
    value: https://tekton.infra.tekton.dev/#/namespaces/tekton-ci/pipelineruns/a7afce10-a0a5-check-pr-has-kind-label-failure
```
