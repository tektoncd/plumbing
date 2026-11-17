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

variable "kueue_release_signing_public_key" {
  description = "Current release signing public key from independently managed release infrastructure, in PEM PUBLIC KEY format."
  type        = string
  nullable    = false

  validation {
    condition     = can(regex("^-----BEGIN PUBLIC KEY-----\\r?\\n([A-Za-z0-9+/]+={0,2}\\r?\\n)+-----END PUBLIC KEY-----$", trimspace(var.kueue_release_signing_public_key)))
    error_message = "Provide a nonempty PEM public key with BEGIN PUBLIC KEY and END PUBLIC KEY boundaries and a base64 body."
  }
}

data "github_team" "kueue_release_maintainers" {
  slug         = "tekton-kueue-maintainers"
  summary_only = true
}

resource "github_repository_environment" "kueue_release_finalization" {
  repository          = "tekton-kueue"
  environment         = "kueue-release-finalization"
  prevent_self_review = true
  can_admins_bypass   = false

  reviewers {
    teams = [data.github_team.kueue_release_maintainers.id]
  }

  deployment_branch_policy {
    protected_branches     = false
    custom_branch_policies = true
  }
}

resource "github_repository_environment_deployment_policy" "kueue_release_finalization_main" {
  repository     = github_repository_environment.kueue_release_finalization.repository
  environment    = github_repository_environment.kueue_release_finalization.environment
  branch_pattern = "main"
}

resource "github_actions_environment_variable" "kueue_release_signing_public_key" {
  repository    = github_repository_environment.kueue_release_finalization.repository
  environment   = github_repository_environment.kueue_release_finalization.environment
  variable_name = "RELEASE_SIGNING_PUBLIC_KEY"
  value         = trimspace(var.kueue_release_signing_public_key)
}
