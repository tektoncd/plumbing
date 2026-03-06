# Automated Release System - Deployment Guide

Quick reference for deploying the automated release system.

## Quick Start - Test Environment

```bash
# 1. Create kind cluster
kind create cluster --name tekton-test

# 2. Install Tekton
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml

# 3. Create namespace
kubectl create namespace automated-releases

# 4. Create secrets
cd tekton/resources/automated-release/overlays/test
cp secrets-template.yaml secrets.yaml
# Edit secrets.yaml with your GitHub token
kubectl apply -f secrets.yaml

# 5. Deploy system
cd ../../../../../  # Back to repo root
kubectl apply -k tekton/resources/automated-release/overlays/test
kubectl apply -k tekton/cronjobs/test/automated-release

# 6. Verify
kubectl get eventlistener,cronjob -n automated-releases
```

## Quick Start - Dogfooding Cluster

```bash
# 1. Connect to dogfooding cluster
kubectl config use-context dogfooding-cluster

# 2. Verify secrets exist
kubectl get secret -n automated-releases github-token release-secret release-images-secret

# 3. Deploy
kubectl apply -k tekton/resources/automated-release/overlays/dogfooding
kubectl apply -k tekton/cronjobs/dogfooding/automated-release

# 4. Verify
kubectl get eventlistener,cronjob -n automated-releases
kubectl get pipelinerun -n automated-releases --watch
```

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Automated Release System                  │
└─────────────────────────────────────────────────────────────┘
                              │
                ┌─────────────┼─────────────┐
                │             │             │
         ┌──────▼──────┐ ┌───▼────┐ ┌─────▼──────┐
         │   GitHub    │ │  Cron  │ │  ChatOps   │
         │  Webhooks   │ │Scanner │ │ Commands   │
         └──────┬──────┘ └───┬────┘ └─────┬──────┘
                │            │             │
                └────────────┼─────────────┘
                             │
                    ┌────────▼─────────┐
                    │  EventListener   │
                    │  tekton-releases │
                    └────────┬─────────┘
                             │
                ┌────────────┼────────────┐
                │                         │
         ┌──────▼──────┐         ┌───────▼────────┐
         │ TaskRun:    │         │  PipelineRun:  │
         │ Create/     │         │  Release       │
         │ Update      │         │  Pipeline      │
         │ Issue       │         │                │
         └─────────────┘         └────────────────┘
```

## Components Deployed

### Resources (tekton/resources/automated-release/)
- **EventListener**: `tekton-releases` - Receives webhooks and cron triggers
- **TriggerBindings**: `release-branch-webhook`, `release-branch-cron`, `release-chatops-command`
- **TriggerTemplates**: `release-automation`, `chatops-template`
- **Tasks**: `manage-release-tracking-issue`, `release-chatops-router`
- **ServiceAccount**: `release-automation` with RBAC

### CronJobs (tekton/cronjobs/)
- **scan-release-branches**: Weekly scanner (Thursday 10:00 UTC)
  - Detects release branches
  - Calculates next version number
  - Triggers validation pipelines
  - Creates/updates tracking issues

## Testing Commands

### Manual CronJob Trigger
```bash
# Create job from cronjob
kubectl create job --from=cronjob/scan-release-branches-test test-001 -n automated-releases

# Watch all container logs
kubectl logs -f -n automated-releases job/test-001 -c scan-branches
kubectl logs -f -n automated-releases job/test-001 -c filter-new-commits
kubectl logs -f -n automated-releases job/test-001 -c generate-uuids
kubectl logs -f -n automated-releases job/test-001 -c trigger-releases
```

### Manual EventListener Trigger
```bash
# Port-forward EventListener
kubectl port-forward -n automated-releases svc/el-tekton-releases 8080:8080

