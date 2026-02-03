# Automated Release System - Deployment Overlays

This directory contains kustomize overlays for deploying the automated release system to different environments.

## Available Overlays

### test/
Local testing environment using kind cluster.

**Use case**: Development and testing before deploying to production
**Target**: Local kind cluster
**Features**:
- EventListener exposed via NodePort
- CronJob suspended by default (manual trigger only)
- Uses fork repositories to avoid production triggers
- Includes testing documentation and secret templates

**Deploy**:
```bash
kubectl apply -k tekton/resources/automated-release/overlays/test
kubectl apply -k tekton/cronjobs/test/automated-release
```

See [test/README.md](test/README.md) for detailed setup instructions.

### dogfooding/
Production deployment to Oracle dogfooding cluster.

**Use case**: Production automated releases for Tekton projects
**Target**: Oracle dogfooding cluster
**Features**:
- Weekly cron scanner (Thursday 10:00 UTC)
- GitHub webhook integration
- ChatOps support via issue comments
- Initial scope: tektoncd/pipeline (expandable)

**Deploy**:
```bash
kubectl apply -k tekton/resources/automated-release/overlays/dogfooding
kubectl apply -k tekton/cronjobs/dogfooding/automated-release
```

See [dogfooding/README.md](dogfooding/README.md) for detailed setup instructions.

## Overlay Structure

Each overlay includes:
- `kustomization.yaml` - Main kustomize configuration
- `*-patch.yaml` - Environment-specific patches
- `README.md` - Deployment and testing instructions
- `secrets-template.yaml` (test only) - Secret templates for local testing

## Required Secrets

All environments require these secrets in the `automated-releases` namespace:

1. **github-token** - GitHub personal access token
   - Scope: `repo:write` (for issue creation/updates)
   - Used by: manage-release-tracking-issue task

2. **release-secret** - GCS service account credentials
   - Used by: release pipeline for artifact publishing
   - Format: JSON service account key

3. **release-images-secret** - Container registry credentials
   - Used by: release pipeline for image publishing
   - Format: Docker config JSON

## Testing Workflow

1. **Local testing** (test overlay):
   ```bash
   # Setup kind cluster
   kind create cluster --name tekton-test

   # Deploy Tekton
   kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
   kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml

   # Create namespace and secrets
   kubectl create namespace automated-releases
   kubectl apply -f tekton/resources/automated-release/overlays/test/secrets.yaml

   # Deploy automated release system
   kubectl apply -k tekton/resources/automated-release/overlays/test
   kubectl apply -k tekton/cronjobs/test/automated-release

   # Manual testing
   kubectl create job --from=cronjob/scan-release-branches-test test-001 -n automated-releases
   ```

2. **Production deployment** (dogfooding overlay):
   ```bash
   # Connect to dogfooding cluster
   kubectl config use-context dogfooding-cluster

   # Verify secrets exist
   kubectl get secret -n automated-releases github-token release-secret release-images-secret

   # Deploy
   kubectl apply -k tekton/resources/automated-release/overlays/dogfooding
   kubectl apply -k tekton/cronjobs/dogfooding/automated-release

   # Monitor
   kubectl get pipelinerun -n automated-releases --watch
   ```

## Customization

To create a new overlay:

1. Create overlay directory:
   ```bash
   mkdir -p tekton/resources/automated-release/overlays/my-env
   mkdir -p tekton/cronjobs/my-env/automated-release
   ```

2. Create kustomization.yaml referencing base:
   ```yaml
   resources:
   - ../../base
   ```

3. Add patches for environment-specific configuration:
   - Repository list (REPOSITORIES env var)
   - EventListener URL
   - Schedule adjustments
   - Resource limits

4. Document in README.md

## Related Documentation

- [Base resources](../base/) - Core Tekton resources
- [CronJob base](../../../cronjobs/bases/automated-release/) - Cron scanner configuration
- [Issue #58](https://github.com/tektoncd/plumbing/issues/58) - Original feature request
