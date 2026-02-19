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

# Single source of truth: config/repo-checks.yaml
# Shared with Prow Tide context_options (see hack/generate-tide-contexts.py)
locals {
  repo_checks = yamldecode(file("${path.module}/../../config/repo-checks.yaml"))

  # All repositories listed in the shared config
  tektoncd_repos = keys(local.repo_checks.repos)

  # Base status checks required for all repos
  # - tide: Prow's merge automation bot
  # - EasyCLA: CLA verification
  base_status_checks = local.repo_checks.base_checks

  # Repository-specific status checks
  repo_specific_checks = local.repo_checks.repos

  # Merge base checks with repo-specific checks
  merged_status_checks = {
    for repo in local.tektoncd_repos :
    repo => concat(
      local.base_status_checks,
      lookup(local.repo_specific_checks, repo, [])
    )
  }
}
