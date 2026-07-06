# Terraform Branch Protection CronJob

Daily CronJob that applies branch protection rules to TektonCD repositories.

- **Schedule:** Daily at midnight UTC
- **Pipeline:** `terraform-branch-protection-sync`
- **Config:** `terraform/branch-protection/`

## Manual Trigger

```bash
tkn pipeline start terraform-branch-protection-sync \
  -p url=https://github.com/tektoncd/plumbing.git \
  -p revision=main \
  -p terraformCommand=apply \
  -w name=shared-workspace,volumeClaimTemplateFile=pvc.yaml \
  -w name=github-oauth,secret=peribolos-token-github
```

## Requirements

- Secret `peribolos-token-github` with admin permissions on tektoncd org
- EventListener `tekton-cd` running
