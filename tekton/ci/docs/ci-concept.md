# CI with Tekton

This document introduces a concept about how to do CI on Tekton using Tekton
itself. The same concept can be ported to other applications.

We use a dedicated Event Listener (one for each repository? to be clarified) to
server webhooks from repo for which we want to setup CI.
The Event Listener, which filters out invalid and irrelevant events using a
GitHub and a CEL filter interceptors.

The data needed by the CI pipeline is extracted from the event using an event
binding. If the different types of events are not compatible, we might use
a custom interceptor to normalise the event body to a common structure.
Environment specific inputs are passed via a dedicated binding.

The trigger template has a base structure defined in the plumbing repo.
This can be used to define common resources and to enforce CI jobs to all repos
e.g. a CLA/DCO check job. The base trigger template defines the pipeline run,
and it forces the pipeline spec to be embedded.

Since one event may trigger the execution of several CI jobs, each CI job is
associated to a Task or Pipeline. The list of Tasks (i.e. CI jobs) is defined
in the repo under test. The complete trigger template is aggregated via
"kustomize". For this to work CI tasks must have a common structure in terms
of inputs and outputs.

![CI Diagram](./ci-setup.svg)

## Reuse of Tasks in CI

It is highly desirable to be able to use existing tasks in CI, for instance
tasks from the catalog to run unit tests or linting.
The design above requires tasks to have certain features:
- never fail, report the result to a task result
- accept a fixed set of inputs

These requirements reduce reusability of tasks. Several missing features in
Tekton might help simplify the design and/or increase reuse of tasks:
- do not stop a pipeline when a task fails
- run custom steps on task start/stop
- send cloud event notifications on task completions

Until we have such features the solution are:
- write dedicated tasks
- write pipeline, to reduce the interface of existing tasks to the required one

Each task shall have the following interface:
- one input git resource
- one output cloudevent resource
