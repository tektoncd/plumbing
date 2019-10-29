# Tekton Plumbing Prow/PipelineRun Logging Service

## What is this?

This is a small Go app that makes it possible to publicly display the logs of
a PipelineRun kicked off by Prow. Without this it is impossible to view the
logs of a PipelineRun started by Prow without access to Tekton's Stackdriver.

We can now dogfood Tekton pipelines as part of Tekton's CI process and have
the log output of those pipelines publicly visible through PRs etc.

## How To Build This App

1. go build -o ./logview ./cmd/http

## How To Use This App

Taking the built binary from the previous section, run it as follows:

```bash
./logview -project my-project -cluster cluster-name -namespace test-pods
```

By default this will start a server bound to localhost on port 9999. To
customise the hostname or port use the `-hostname` and `-port` flags.

Run this app on a VM or in a container that has an application default
credential with permissions to read from the Stackdriver API.

Once the app is running somewhere publicly accessible, modify plank's
job_url_template to point at the public URL of the app. The app expects
the Prow Build ID to be provided as a query parameter. Example url:

```
https://app-public-address/?buildid=12345678
```

## Deploying This App To Kubernetes

You can deploy this app using `ko`. Simply run `GO111MODULE=on ko apply -f ./config` from
the 'pipelinerun-logs' directory of this repo.

## Access to Stackdriver

This app relies on access to the Stackdriver API in order to fetch log
entries for a given Prow Build ID. This requires a cluster running on GKE
with "Stackdriver Kubernetes Engine Monitoring" enabled.

### Service Account

This app's `ko` deployment is configured to use a service account named
"pipelinerun-logs-viewer" when accessing the Stackdriver API. This service
account has been added to the dogfooding cluster with a Workload Identity
that ties it to an IAM service account in the Google Cloud Console.
