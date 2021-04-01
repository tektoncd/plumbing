# Tekton CI with Tekton

This folder includes all the resources that are used to setup and run CI for
Tekton using Tekton. A general [concept](../ci/docs/ci-concept.md) is available
with diagrams to explain the general setup.

The CI system provides the following facilities:

- Run Tekton tasks or pipelines as GitHub checks, in response to pull request
  events and specific comments
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

```yaml
apiVersion: v1
data:
  secret: [Base64 encoded secret specified when creating the WebHook in GitHub]
kind: Secret
metadata:
  name: ci-webhook
  namespace: tektonci
type: Opaque
```

A domain name `webhook.ci.dogfooding.tekton.dev` is configured automatically in
Netlify through an annotation on the ingress. The ingress in annotated so that
cert-manager automatically obtains a certificate from Let's Encrypt and configures 
HTTPs termination on the load balancer.

The webhook configuration on GitHub side is manual. Two events are required:

- pull_request
- issue_comment

If more events are added to the webhook, they will be filtered out by the
GitHub interceptor.

The `eventlistener.yaml` does not defined any trigger. Triggers are instead
defined as separate resources, so that each project can define its own.
There is a shared trigger that defines jobs that should run on groups of
repositories.

Each project has a dedicated folder with project specific resources:

- `trigger.yaml` which defines the triggers for the project (pull_request, comment)
- `template.yaml` which defines the CI pipelines for the project

The Tasks and Pipelines used to define the CI Job must be available in the
`tekton-ci` namespace. There is no facility yet to avoid name conflicts, so
projects should namespace job names by including the project name in the check
names.

The comment trigger requires a custom interceptor
[add-pr-body](./interceptors/add-pr-body/README.md) to enrich the event with the
details of the pull request where the comment was made.

The ability to filter events based on the user requires a custom interceptor
[add-team-members](./interceptors/add-team-members/README.md) to enrich the event
payload with details about members of the GitHub org and of the repo maintainer
team.

## Setting up the update of the status check

Tekton is deployed the `dogfooding` cluster with cloud events enabled.
All cloud events are sent to the [`tekton-events`](../resources/cd/eventlistener.yaml)
event listener. CEL filters are used to select events from CI jobs `TaskRuns`.

When a start, failed or succeeded event is received for a CI job, the
[`github-template.yaml`](../resources/ci/github-template.yaml) is triggered,
which takes care of updating the check status on GitHub side accordingly.

Conditions must trigger github updates - because of have tasks that implement
conditions must be named `check-*`, which is used in the trigger CEL filter
to skip the associated events.

The `github-template` adds labels to the task runs it triggers to make it
easier to associate them back with the source task run:

```yaml
      labels:
        prow.k8s.io/build-id: $(tt.params.buildUUID)
        ci.tekton.dev/source-taskrun-namespace: $(tt.params.taskRunNamespace)
        ci.tekton.dev/source-taskrun-name: $(tt.params.taskRunName)
```

## CI Job Interface

The existing overlays and bindings produce a set of parameters available to CI jobs
via the trigger templates. This interface is maintained consistent across CI jobs and
trigger templates:

Parameter Name    | Description                | Source                     | Notes
------------------|----------------------------|----------------------------|--------------------------------------------
buildUUID         | Unique GitHub Event ID     | `X-GitHub-Delivery` header | base binding
package           | GitHub org/repo            | repository.full_name       | base binding
gitRepository     | GitHub repo HTML URL       | repository.html_url        | base binding
gitRevision       | Git rev of the HEAD commit | pull_request.head.sha      | Added by add_pr_body for comments
gitCloneDepth     | Number of commits + 1      | extensions.git_clone_depth | Added by an overlay
pullRequestNumber | Pull request number        | pull_request.number        | Added by add_pr_body or overlay for comments
pullRequestURL    | Pull request HTML URL      | pull_request.html_url      | Added by add_pr_body for comments
pullRequestBaseRef| Pull request Base Branch   | pull_request.base.ref      | Added by add_pr_body for comments
gitHubCommand     | GitHub comment body        | comment.body               | Only available for comments, default for PR
labels            | GitHub labels for PR       | pull_request.labels        | Only available for PRs, missing for comment

## Define new CI Jobs

A new CI Job requires the following:

- a unique GitHub check name, used to identify the check on GitHub side and to
  trigger the job on demand
