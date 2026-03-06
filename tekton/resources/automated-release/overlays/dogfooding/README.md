# Dogfooding Environment Deployment

Deploy the automated release system to the Oracle dogfooding cluster.

## Prerequisites

1. **Access to dogfooding cluster**:
   ```bash
   kubectl config use-context dogfooding-cluster
   ```

2. **Namespace** (likely already exists):
   ```bash
   kubectl get namespace tekton-releases
   ```

3. **Secrets** (should already exist in dogfooding cluster):
   - `github-token` - GitHub API access for issue management
   - `release-secret` - GCS service account for releases
   - `release-images-secret` - Container registry credentials

   Verify:
   ```bash
   kubectl get secret -n tekton-releases github-token
   kubectl get secret -n tekton-releases release-secret
   kubectl get secret -n tekton-releases release-images-secret
   ```

## Deploy

```bash
# From repository root
kubectl apply -k tekton/resources/automated-release/overlays/dogfooding
kubectl apply -k tekton/cronjobs/dogfooding/automated-release
```

## Verify Deployment

```bash
# Check EventListener
kubectl get eventlistener -n tekton-releases
kubectl describe el tekton-releases -n tekton-releases

# Check CronJob
kubectl get cronjob -n tekton-releases scan-release-branches-dogfooding

# Check next scheduled run
kubectl get cronjob scan-release-branches-dogfooding -n tekton-releases -o jsonpath='{.status.lastScheduleTime}'

# Get EventListener service
kubectl get svc -n tekton-releases el-tekton-releases
```

## Configure GitHub Webhook

1. Go to https://github.com/tektoncd/pipeline/settings/hooks
2. Add webhook:
   - **Payload URL**: `http://el-tekton-releases.tekton-releases.svc.cluster.local:8080` (or public ingress URL)
   - **Content type**: `application/json`
   - **Events**: Just the `push` event
   - **Active**: ✓

3. Test delivery with a test push to a release branch

## Monitor

```bash
# Watch for triggered PipelineRuns
kubectl get pipelinerun -n tekton-releases --watch

# Check CronJob history
kubectl get jobs -n tekton-releases -l app.kubernetes.io/component=cron-scanner

# View CronJob logs
kubectl logs -n tekton-releases \
  -l app.kubernetes.io/component=cron-scanner \
  -c trigger-releases --tail=100
```

## Troubleshooting

### EventListener not receiving webhooks

```bash
# Check EventListener pod
kubectl get pods -n tekton-releases -l eventlistener=tekton-releases

# View EventListener logs
kubectl logs -n tekton-releases -l eventlistener=tekton-releases
```

### CronJob not running

```bash
# Check if suspended
kubectl get cronjob scan-release-branches-dogfooding -n tekton-releases -o jsonpath='{.spec.suspend}'

# Manually trigger for testing
kubectl create job --from=cronjob/scan-release-branches-dogfooding manual-test-001 -n tekton-releases
```

### Issue creation failing

```bash
# Check github-token secret
kubectl get secret github-token -n tekton-releases -o jsonpath='{.data.token}' | base64 -d

# Check TaskRun logs
kubectl logs -n tekton-releases -l tekton.dev/task=manage-release-tracking-issue
```

## Expanding to More Repositories

Edit the cronjob patch to add more repositories:

```yaml
# tekton/cronjobs/dogfooding/automated-release/cronjob-patch.yaml
- name: REPOSITORIES
  value: "tektoncd/pipeline tektoncd/triggers tektoncd/operator"
```

Then reapply:
```bash
kubectl apply -k tekton/cronjobs/dogfooding/automated-release
```