# Send test payload (cron format)
curl -X POST http://localhost:8080 \
  -H "Content-Type: application/json" \
  -d @- << 'EOF'
{
  "buildUUID": "test-manual-001",
  "params": {
    "release": {
      "gitRepository": "https://github.com/tektoncd/pipeline",
      "releaseBranch": "release-v0.50.x",
      "projectName": "pipeline",
      "repositoryFullName": "tektoncd/pipeline",
      "releaseMode": "validation-only",
      "releaseType": "bugfix",
      "releaseVersion": "v0.50.1",
      "commitSha": "abc123def456"
    }
  }
}
EOF
```

### Check PipelineRun Status
```bash
# Watch for new PipelineRuns
kubectl get pipelinerun -n automated-releases --watch

# Get latest PipelineRun
kubectl get pipelinerun -n automated-releases --sort-by=.metadata.creationTimestamp | tail -1

# View PipelineRun logs
tkn pipelinerun logs -n automated-releases -f <pipelinerun-name>
```

### Check GitHub Issue Creation
```bash
# Get TaskRun logs for issue creation
kubectl get taskrun -n automated-releases -l tekton.dev/task=manage-release-tracking-issue

# View logs
kubectl logs -n automated-releases -l tekton.dev/task=manage-release-tracking-issue
```

## Monitoring

```bash
# Check EventListener health
kubectl get pods -n automated-releases -l eventlistener=tekton-releases

# View EventListener logs
kubectl logs -n automated-releases -l eventlistener=tekton-releases --tail=100

# Check CronJob schedule
kubectl get cronjob -n automated-releases

# View recent jobs
kubectl get jobs -n automated-releases --sort-by=.metadata.creationTimestamp

# Monitor all releases
kubectl get pipelinerun,taskrun -n automated-releases -w
```

## Troubleshooting

### EventListener not receiving webhooks

```bash
# Check service
kubectl get svc -n automated-releases el-tekton-releases

# Check if EventListener pod is running
kubectl get pods -n automated-releases -l eventlistener=tekton-releases

# View detailed events
kubectl describe el tekton-releases -n automated-releases
```

### CronJob not running

```bash
# Check if suspended
kubectl get cronjob scan-release-branches-test -n automated-releases -o yaml | grep suspend

# Check schedule
kubectl get cronjob scan-release-branches-test -n automated-releases -o jsonpath='{.spec.schedule}'

# Check last execution
kubectl get cronjob scan-release-branches-test -n automated-releases -o jsonpath='{.status.lastScheduleTime}'
```

### Pipeline failing

```bash
# Get failed PipelineRuns
kubectl get pipelinerun -n automated-releases --field-selector status.conditions[0].status=False

# Get detailed logs
tkn pipelinerun logs <pipelinerun-name> -n automated-releases

# Check task status
kubectl get taskrun -n automated-releases
```

### Secret issues

```bash
# Verify secrets exist
kubectl get secret -n automated-releases github-token release-secret release-images-secret

# Check secret content (be careful with this!)
kubectl get secret github-token -n automated-releases -o jsonpath='{.data.token}' | base64 -d | wc -c
```

## Cleanup

### Test Environment
```bash
kubectl delete -k tekton/cronjobs/test/automated-release
kubectl delete -k tekton/resources/automated-release/overlays/test
kubectl delete namespace automated-releases
kind delete cluster --name tekton-test
```

### Dogfooding Environment
```bash
# Be very careful with this!
kubectl delete -k tekton/cronjobs/dogfooding/automated-release
kubectl delete -k tekton/resources/automated-release/overlays/dogfooding
```

## Next Steps

1. **Test locally** with kind cluster and manual triggers
2. **Deploy to dogfooding** cluster for initial validation
3. **Configure GitHub webhook** to start receiving events
4. **Monitor first runs** and verify issue creation
5. **Expand repositories** once validated (add triggers, operator, etc.)
6. **Implement Phase 5** (Slack bot) for additional ChatOps support

## Documentation

- [Base Resources README](tekton/resources/automated-release/base/README.md) - EventListener configuration
- [CronJob README](tekton/cronjobs/bases/automated-release/README.md) - Scanner details
- [Test Overlay README](tekton/resources/automated-release/overlays/test/README.md) - Local testing
- [Dogfooding Overlay README](tekton/resources/automated-release/overlays/dogfooding/README.md) - Production deployment
- [Issue #58](https://github.com/tektoncd/plumbing/issues/58) - Original requirements