- one or more `Tasks` to be executed
- a `Pipeline` that maps the [CI Job Interface](#ci-job-interface) to the `Tasks`
- a `PipelineRun` to be added to a `TriggerTemplate` that runs the `Pipeline`
  with the right metadata and parameters from the event

### Check Names

The check name is build according to the following convention:

```shell
TRIGGER[-PROJECT]-TEST_NAME
```

The `TRIGGER` can be:

- `pull` for jobs executed against a pull request
- `periodic[-BRANCH]` for periodic jobs executed against a `BRANCH`
  The `BRANCH` part can be omitted if the branch is the main one
  for the repository
- `post` for jobs executed after a change is merged

The `PROJECT` part is optional, so job that are identical across repositories
shall not include it in the name.

Example of job names:

- `pull-pipeline-build-tests`
- `pull-triggers-go-coverage`
- `periodic-dashboard-integration-tests`
- `post-catalog-publish-tasks`
- `pull-check-kind-label`

### Tasks

Tasks should be from the catalog when possible. Non catalog tasks shall be stored
in the `tektoncd/plumbing` repo under `tekton/ci-workspace/jobs` if they are applicable
across repos, or in the `tekton/ci` folder of the specific repository.

*NOTE* Resources from the plumbing repo are deployed automatically to the `dogfooding`
cluster. Tasks from the catalog and from other repos must be deployed and updated
manually for now.

### Pipeline

Common CI pipelines shall be stored in the `tektoncd/plumbing` repo under
`tekton/ci-workspace/jobs`, ideally one YAML file per pipeline.

```yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: [PIPELINE NAME]
spec:
  params:
    - name: pullRequestNumber
      description: The pullRequestNumber
    - name: pullRequestBaseRef
      description: The pull request base branch
    - name: gitRepository
      description: The git repository that hosts context and Dockerfile
    - name: gitCloneDepth
      description: Number of commits in the change + 1
    - name: fileFilterRegex
      description: Names regex to be matched in the list of modified files
    - name: checkName
      description: The name of the GitHub check that this pipeline is used for
    - name: gitHubCommand
      description: The command that was used to trigger testing
    # Additional parameters may be added here as required, as long as the
    # pipeline run will be in a position to supply them based on the CI interface
  workspaces:
    - name: sources
      description: Workspace where the git repo is prepared for testing
  tasks:
    - name: clone-repo
      taskRef:
        name: git-batch-merge # from the catalog
      params:
        - name: url
          value: $(params.gitRepository)
        - name: mode
          value: "merge"
        - name: revision
          value: $(params.pullRequestBaseRef)
        - name: refspec
          value: refs/heads/$(params.pullRequestBaseRef):refs/heads/$(params.pullRequestBaseRef)
        - name: batchedRefs
          value: "refs/pull/$(params.pullRequestNumber)/head"
      workspaces:
        - name: output
          workspace: sources
    - name: check-name-matches
      taskRef:
        name: check-name-matches
      params:
        - name: gitHubCommand
          value: $(params.gitHubCommand)
        - name: checkName
          value: pull-community-teps-lint
    - name: check-git-files-changed
      runAfter: ['clone-repo']
      taskRef:
        name: check-git-files-changed
      params:
        - name: gitCloneDepth
          value: $(params.gitCloneDepth)
        - name: regex
          value: $(params.fileFilterRegex)
      workspaces:
        - name: input
          workspace: sources
    - name: [CI JOB specific name]
      when: # implicit dependency on the check tasks
        - input: $(tasks.check-name-matches.results.check)
          operator: in
          values: ["passed"]
        - input: $(tasks.check-git-files-changed.results.check)
          operator: in
          values: ["passed"]
      workspaces:
        - name: input
          workspace: sources
      taskRef:
        name: [CI JOB Task Ref]
      params:
        - # Any task specific parameter
```

In case the CI Job is made of multiple tasks, all should run after the task
that evaluate the conditions are executed.

The `check-name-matches` task is required for the CI job to
executed on demand via the `/test [regex]` command.
The `check-git-files-changed` task is optional, it is used to only execute
the CI job when relevant files have been modified.

### PipelineRun

The `PipelineRun` is added to the relevant `TriggerTemplate` in the
`tektoncd/plumbing` repo under `tekton/ci-workspace/<project>/template.yaml`.
The `shared` folder is used for jobs that are shared across repos.

The event listener will trigger the correct template based on the event.
The `PipelineRun` must define specific metadata for the conditions and the
downstream CEL filters to work correctly.

```yaml
  - apiVersion: tekton.dev/v1beta1
    kind: PipelineRun
    metadata:
      generateName: CHECK-NAME- # generateName *MUST* be used here. The name is for information only.
      labels:
        tekton.dev/kind: ci
        tekton.dev/check-name: CHECK-NAME # *MUST* be the GitHub check name
        tekton.dev/pr-number: $(tt.params.pullRequestNumber)
        prow.k8s.io/build-id: $(tt.params.buildUUID)
      annotations:
        tekton.dev/gitRevision: "$(tt.params.gitRevision)"
        tekton.dev/gitURL: "$(tt.params.gitRepository)"
    spec:
      serviceAccountName: tekton-ci-jobs
      pipelineRef:
        name: PIPELINE_NAME # The name of the CI pipeline
      params:
        - name: checkName
          value: CHECK-NAME # *MUST* be the GitHub check name
        - name: pullRequestNumber
          value: $(tt.params.pullRequestNumber)
        - name: gitCloneDepth
          value: $(tt.params.gitCloneDepth)
        - name: fileFilterRegex
          value: "some/relevant/path/**" # A RegExp to match all relevant files
          # The match is executed roughly follows:
          # git diff-tree --no-commit-id --name-only -r HEAD "$(params.gitCloneDepth) - 1" | \
          #   grep -E '$(params.fileFilterRegex)
        - name: gitHubCommand
          value: $(tt.params.gitHubCommand)
        # Extra parameters required  by the pipeline shall be passed here
      resources:
      - name: source
        resourceSpec: # Pipeline resources *MUST* be embedded
          type: git
          params:
          - name: revision
            value: $(tt.params.gitRevision)
          - name: url
            value: $(tt.params.gitRepository)
          - name: depth
            value: $(tt.params.gitCloneDepth)
```

*NOTE* The naming convention for labels and annotations may change in future
as the `tekton.dev` namespace has been reserved for Tekton itself only.

### New Trigger Template

If a `TriggerTemplate` for a specific repository does not exists yet, it must be
created under `tekton/ci/templates` and named `REPO-template.yaml`.
When a new trigger template is added, the event listener needs to be updated to
trigger the new template for the right events.

A good starting point is to look at the two triggers already defined for the
`plumbing` repo and replicate them for the new repo.
To react to pull requests:

```yaml
  triggers:
    - name: plumbing-pull-request-ci
      interceptors:
        - github:
            secretRef:
              secretName: ci-webhook
              secretKey: secret
            eventTypes:
              - pull_request
        - cel:
            filter: >-
              body.repository.full_name == 'tektoncd/plumbing' &&
              (body.action == 'opened' || body.action == 'synchronize')
            overlays:
            - key: git_clone_depth
              expression: "string(body.pull_request.commits + 1.0)"
      bindings:
        - ref: tekton-ci-github-base
        - ref: tekton-ci-webhook-pull-request
        - ref: tekton-ci-clone-depth
        - ref: tekton-ci-webhook-pr-labels
      template:
        ref: tekton-plumbing-ci-pipeline
```

To react to issue comments:

```yaml
    - name: all-comment-ci
      interceptors:
        - github:
            secretRef:
              secretName: ci-webhook
              secretKey: secret
            eventTypes:
              - issue_comment
        - cel:
            filter: >-
              body.repository.full_name.startsWith('tektoncd/') &&
              body.repository.name in ['plumbing', 'pipeline', 'triggers', 'cli', 'dashboard', 'catalog', 'hub'] &&
              body.action == 'created' &&
              'pull_request' in body.issue &&
              body.issue.state == 'open' &&
              body.comment.body.matches('^/test($| [^ ]*[ ]*$)')
            overlays:
            - key: add_pr_body.pull_request_url
              expression: "body.issue.pull_request.url"
        - webhook:
            objectRef:
              kind: Service
              name: add-pr-body
              apiVersion: v1
              namespace: tektonci
        - cel:
            overlays:
            - key: git_clone_depth
              expression: "string(body.extensions.add_pr_body.pull_request_body.commits + 1.0)"
      bindings:
        - ref: tekton-ci-github-base
        - ref: tekton-ci-webhook-comment
        - ref: tekton-ci-clone-depth
        - ref: tekton-ci-webhook-pr-labels
      template:
        ref: tekton-plumbing-ci-pipeline
```
