# CI with Tekton reviewed

This document presents an overview of how to do CI on Tekton using Tekton
itself. The same concept can be ported to other applications.

## Pipeline and Triggers resources

We use a dedicated Event Listener to serve webhooks from repo for which we
want to run CI. The Event Listener filters out invalid requests and unwanted
events using the stock GitHub interceptor.
CEL filters can then be used to select which events we want to react to by
triggering the CI pipelines.

Since not all different types of events include the same information in the
same place, trigger bindings and custom interceptors with overlays are used
to present a standard interface towards the trigger bindings.

We use multiple bindings to make them re-usable:

- A GitHub base binding extracts parameters common to all GitHub events
- GitHub event specific bindings extract other parameters

Custom interceptors are used to add more information into the payload about
the Pull Request and the user who submitted it.

Trigger templates receive parameters from the bindings and run the CI
pipelines. They are organized per repo:

- One trigger template includes all CI jobs common to all repos
- Each repo then has an own trigger template, which can be hosted in the
  repo itself.

Trigger templates may include tasks and pipelines that are not CI jobs that
may perform tasks required for the actual CI jobs to work correctly.

CI Jobs are implemented as pipelines with one or more tasks and one or more
conditions. Conditions are used to ensure only relevant CI jobs are executed:

- Only run a CI job when relevant files were modified
- Only run a CI job when the job name matches the regex requested via the
  `/test [regex]` command in GitHub

The `Pipeline` is used to adapt the input parameters available in the trigger
template to those needed by the CI `Task(s)`. It adds labels and annotations
useful for identifying CI Jobs and for GitHub update logic downstream.

![CI Diagram](./ci-setup.svg)

## GitHub check updates

Tekton for CI is configured to send cloud events notifications for all `TaskRun`
and `PipelineRun` life-cycle events. The events are sent to a dedicated
`tekton-events` event listener, which gathers and filters them, and uses them
to update checks on GitHub side.

When a CI `Task` starts, the `start` event is intercepted and the GitHub check
is set to pending, with a link to the Tekton Dashboard for developers to stream
execution logs live if they wish to.

When a CI `Task` completes, the `end` event is intercepted to the GitHub check
is set to passed or failed, depending on the outcome of the `Task`.

If a CI job is made of multiple `Tasks`, extra filtering logic is required to
select the correct events for GitHub status updates:

- Pipeline Start -> We never use this one as condition may tell us later
  that the CI job should not be executed
- Task Start -> This can be used to set the status to pending, even in case of
  multiple tasks
- Task End -> This can be used in case of single task CI jobs, or also in case
  of multiple tasks if we have annotations to tell us that it's the last one
- Pipeline End -> This can used in all cases, however the Task End is preferred
  in case of single Task CI jobs since it will trigger faster

In edge cases events may be delivered out of order (for instance in case of retries)
which could lead to the start event being processed after the end one.
Logic in the GitHub update tasks helps minimize the impact of this case.

## Reuse of Tasks in CI

It is highly desirable to be able to use existing `Tasks` in CI, for instance
`Tasks` from the catalog to run unit tests or linting.
When a suitable `Task` does not exists, it should be added to the catalog if
applicable.

The CI pipelines can be used to adapt the parameters available from the CI
system to the catalog tasks.
