# Tekton CI with Tekton

This folder includes all the resources that are used to setup and run CI for
Tekton using Tekton. A general [concept](docs/ci-concept.md) is available
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

The CRDs in `eventlistener.yaml` and `ingress.yaml` set up the service and
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

A domain name `webhook-draft.ci.dogfooding.tekton.dev` is configured manually
in Netlify to point to the public IP from the ingress.
The ingress in annotated so that cert-manager automatically obtains a
certificate from Let's Encrypt and configures HTTPs termination on the load
balancer.

The configuration on GitHub side is manual. Two events are required:

- pull_request
- issue_comment

If more events are added to the webhook, they will be filtered out by the
GitHub interceptor.

Triggers in the `eventlistener.yaml` that are not filtered by project apply to
all project with the webhook registered. They can be used to jobs to be applied
across the org. Templates that apply to all repos are defined in `all-template.yaml`.

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
via the trigger templates. This interface is maintained consistently across CI jobs and
trigger templates:

Parameter Name    | Description                | Source                     | Notes
------------------|----------------------------|----------------------------|--------------------------------------------
buildUUID         | Unique GitHub Event ID     | `X-GitHub-Delivery` header | base binding
package           | GitHub org/repo            | repository.full_name       | base binding
gitRepository     | GitHub repo HTML URL       | repository.html_url        | base binding
gitRevision       | Git rev of the HEAD commit | pull_request.head.sha      | Added by add_pr_body for comments
gitCloneDepth     | Number of commits + 1      | extensions.git_clone_depth | Added by an overlay
pullRequestNumber | Pull request number        | pull_request.number        | Added by add_pr_body or overlay for comments
pullRequestURL    | Pull request HTML URL      | pull_request.html_url      |
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
- `pull-kind-label`

### Tasks

Tasks should be from the catalog when possible. Non catalog tasks shall be stored
in the `tektoncd/plumbing` repo under `tekton/ci/jobs` if they are applicable
across repos, or in the `tekton` folder of the specific repository.

*NOTE* Configuration from the plumbing repo is deployed automatically to the `dogfooding`
cluster. Tasks from the catalog and from other repos must be deployed and updated
manually for now.

### Pipeline

Common CI pipelines shall be stored in the `tektoncd/plumbing` repo under
`tekton/ci/jobs`, ideally one YAML file per pipeline.

We are in the process of migrating pipelines to use
[when expressions](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#guard-task-execution-using-when-expressions)
instead of [the deprecated Conditions feature](https://github.com/tektoncd/pipeline/blob/main/docs/pipelines.md#guard-task-execution-using-conditions)
but this is a work in progres and some pipelines still use `Conditions`.

```yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: [PIPELINE NAME]
  namespace: tektonci
spec:
  params:
    - name: gitRepository
      description: The git repository that hosts context and Dockerfile
    - name: gitRevision
      description: The Git revision to be used.
    - name: checkName
      description: The name of the GitHub check that this pipeline is used for
    - name: gitCloneDepth
      description: Number of commits in the change + 1
    - name: fileFilterRegex
      description: Names regex to be matched in the list of modified files
    - name: gitHubCommand
      description: The command that was used to trigger testing
    # Additional parameters may be added here as required, as long as the
    # pipeline run will be in a position to supply them based on the CI interface
  workspaces:
    - name: source
  tasks:
  # The git-clone task should run first in order to clone the source needed for other Tasks.
  - name: git-clone
    taskRef:
      name: git-clone
    params:
      - name: url
        value: $(params.gitRepository)
      - name: revision
        value: $(params.gitRevision)
      - name: depth
        value: $(params.gitCloneDepth)
    workspaces:
      - name: output
        workspace: source
  - name: extract-check-from-command
    taskRef:
      name: extract-check-from-command
    params:
      - name: gitHubCommand
        value: $(params.gitHubCommand)
  - name: check-git-files-changed
    runAfter: [git-clone] # expects source to be populated by git-clone (TEP-0063)
    taskRef:
      name: check-git-files-changed
    params:
      - name: gitCloneDepth
        value: $(params.gitCloneDepth)
      - name: regex
        value: $(params.fileFilterRegex)
    workspaces:
      - name: source
        workspace: source
  - name: [CI JOB specific name]
    runAfter: [git-clone] # expects source to be populated by git-clone (TEP-0063)
    when:
      - input: "$(tasks.check-git-files-changed.results.numFilesChanged)"
        operator: notin
        values: ["0"]
      - input: "$(tasks.extract-check-from-command.results.check)"
        operator: in
        values:
          - "$(params.checkName)" # When it is explicitly run
          - ".*"                  # When all checks are run
          - ""                    # In cases triggered not by specific github comments
    taskRef:
      name: [CI JOB Task Ref]
    workspaces:
      - name: source
        workspace: source
```

In case the CI Job is made of multiple tasks, all should run after an initial
one to which conditions are applied.

The `when` expression that uses the results of `extract-check-from-command`
is required for the CI job to executed on demand via the `/test [regex]` command.
The expression that uses the results of `check-git-files-changed` is optional,
it is used to only execute the CI job when relevant files have been modified.

### PipelineRun

The `PipelineRun` is added to the relevant `TriggerTemplate` in the
`tektoncd/plumbing` repo under `tekton/ci/templates`. The `all-template.yaml`
is used for jobs that are shared across repos. The `REPO-template.yaml`
ones are used for specific repos.

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
        - name: gitRepository
          value: $(tt.params.gitRepository)
        - name: gitRevision
          value: $(tt.params.gitRevision)
        - name: checkName
          value: CHECK-NAME # *MUST* be the GitHub check name e.g. `plumbing-yamllint`
        - name: pullRequestNumber
          value: $(tt.params.pullRequestNumber)
        - name: gitCloneDepth
          value: $(tt.params.gitCloneDepth)
        - name: fileFilterRegex
          value: "some/relevant/path/**" # A RegExp to match all relevant files, see the check-git-files-changed to see how this is used
        - name: gitHubCommand
          value: $(tt.params.gitHubCommand)
        # Extra parameters required  by the pipeline shall be passed here
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
