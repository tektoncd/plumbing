# Tekton buildcop

This folder holds the Slack buildcop bot code and configuration.

* [Build Cop Runbook](https://docs.google.com/document/d/1QJV0z2bMXdz_BZOkBwfxIP1BiktUb8c1lcifwqxF5wg/edit)
* [Build Cop Log](https://docs.google.com/document/d/1kUzH8SV4coOabXLntPA1QI01lbad3Y1wP5BVyh4qzmk/edit#)

## Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: buildcop-cfg
  namespace: default
data:
  SLACKTOKEN: …token…
  BOTID: URCPZNB37
  CHANNELID: CPY3T4YHM
```

## Deploying

When connected to [the dogfood cluster](https://github.com/tektoncd/plumbing/blob/master/README.md#gcp-projects):

```bash
# must be run from the `buildbot` dir or it will use the go.mod file one level up
buildbot$ KO_DOCKER_REPO=gcr.io/tekton-releases/buildbot ko --context dogfood apply -f config/deployment.yaml
```