# Tekton buildcaptain

This folder holds the Slack buildcaptain bot code and configuration.

* [Build Captain Runbook](https://docs.google.com/document/d/1QJV0z2bMXdz_BZOkBwfxIP1BiktUb8c1lcifwqxF5wg/edit)
* [Build Captain Log](https://docs.google.com/document/d/1kUzH8SV4coOabXLntPA1QI01lbad3Y1wP5BVyh4qzmk/edit#)

Current build captains are:
- Andrea Frittoli @afrittoli (andrea.frittoli)
- Dibyo Mukherjee @dibyom (dibyo)
- Nikhil Thomas @nikhil-thomas (nikthoma)
- Savita Ashture @savitaashture (sashture)
- Sharon Jerop Kipruto @jerop (jerop)
- Vincent Demeester @vdemeester (vdemeest)

Other folks who are not build captains but have build captain access:
- Piyush Garg @piyush-garg (for maintaining the Hub in dogfooding)
- Priti Desai @pritidesai (for Pipelines releases)
- Priya Wadhwa @priyawadhwa (for Chains releases)
- Alan Greene @AlanGreene (for Dashboard releases)
- Billy Lynch @wlynch (for Results / Chains releases)

Build captain access is given via https://github.com/tektoncd/infra Terraform.

## Rotation

* The rotation is stored in [rotation.csv](rotation.csv).
* Update the rotation with [generate-rotation-csv](cmd/generate-rotation-csv).

### Generate a new rotation

Here's a one-liner for generating a new rotation:

```bash
go run ./cmd/generate-rotation-csv/main.go \
  -start-date $(date +%Y-%m-%d) \
  -days 365 \
  -names andrea.frittoli,dibyo,nikthoma,sashture,jerop,vdemeest \
  > ./rotation.csv
```

## Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: buildcaptain-cfg
  namespace: default
data:
  SLACKTOKEN: …token…
  BOTID: URCPZNB37
  CHANNELID: CPY3T4YHM
```

## Deploying

When connected to [the dogfood cluster](https://github.com/tektoncd/plumbing/blob/main/README.md#gcp-projects):

```bash
# must be run from the `buildcaptain` dir or it will use the go.mod file one level up
buildcaptain$ KO_DOCKER_REPO=gcr.io/tekton-releases/buildcaptain ko --context dogfood apply -f config/deployment.yaml
```
