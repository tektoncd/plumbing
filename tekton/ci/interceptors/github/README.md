# GitHub Simple Interceptor

This interceptor aims to provide simplified configuration for GitHub repository
triggers. It is heavily motivated by the existing
[add-pr-body](https://github.com/tektoncd/plumbing/tree/main/tekton/ci/interceptors/add-pr-body)
and
[add-team-members](https://github.com/tektoncd/plumbing/tree/main/tekton/ci/interceptors/add-team-members)
interceptors, but provides a more opinionated strategy to simplify the config
needed to be set by users.

## Configuration

This interceptor expects its configuration in an InterceptorParam named
`config`. The full config spec (including defaults) can be found at
`pkg/proto/v1alpha1/config.proto`[./pkg/proto/v1alpha1/config.proto].

### Cookbook

#### Allow all pushes, pull requests

```yaml
interceptors:
  - ref:
      name: "github-simple"
      params:
        - name: config
          value:
            push:
            pull_request:
```

This config will allow pushes to any branch or tag, pull requests to any branch.

#### Allow all, require approver sign off

```yaml
interceptors:
  - ref:
      name: "github-simple"
      params:
        - name: config
          value:
            push:
            pull_request:
              comment:
```

This config will allow pushes to any branch or tag, pull requests to any branch,
but requires pull requests to be approved by users in the `OWNERS` file in the
repo's default branch before they are ran.

#### Full Example:

```yaml
interceptors:
  - ref:
      name: "github-simple"
      params:
        - name: config
          value:
            push:
              ref: ["refs/heads/*", "refs/tags/*"]
            pull_request:
              branch: ["*"]
              comment:
                approvers:
                  path: "OWNERS"
                  revision: "main"
                match: "/ok-to-test"
```

This is the same as the previous example, but explicitly configures all the
default fields.

## Extensions

This interceptor will provide the following extension outputs that can be used in
TriggerTemplates.

These values are intended to be recommended defaults. If you wish to use
different values, simply specify the desired values in your Trigger binding.

## git

These extension values provide information on what Git source to checkout as
part of the build. This data aims to be VCS agnostic.

| key      | value                                                                                                                                                                                                                                                      |
| -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| url      | URL suitable for use with a `git clone` operation                                                                                                                                                                                                          |
| revision | Recommended Git revision to build/test against. For pushes this is the new ref SHA. For pull requests this is the revision of the pull request head (this does not provide the merge SHA, since this is not guaranteed to be populated at trigger runtime) |

## github

These extension values provide information on what GitHub
repository/installation we are operating on.

| key          | value                                                                                     |
| ------------ | ----------------------------------------------------------------------------------------- |
| owner        | GitHub Repo owner (e.g. for https://github.com/tektoncd/pipeline -> tektoncd)             |
| repo         | GitHub Repo name (e.g. for https://github.com/tektoncd/pipeline -> pipeline)              |
| installation | If the event came from a GitHub App integration, the installation ID that sent the event. |

## pull_request

For pull request related events (pull request updates, comments), the
[GitHub Pull Request API object](https://docs.github.com/en/rest/reference/pulls#get-a-pull-request)
will be embedded.
