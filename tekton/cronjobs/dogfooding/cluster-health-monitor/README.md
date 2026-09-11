# Cluster Health Monitor

A Tekton Task that monitors the health of CronJobs, Jobs, PipelineRuns, and
TaskRuns in the dogfooding cluster. It runs as a CronJob that directly creates
a TaskRun (no EventListener needed), making it independent of the trigger
infrastructure it monitors.

## What it checks

### CronJob/Job checks (`check-cronjobs.sh`)

- **Stuck active jobs**: CronJobs with active jobs that never completed —
  blocks all future runs under `concurrencyPolicy=Forbid`
- **Stale CronJobs**: CronJobs that haven't succeeded in a configurable
  threshold (default: 48h) — catches silent failures
- **Image pull failures**: Pods stuck in `ImagePullBackOff`/`ErrImagePull`
- **Failed jobs**: Jobs with `status.failed > 0`

Intentionally skips suspended CronJobs (those are intentional).

### PipelineRun checks (`check-runs.sh`)

Uses smart filtering to avoid noise from flaky tests:

- **Infrastructure failures**: Always flagged regardless of rate —
  `PipelineRunTimeout`, `TaskRunImagePullFailed`, `CouldntGetTask`, etc.
  These indicate platform problems, not user code issues.
- **Consistently failing**: Pipelines where **all** of the last N runs
  failed (default: N=3). Skips pipelines with mixed success/failure.
- **Regressions**: Subset of consistently failing pipelines that
  previously had successes — flagged separately as regressions.

## How it alerts

When issues are detected, the report step creates a GitHub issue in
`tektoncd/plumbing` with structured labels (`area/infra`, `kind/monitoring`).
Deduplicates: won't create a new issue if one is already open.

## Architecture

```
CronJob (daily at 06:00 UTC)
  └── creates Job
       └── creates TaskRun (via kubectl)
            └── runs cluster-health-monitor Task
                 ├── step 1: clone plumbing repo
                 ├── step 2: check-cronjobs.sh  (kubectl)
                 ├── step 3: check-runs.sh      (kubectl)
                 └── step 4: report.sh           (creates GitHub issue)
```

The CronJob creates the TaskRun directly using `kubectl`, avoiding dependency
on EventListeners/TriggerTemplates. This means the monitor works even when the
trigger infrastructure is broken.

The Task clones the plumbing repository and runs the scripts from
`tekton/cronjobs/dogfooding/cluster-health-monitor/scripts/`. This keeps the
logic in real shell scripts that maintainers can run and test locally.

## Running locally

All scripts live in [`scripts/`](scripts/) and can be run with `kubectl`
access to the cluster:

```bash
export KUBECONFIG=~/.kube/config.tekton-oracle

# Create a report directory
mkdir -p /tmp/health-report

# Run the checks
./scripts/check-cronjobs.sh /tmp/health-report        # default: 48h stale threshold
./scripts/check-runs.sh /tmp/health-report             # default: 3-run window

# View the report (skip report.sh to avoid creating a GitHub issue)
cat /tmp/health-report/health-report.md
cat /tmp/health-report/status
```

### Script options

```bash
# Custom stale threshold (hours)
./scripts/check-cronjobs.sh /tmp/report 72

# Custom namespaces and window size
./scripts/check-runs.sh /tmp/report "default,tekton-nightly" 10
```

## RBAC

The `tekton-monitor` ServiceAccount needs:
- `get`, `list` on `cronjobs`, `jobs`, `pods` in `default` namespace
- `get`, `list` on `pipelineruns`, `taskruns` across monitored namespaces
- `create` on `taskruns` in `default` namespace (for the CronJob to create the TaskRun)
