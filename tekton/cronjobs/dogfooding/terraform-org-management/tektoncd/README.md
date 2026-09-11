## Terraform Org Management CronJob

Daily CronJob that triggers Terraform to synchronize the tektoncd GitHub
org configuration (membership, teams, team→repo permissions) from
`org/org.yaml` in `tektoncd/community`.

This replaces the peribolos CronJob that previously performed this function.

### How it works

The CronJob sends a POST to the `tekton-cd` EventListener which triggers
the `terraform-org-management-sync` Pipeline. The pipeline:

1. Clones `tektoncd/community` (for `org/org.yaml`)
2. Clones `tektoncd/plumbing` (for Terraform configuration)
3. Runs `terraform plan` / `terraform apply`

### Manual trigger

```bash
kubectl create job --from=cronjob/terraform-org-management-trigger manual-org-mgmt-$(date +%s)
```
