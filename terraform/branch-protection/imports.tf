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

# Import existing branch protections into Terraform state
# These protections already exist on GitHub and need to be imported
# so Terraform can manage them going forward.
#
# Uses for_each to dynamically import based on tektoncd_repos list.
# After successful import, these blocks can be removed.

import {
  for_each = toset(local.tektoncd_repos)
  to       = github_branch_protection.main[each.key]
  id       = "${each.key}:main"
}

import {
  for_each = toset(local.tektoncd_repos)
  to       = github_branch_protection.releases[each.key]
  id       = "${each.key}:release-v*"
}

# hub repository was archived — its branch protection was removed from
# config/repo-checks.yaml. After the first apply, remove the stale state
# entries manually:
#   terraform state rm 'github_branch_protection.main["hub"]'
#   terraform state rm 'github_branch_protection.releases["hub"]'
