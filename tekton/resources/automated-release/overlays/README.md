# Automated Release System - Overlays

This directory contains kustomize overlays for per-project and per-environment configuration.

## Architecture

The automated release system uses per-project overlays following the same
pattern as the nightly-release system. Each project gets its own:

- **TriggerTemplate** — defines the PipelineRun spec with the git resolver
  pointing to the project's release pipeline in its own repository/branch
- **Trigger (webhook)** — filters GitHub push events for the specific project
- **Trigger (cron)** — filters cron scanner events for the specific project

The **EventListener** uses `namespaceSelector` to auto-discover all Triggers
in the namespace. Shared resources (bindings, tracking issue triggers, tasks,
ChatOps) live in `base/`.

## Per-Project Overlays

### pipeline/
Release automation for `tektoncd/pipeline`.
- Pipeline: `tekton/release-pipeline.yaml` (fetched via git resolver)
- Registry: ghcr.io/tektoncd/pipeline
- Status: **Active**

### triggers/ (planned)
Release automation for `tektoncd/triggers`.
- Pipeline: `tekton/release-pipeline.yaml`

### chains/ (planned)
Release automation for `tektoncd/chains`.
- Pipeline: `release/release-pipeline.yaml` (note: `release/` dir, not `tekton/`)

### operator/ (planned)
Release automation for `tektoncd/operator`.
- Pipeline: `tekton/operator-release-pipeline.yaml` (different filename)
- Extra params: `kubeDistros`, `components`

### dashboard/ (planned)
Release automation for `tektoncd/dashboard`.
- Pipeline: `tekton/release-pipeline.yaml`
- Note: Uses gcr.io default, own prerelease-checks, extra build task

## Environment Overlays

### dogfooding/
Production deployment to Oracle dogfooding cluster.
Includes `base/` and all active per-project overlays.

**Deploy**:
```bash
kubectl apply -k tekton/resources/automated-release/overlays/dogfooding
kubectl apply -k tekton/cronjobs/dogfooding/automated-release
```

## Adding a New Project

To add support for a new tektoncd project:

1. Create the overlay directory:
   ```bash
   mkdir tekton/resources/automated-release/overlays/<project>
   ```

2. Create the three files (use `pipeline/` as a template):
   - `kustomization.yaml` — references the three resource files
   - `release-template.yaml` — TriggerTemplate with git resolver + project-specific params
   - `trigger-webhook.yaml` — Trigger filtering on project name for webhooks
   - `trigger-cron.yaml` — Trigger filtering on project name for cron

3. Key things to customize per project:
   - `pipelineRef` git resolver: URL, pathInRepo
   - PipelineRun params: package, repoName, imageRegistryPath, releaseBucket
   - Secret names (if different)

4. Add the overlay to `dogfooding/kustomization.yaml`:
   ```yaml
   resources:
   - ../../base
   - ../pipeline
   - ../<project>  # Add here
   ```

5. Add the repository to the scan-release-branches CronJob:
   ```yaml
   env:
   - name: REPOSITORIES
     value: "tektoncd/pipeline tektoncd/<project>"
   ```

## Related

- [Base resources](../base/) — Shared Tekton resources
- [Nightly release overlays](../../nightly-release/overlays/) — Similar pattern
- [Issue #58](https://github.com/tektoncd/plumbing/issues/58) — Feature request
