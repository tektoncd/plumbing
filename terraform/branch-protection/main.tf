# Copyright 2026 The Tekton Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Main branch protection for all tektoncd repositories
resource "github_branch_protection" "main" {
  for_each = toset(local.tektoncd_repos)

  repository_id = each.key
  pattern       = "main"

  enforce_admins                  = var.enforce_admins
  require_signed_commits          = var.require_signed_commits
  required_linear_history         = var.required_linear_history
  allows_deletions                = var.allow_deletions
  allows_force_pushes             = var.allow_force_pushes
  require_conversation_resolution = var.require_conversation_resolution

  required_status_checks {
    strict   = false # Match existing: branch doesn't need to be up-to-date before merge
    contexts = local.merged_status_checks[each.key]
  }

  # Note: PR reviews not currently required on main branches
  # Can be enabled later by uncommenting:
  # required_pull_request_reviews {
  #   dismiss_stale_reviews           = var.dismiss_stale_reviews
  #   require_code_owner_reviews      = var.require_code_owner_reviews
  #   required_approving_review_count = var.required_approving_review_count
  # }
}

# Release branch protection (release-v*) for all tektoncd repositories
# Uses wildcard pattern to protect all release branches
resource "github_branch_protection" "releases" {
  for_each = toset(local.tektoncd_repos)

  repository_id = each.key
  pattern       = "release-v*"

  # Allow more flexibility for release managers
  enforce_admins          = false
  require_signed_commits  = var.require_signed_commits
  required_linear_history = var.required_linear_history
  allows_deletions        = false
  allows_force_pushes     = false # Keep false for safety

  required_status_checks {
    strict = true
    # Use subset of checks for release branches (build + test minimum, or unified CI summary)
    contexts = concat(
      local.base_status_checks,
      [for check in lookup(local.repo_specific_checks, each.key, []) :
        check if can(regex("^(build|test|lint|CI summary)$", check))
      ]
    )
  }

  required_pull_request_reviews {
    dismiss_stale_reviews           = var.dismiss_stale_reviews
    require_code_owner_reviews      = var.require_code_owner_reviews
    required_approving_review_count = var.required_approving_review_count_releases
  }
}
