# Nightly Tests

This folder contains resources used for Tekton nightly tests.
The tests are triggered via [cronjobs](../../cronjobs/dogfooding/nightly-tests).

## Concept and Resource Structure

The nightly tests can be executed for different Tekton components and
based on different hardware architectures. At this moment only pipelines
for `s390x` architecture are available.
Nightly e2e tests are running based on the latest (`master` or `main`)
branch and include components build step with `ko` tool.

Basic flow of the pipeline:
- install Kubernetes cluster (optional)
- build and install required components
- run e2e tests
- cleanup
- uninstall Kubernetes cluster (optional)

Separate TriggerTemplate for each Tekton component and architecture
should be specified.
TriggerTemplates are stored in parent directory and via `kustomize`
are applied to `default` namespace. Tasks for test pipelines should
be stored in the separate folder (1 folder for 1 architecture) and
will be applied to corresponding separate namespace via `kustomize`.

Since the test pipelines are component and architecture specific, triggers
are customized with the component name and architecture name in the CEL
filter, to drive cron triggers to the correct test pipeline.

```
    - cel:
        filter: >-
          'trigger-template' in body &&
           body['trigger-template'] == 'pipeline' &&
           'arch' in body.params.target &&
           body.params.target.arch == 's390x'
```

> Problems with build/support of the concrete Tekton component for
  concrete hardware architecture is out of scope for nightly tests.
  Expectation is that component is buildable for target architecture.

## Architecture Specific Setup

### IBM Z (s390x) architecture

For s390x architecture `bastion-z` namespace is used to run the tests.
It is manually precreated.

Extra setup in the namespace was also done to get access to Z hardware
to run the tests, see [TEP 20](https://github.com/tektoncd/community/blob/main/teps/0020-s390x-support.md).

`s390x-k8s-ssh` secret was manually created in the namespace to gain
ssh access to provided Z machine.

The following pipelines are available:
- pipeline `e2e` and `examples` e2e tests
- triggers `e2e` tests
- cli `e2e` tests
- operator `e2e` tests
- dashboard `e2e` tests
- catalog `e2e` tests

They are running once a day automatically. The results can be seen
at the [dogfooding dashboard](https://tekton.infra.tekton.dev/#/namespaces/bastion-z/pipelineruns).

### IBM Power Systems (ppc64le) architecture

For ppc64le architecture `bastion-p` namespace is used to run the tests. 
It is manually precreated.

`ppc64le-k8s-ssh` secret & `ppc64le-cluster` headless service & endpoint were manually created in the namespace 
to gain ssh access to provided P machine.

The following pipelines are available:
- pipeline `e2e` e2e tests
- triggers `e2e` tests
- cli `e2e` tests
- operator `e2e` tests
- dashboard `e2e` tests
- catalog `e2e` tests

They are running once a day automatically. The results can be seen
at the [dogfooding dashboard](https://tekton.infra.tekton.dev/#/namespaces/bastion-p/pipelineruns).

