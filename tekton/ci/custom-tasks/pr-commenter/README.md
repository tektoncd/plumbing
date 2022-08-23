# Dogfooding Job PR Commenter

This folder contains a Custom Task that can be used to comment on tektoncd PRs with information
about failing Tekton runs in the dogfooding cluster. It will add a new comment with up-to-date
information on all failing Tekton runs for the PR, pulling information about runs other than the
one triggering the task from earlier comments, and will delete the comment when updating or when
all runs have passed.

## Configuration

All configuration of the custom task is done via [environment variables on the deployment](./config/500-controller.yaml).
The `GITHUB_TOKEN` secret is the same GitHub OAuth token used in a number of other places in dogfooding.

The `RETEST_PREFIX` environment variable is there so that if we, in the future, change the command used
to re-run a Tekton job from `/test ...` to something else, we just need to change the value in the
deployment for that to be reflected in the comment.

## Example `Run`

```yaml
apiVersion: tekton.dev/v1alpha1
kind: Run
metadata:
  name: example-pr-comment
  namespace: tekton-ci
spec:
  ref:
    apiVersion: custom.tekton.dev/v0
    kind: PRCommenter
  params:
  - name: repo
    value: plumbing
  - name: prNumber
    value: 1234
  - name: sha
    value: abcd1234
  - name: jobName
    value: check-pr-has-kind-label
  - name: isSuccess
    value: "false"
  - name: isOptional
    value: "false"
  - name: logURL
    value: https://prow.tekton.dev/view/gs/tekton-prow/pr-logs/pull/tektoncd_plumbing/1185/check-pr-has-kind-label/1564708061786935296
```
