# Add PR Body Cluster Interceptor

This folder contains an implementaion of the add-pr-body cluster interceptor that enriches the payload of an incoming request with
the JSON representation of a pull request as returned by the GitHub API.

This implementation uses the ClusterInterceptor interface. It adds the PR body under the
`extensions.add-pr-body.pull-request-body` field. For an implementation that uses the older webhook interceptor interface, see [tekton/ci/interceptors/add-pr-body](../interceptors/add-pr-body).

## Cluster Interceptor Interface

`add-pr-body` expects the URL to the PR representation to be included in the
incoming Interceptor Request as follows:

```json
{
  "body": "....",
  "headers": "",
  "extensions": {
    "add-pr-body": {
      "pull-request-url": "https://api.github.com/repos/tektoncd/plumbing/pulls/225"
    },
  },
}
```

It returns the payload body as an extension:

```json
{
  "continue": true,
  "extensions": {
    "add-pr-body": {
      "pull-request-body": {
        "url": "https://api.github.com/repos/tektoncd/plumbing/pulls/225",
        "id": 372779052,
        "node_id": "MDExOlB1bGxSZXF1ZXN0MzcyNzc5MDUy",
        "html_url": "https://github.com/tektoncd/plumbing/pull/225",
        ....
      }
    },
  },
}
```

## Example usage

A trigger in an event listener:

```yaml
- name: comment-trigger
  interceptors:
  - name: "Filter created PRs that contain /test"
    ref:
      name: cel
    params:
    - name: "filter"
      value: >- // TODO(dibyom): This can be part of the interceptor itself
          body.action == 'created' &&
          in('pull_request', body.issue) &&
          && body.issue.state == 'open' &&
          body.comment.body.matches('^/test($| [^ ]*$)')
    - name: "overlays"
      value:
      - key: add_pr_body.pull_request_url
        expression: "body.issue.pull_request.url"
  - name: "add PR body"
    ref: 
      name: "add-pr-body"
```

## Installation

The interceptor is installed via `ko`:

```bash
export KO_DOCKER_REPO=ghcr.io/tektoncd/plumbing
ko apply -P -f tekton/ci/cluster-interceptors/add-pr-body/config/
```

## GitHub Enterprise

The interceptor needs authentication if you are using GitHub Enterprise.
In order to authenticate to GitHub Enterprise API, you need to set `GITHUB_OAUTH_SECRET` environment variable.

Add GitHub OAuth secret to the deployment in `config/interceptor-deployment.yaml` like below.

```yaml
    spec:
      serviceAccountName: add-pr-body-bot
      containers:
        - name: add-pr-body-interceptor
          image: github.com/tektoncd/plumbing/tekton/ci/cluster-interceptors/add-pr-body/cmd/interceptor
          env:
            - name: GITHUB_OAUTH_SECRET
              valueFrom:
                secretKeyRef:
                  name: github-secret
                  key: oauth
```


## TODOs:
1. Add logic to filter event types i.e this interceptor should only run for PR comment type
2. Make the input a param instead of passing it via a previous CEL interceptor
3. Add support for GitHub Oauth secret via params