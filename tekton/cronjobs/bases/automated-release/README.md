# Automated Release Cron Scanner

This CronJob scans tektoncd repositories for release branches and triggers validation pipelines when new commits are detected.

## How It Works

The CronJob runs weekly (Thursday 10:00 UTC) with 4 stages:

### 1. Scan Branches (init container)
- Uses `git ls-remote` to find all `release-v*` branches
- Configured repositories via `REPOSITORIES` env var
- Detects next bugfix version number:
  - Queries existing tags for each branch (e.g., `v1.10.*`)
  - Increments patch version (v1.10.5 → v1.10.6)
  - Defaults to `.0` if no tags exist (v1.10.0)
- Outputs: `/shared/branches.txt` (format: `repo:branch:commit:project:version`)

### 2. Filter New Commits (init container)
- Compares found commits against previously processed ones
- TODO: Implement persistent tracking (ConfigMap/PVC)
- Currently processes all branches (for testing)
- Outputs: `/shared/new_commits.txt`

### 3. Generate UUIDs (init container)
- Creates unique build ID for each trigger
- Uses Python's uuid module
- Outputs: `/shared/triggers.txt`

### 4. Trigger Releases (main container)
- POSTs JSON payload to EventListener
- Uses `validation-only` mode (non-intrusive)
- Enforces `bugfix` release type (only patch version increments)
- Includes detected version number in payload
- Expects HTTP 202/201 response
- Configurable via `EVENTLISTENER_URL` env var

## Configuration

### Environment Variables

- `REPOSITORIES`: Space-separated list of repos (default: `tektoncd/pipeline`)
- `EVENTLISTENER_URL`: EventListener endpoint (default: cluster-local service)

### Schedule

Default: `0 10 * * 4` (Thursday 10:00 UTC)

Modify via kustomize patch:
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: scan-release-branches
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
```

## Deployment

### Base
```bash
kubectl apply -k tekton/cronjobs/bases/automated-release/
```

### With Overlays (Recommended)
```bash
kubectl apply -k tekton/cronjobs/dogfooding/automated-release/
```

## Testing

### Manual Trigger
```bash
kubectl create job --from=cronjob/scan-release-branches manual-scan-001 -n automated-releases
```

### Check Logs
```bash
# Init containers
kubectl logs -n automated-releases job/manual-scan-001 -c scan-branches
kubectl logs -n automated-releases job/manual-scan-001 -c filter-new-commits
kubectl logs -n automated-releases job/manual-scan-001 -c generate-uuids

# Main container
kubectl logs -n automated-releases job/manual-scan-001 -c trigger-releases
```

## Persistent Tracking (TODO)

Currently, the scanner processes all branches on each run. For production:

### Option 1: ConfigMap
Store processed commits in a ConfigMap:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: release-scanner-state
data:
  tektoncd-pipeline-release-v0.50.x: "abc123def456"
  tektoncd-pipeline-release-v0.51.x: "def456ghi789"
```

### Option 2: PersistentVolumeClaim
Mount a PVC and store state in `/state/processed_commits.txt`

### Option 3: External Service
Query an external API or database for state

## Integration

Works with:
- **EventListener**: `tekton-releases` (receives triggers)
- **TriggerBinding**: `release-branch-cron` (parses payload)
- **TriggerTemplate**: `release-automation` (creates PipelineRun)

## Monitoring

Check CronJob status:
```bash
kubectl get cronjob scan-release-branches -n automated-releases
kubectl get jobs -n automated-releases -l app.kubernetes.io/component=cron-scanner
```

## Troubleshooting

### No branches found
- Check `REPOSITORIES` env var
- Verify network access to github.com
- Check git ls-remote works: `git ls-remote --heads https://github.com/tektoncd/pipeline`

### EventListener not responding
- Verify EventListener is running: `kubectl get el -n automated-releases`
- Check service exists: `kubectl get svc el-tekton-releases -n automated-releases`
- Test manually: `curl http://el-tekton-releases.automated-releases.svc.cluster.local:8080`

### All branches triggering every week
- Implement persistent state tracking (see above)
- Or accept this behavior for weekly validation runs
