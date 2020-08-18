# Add Team Members

`add-team-members` is a custom interceptor for Tekton Triggers that enriches the
payload of an incoming request with the list of public members of the org as well
as the list of maintainers for the project.

## Interface

`add-team-members` expects the URL to the PR representation to be included in the
incoming JSON as follows:

```json
{
  "add_team_members":
  {
    "org_base_url": "https://api.github.com/repos/tektoncd/",
    "team": "plumbing"
  },
  "other-keys": "other=values"
}
```

It returns the original JSON payload untouched, with the addition of the PR:

```json
{
  "add_team_members":
  {
    "org_base_url": "https://api.github.com/repos/tektoncd/",
    "team": "plumbing",
    "public_org_members": ["a", "b", "c"],
    "maintainers_team_members": ["a", "b"],
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
        - key: add_team_members.org_base_url
          expression: "body.organization.url"
    - webhook:
        objectRef:
          kind: Service
          name: add-team-member
          apiVersion: v1
          namespace: tektonci
    - cel:
        filter: >-
          body.comment.user.login in body.add_team_members.maintainers_team_members
```

## Installation

The interceptor is installed via `ko`:
```
export KO_DOCKER_REPO=gcr.io/tekton-releases/dogfooding
ko apply -P -f tekton/ci/interceptors/add-team-members/config/
```

Eventually it should be included in nightly releases and installed from there.
