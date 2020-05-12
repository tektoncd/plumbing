# Mario bot

Mario is an experimental bot that automates tasks in the plumbing repository.

It's intended to be used as a Tekton Webhook endpoint.

## Usage

Mario responds to comments on issues received via GitHub hook events.

### Triggering builds

Currently the only recognised command is `build`:

Adding a comment like this to an issue:

```
/mario build tekton/images/tkn tkn:mario
```

This will respond with a JSON body something like this:

```json
{
  "buildUUID": "81799043-9591-4b1d-bb00-ca76e1ced916",
  "gitRepository": "github.com/tektoncd/plumbing",
  "gitRevision": "pull/20/head",
  "contextPath": "tekton/images/tkn",
  "targetImage": "gcr.io/tekton-releases/dogfooding/tkn:mario",
  "pullRequestID": "20"
}
```

These fields can be picked up in a template binding:

```yaml
apiVersion: tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: mario-trigger-binding
spec:
  params:
  - name: buildUUID
    value: $(body.buildUUID)
  - name: gitRepository
    value: $(body.gitRepository)
  - name: gitRevision
    value: $(body.gitRevision)
  - name: contextPath
    value: $(body.contextPath)
  - name: targetImage
    value: $(body.targetImage)
  - name: pullRequestID
    value: $(body.pullRequestID)
```

## Triggering from an Event Listener

As this is a [Webhook interceptor](https://github.com/tektoncd/triggers/blob/master/docs/eventlisteners.md#Webhook-Interceptors) it needs to be configured in the list of interceptors for a trigger:

It does its own GitHub secret validation.

```yaml
apiVersion: tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: mario-image-builder
spec:
  serviceAccountName: mario-listener
  serviceType: LoadBalancer
  triggers:
    - name: trigger
      interceptors:
        - webhook:
            objectRef:
              kind: Service
              name: mario
              apiVersion: v1
              namespace: mario
      bindings:
        - name: mario-trigger-binding
      template:
        name: mario-trigger-template
```

## Configuring the GitHub secret

The Mario Bot requires a GitHub Hook Secret in the environment variable `GITHUB_SECRET_TOKEN`.

The bot uses [secrets in the dogfooding cluster](../README.md#dogfooding-secrets).
