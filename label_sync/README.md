# label_sync

Update or migrate github labels on repos in a github org based on a YAML file

## Configuration

A typical labels.yaml file looks like:

```yaml
---
labels:
  - color: 00ff00
    name: lgtm
  - color: ff0000
    name: priority/P0
    previously:
    - color: 0000ff
      name: P0
  - name: dead-label
    color: cccccc
    deleteAfter: 2017-01-01T13:00:00Z
```

This will ensure that:

- there is a green `lgtm` label
- there is a red `priority/P0` label, and previous labels should be migrated to it:
  - if a `P0` label exists:
    - if `priority/P0` does not, modify the existing `P0` label
    - if `priority/P0` exists, `P0` labels will be deleted, `priority/P0` labels will be added
- if there is a `dead-label` label, it will be deleted after 2017-01-01T13:00:00Z

## Usage

```sh
# add or migrate labels on all repos in the tekton org
# Under kubernetes/test-infra/label_sync, run:
bazel run //label_sync -- \
  --config /path/to/labels.yaml \
  --token /path/to/github_oauth_token \
  --orgs tektoncd
```

## Cron Job

We currently run a cron job synchronize labels in

  * **Project**: tekton-release
  * **Namespace**: github-admin

We use a separate cluster in a restricted project because modifying the labels requires write permission on all repos.

We have a CronJob to sync the labels, defined
[here](https://github.com/tektoncd/plumbing/blob/master/label_sync/cluster/label_sync_job.yaml).
After making changes to `labels.yaml`, we need to update the configmap
[label-config-v2](https://github.com/tektoncd/plumbing/blob/master/label_sync/cluster/label_sync_job.yaml#L37):
```
# Setup kubectl to point to prow cluster in tekton-release
kubectl -n github-admin delete configmap label-config-v2
kubectl -n github-admin create configmap label-config-v2 --from-file=labels.yaml
```

### Create a GitHub OAuth token

Use GitHub to create an OAuth token
 
  * You need repo scope in order to modify labels on issues

```
kubectl -n github-admin create secret generic bot-token-github --from-literal=bot-token=${GITHUB_TOKEN}
```

## Create the cron job

```
kubectl -n github-admin apply -f cluster/label_sync_cron_job.yaml
```
