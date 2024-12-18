# Build ID Cluster Interceptor

This cluster interceptor produces a unique build ID produced through the [snowflake](https://github.com/bwmarrin/snowflake)
library. This format is compatible with the build IDs produced by [prow](https://github.com/kubernetes/test-infra/tree/master/prow),
so it can be used to let prow tooling (like deck and spyglass) discover logs
from Tekton runs.

## Cluster Interceptor Interface

`build-id` does not expect any interceptor specific input:

```json
{
  "body": "....",
  "headers": "",
}
```

It returns the payload body as an extension:

```json
{
  "continue": true,
  "extensions": {
    "build-id": {
      "id": 372779052,
    },
  },
}
```

## Example usage

A trigger in an event listener:

```yaml
- name: ci-job-trigger
  interceptors:
  - name: "Filter created PRs that contain /test"
    ref:
      name: github
    params:
    - (...)
  - name: "Add Build ID"
    ref:
      name: "build-id"
```

## Installation

The interceptor is installed via `ko`:

```bash
export KO_DOCKER_REPO=ghcr.io/tektoncd/plumbing
ko apply -P -f tekton/ci/cluster-interceptors/build-id/config/
```
