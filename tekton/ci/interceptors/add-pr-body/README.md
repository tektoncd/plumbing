# Add PR Body

`add-pr-body` is a custom interceptor for Tekton Triggers that enriches the
payload of an incoming request with the JSON representation of a pull request,
as returned by the GitHub API.

## Interface

`add-pr-body` expects the URL to the PR representation to be included in the
incoming JSON as follows:

```json
{
  "add-pr-body":
  {
    "pull-request-url": "https://api.github.com/repos/tektoncd/plumbing/pulls/225"
  },
  "other-keys": "other=values"
}
```

It returns the original JSON payload untouched, with the addition of the PR:

```json
{
  "add-pr-body":
  {
    "pull-request-url": "https://api.github.com/repos/tektoncd/plumbing/pulls/225",
    "pull-request-body":
    {
      "url": "https://api.github.com/repos/tektoncd/plumbing/pulls/225",
      "id": 372779052,
      "node_id": "MDExOlB1bGxSZXF1ZXN0MzcyNzc5MDUy",
      "html_url": "https://github.com/tektoncd/plumbing/pull/225",
      "diff_url": "https://github.com/tektoncd/plumbing/pull/225.diff",
      "patch_url": "https://github.com/tektoncd/plumbing/pull/225.patch",
      "issue_url": "https://api.github.com/repos/tektoncd/plumbing/issues/225",
      "number": 225,
      "state": "open",
      "locked": false,
      "etc": "...",
    },
  },
  "other-keys": "other=values"
}
```

HTTP Headers are left untouched.

## Example usage:

A trigger in an event listener:

```yaml
- name: comment-trigger
  interceptors:
    - github:
        secretRef:
          secretName: ci-webhook
          secretKey: secret
        eventTypes:
          - issue_comment
    - cel:
        filter: >-
          body.action == 'created' &&
          in('pull_request', body.issue) &&
          && body.issue.state == 'open' &&
          body.comment.body.matches('^/test($| [^ ]*$)')
        overlays:
        - key: add-pr-body.pull-request-url
          expression: "body.issue.pull_request.url"
```

## Installation

The interceptor is installed via `ko`:
```
export KO_DOCKER_REPO=gcr.io/tekton-releases/dogfooding
ko apply -P -f tekton/ci/interceptors/add-pr-body/config/
```

Eventually it should be included in nightly releases and installed from there.

## GitHub Enterprise

The interceptor needs authentication if you are using GitHub Enterprise. 
In order to authenticate to GitHub Enterprise API, you need to set `GITHUB_OAUTH_SECRET` environment variable.

Add GitHub OAuth secret to the deployment like below.
```yaml
    spec:
      serviceAccountName: add-pr-body-bot
      containers:
        - name: add-pr-body-interceptor
          image: github.com/tektoncd/plumbing/tekton/ci/interceptors/add-pr-body/cmd/add-pr-body
          env:
            - name: GITHUB_OAUTH_SECRET
              valueFrom:
                secretKeyRef:
                  name: github-secret
                  key: oauth
```
