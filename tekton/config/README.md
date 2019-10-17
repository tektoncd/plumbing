# Tekton Deployment Config

This folder holds kustomize overlays, that can be used to deploy resources
defined in the tekton folder.

For now this is only used to maintain cron job configurations. To add a new
cron job to be deployed to the `dogfooding` cluster, create a folder and add
a kustomization.yaml into it, along with the cronjob overlay.

Example folders structure:
```
├── tekton
│   ├── config
│   |   ├── cron
│   │   │   └── cronjob.yaml
│   │   ├── pipeline-nighty-release
│   │   │   ├── kustomization.yaml
│   │   │   └── cron.yaml
│   │   ├── periodic-daily-tests
│   │   │   ├── kustomization.yaml
│   │   │   └── cron.yaml
│   │   ├── README.md
```

Kustomization configuration file:
```
# kustomization.yaml
bases:
- ../../cron
patchesStrategicMerge:
- cronjob.yaml
nameSuffix: "-pipeline-nightly-release"
```

Cronjob definition file:
```
# cronjob.yaml
apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: pipeline-cron-trigger
spec:
  schedule: "*/1 * * * *"  # <-- Change the schedule here
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: trigger
            env: SINK_URL
            value: [URL of the event-binding sink] # <-- Change the URL here
          initContainers:
          - name: git
            env: GIT_REPO
            value: [URL of the git repo - no protocol] # <-- Change the REPO here
```

To generate the YAML for a cron configuration, run the following:
```
kustomize build tekton/config/pipeline-nighty-release/
```

To apply the cron configuration directly, run the following:
```
kustomize build tekton/config/pipeline-nightly-release-cron/ | kubectl apply -f -
```
