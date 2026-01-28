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

# Repository-specific status check contexts
variable "repo_status_checks" {
  description = "Map of repository names to their required status check contexts"
  type = map(object({
    contexts = list(string)
  }))
  default = {}
}

# Common settings
variable "enforce_admins" {
  description = "Enforce branch protection for admins"
  type        = bool
  default     = false
}

variable "require_signed_commits" {
  description = "Require signed commits"
  type        = bool
  default     = false
}

variable "required_linear_history" {
  description = "Require linear history (no merge commits)"
  type        = bool
  default     = false
}

variable "allow_force_pushes" {
  description = "Allow force pushes"
  type        = bool
  default     = false
}

variable "allow_deletions" {
  description = "Allow branch deletions"
  type        = bool
  default     = false
}

variable "required_approving_review_count" {
  description = "Number of required approving reviews for main branch"
  type        = number
  default     = 2
}

variable "required_approving_review_count_releases" {
  description = "Number of required approving reviews for release branches"
  type        = number
  default     = 1
}

variable "dismiss_stale_reviews" {
  description = "Dismiss stale pull request approvals when new commits are pushed"
  type        = bool
  default     = true
}

variable "require_code_owner_reviews" {
  description = "Require review from code owners"
  type        = bool
  default     = false
}

variable "require_conversation_resolution" {
  description = "Require all conversations to be resolved before merging"
  type        = bool
  default     = false
}
