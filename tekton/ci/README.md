# Tekton CI with Tekton

This folder includes all the resources that are used to setup and run CI for
Tekton using Tekton. A general [concept](docs/ci-concept.md) is available
with diagrams to explain the general setup.

The CI system provides the following facilities:
- Run Tekton tasks or pipelines as GitHub checks, in response to pull request
  and specific comments
- Update the status of the check on GitHub when a job is started and when it
  completes
- Filter the jobs to be executed based on the content of the PR


## Where are the CI services running

All the resources used for CI are deployed in the `tektonci` namespace in the
`dogfooding` cluster.


## Setting up the response to pull requests and comments

The resources in `eventlistener.yaml` and `ingress.yaml` set up the service and
ingress that are configured in repository specific webhook settings on GitHub.
The Event Listener uses a secret called `ci-webhook`:

```
apiVersion: v1
data:
  secret: [Base64 encoded secret specified when creating the WebHook in GitHub]
kind: Secret
metadata:
  name: ci-webhook
  namespace: tektonci
type: Opaque
```

A domain name `webhook-draft.ci.dogfooding.tekton.dev` is configured manually
in Netifly to point to the public IP from the ingress.
The ingress in annotated so that cert-manager automatically obtains a
certificate from Let's Encrypt and configures HTTPs termination on the load
balancer.

The configuration on GitHub side is manual. Two events are required:
- pull_request
- issue_comment
if more events are added to the webhook, they will be filtered out by the
GitHub interceptor.

Triggers in the `eventlistener.yaml` that are not filtered by project apply to
all project with the webhook registered. They can be used to jobs to be applied
across the org.

The project specific template defines which CI Jobs are available for execution.
For `tektoncd/plumbing` they are defined in `plumbing-template.yaml`.
Eventually this template needs to be split up in reusable parts and project
specific ones, so to reduce boiler-plate code and simplify the configuration of
CI jobs.

The Tasks and Pipelines used to define the CI Job must be available in the
`tektonci` namespace. There is no facility yet to avoid name conflicts, so
projects should namespace job names by including the project name in the check
names.

The comment trigger requires a custom interceptor
[add-pr-body](./interceptors/add-pr-body/README.md) to enrich the event with the
details of the pull request where the comment was made.


## Setting up the update of the status check

The resources in `github-eventlistener.yaml` define two Event Listeners:
`github-check-start` is used to reset the status of a check, and
`github-check-done` is used to mark the status of a check completed.

The two Event Listener use different bindings to inject start/end information
in the shared Trigger Template defined in `github-template.yaml`.

For the GitHub update to work, the first task in the CI pipeline must use a
CloudEvent PipelineResource that points to the `el-github-check-start.tektonci`
service. The task that executes the test in the CI pipeline must use a
CloudEvent PipelineResource that points to the `el-github-check-done.tektonci`
service.

This is true until we will be able to trigger cloud events without the need of
a CloudEvent PipelineResource.

Example pipeline:

```
apiVersion: tekton.dev/v1alpha1
kind: Pipeline
metadata:
  name: tekton-noop-check
  namespace: tektonci
spec:
  params:
    - name: passorfail
      description: Should the CI Job 'pass' or 'fail'
    - name: message
      description: Message to be logged in the CI Job
    - name: gitCloneDepth
      description: Number of commits in the change + 1
    - name: fileFilterRegex
      description: Names regex to be matched in the list of modified files
  resources:
    - name: source
      type: git
    - name: starttrigger
      type: cloudEvent
    - name: endtrigger
      type: cloudEvent
  tasks:
    - name: check-reset
      taskRef:
        name: tekton-ci-reset-check-status
      params:
        - name: checkName
          value: tekton-noop-check
      resources:
        outputs:
          - name: trigger
            resource: starttrigger
      conditions:
      - conditionRef: "check-git-files-changed"
        params:
          - name: gitCloneDepth
            value: $(params.gitCloneDepth)
          - name: regex
            value: $(params.fileFilterRegex)
        resources:
          - name: source
            resource: source
    - name: ci-job
      taskRef:
        name: tekton-noop
      runAfter: [check-reset]
      params:
        - name: message
          value: $(params.message)
        - name: passorfail
          value: $(params.passorfail)
      resources:
        inputs:
          - name: source
            resource: source
        outputs:
          - name: endtrigger
            resource: endtrigger
```

## Filter the jobs to be executed based on the content of the PR

[TBD]


## Define new CI Jobs

[TBD]
