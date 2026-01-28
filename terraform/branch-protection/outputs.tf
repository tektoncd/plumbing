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

output "protected_repos" {
  description = "List of repositories with branch protection applied"
  value       = local.tektoncd_repos
}

output "main_branch_protections" {
  description = "Main branch protection resource IDs"
  value = {
    for repo in local.tektoncd_repos :
    repo => github_branch_protection.main[repo].id
  }
}

output "release_branch_protections" {
  description = "Release branch protection resource IDs"
  value = {
    for repo in local.tektoncd_repos :
    repo => github_branch_protection.releases[repo].id
  }
}
