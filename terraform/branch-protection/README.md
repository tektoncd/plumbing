# Terraform Branch Protection for TektonCD

Manages branch protection rules for TektonCD repositories using Terraform.

## Protected Branches

- **main**: 2 required reviews, strict status checks
- **release-v***: 1 required review, no force pushes

## Usage

```bash
export GITHUB_TOKEN="ghp_..."
terraform init
terraform plan
terraform apply
```

## Maintenance

**Add a repository:** Edit `locals.tf` → `tektoncd_repos` list

**Add status checks:** Edit `locals.tf` → `repo_specific_checks` map

**Change settings:** Override in `terraform.tfvars` or modify `variables.tf`

## Automation

Runs daily via Tekton CronJob. See `tekton/cronjobs/dogfooding/terraform-branch-protection/`.
