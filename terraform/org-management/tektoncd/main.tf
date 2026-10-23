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

# --- Org settings ---
# Manages top-level organization settings from org.yaml.
# Requires the GitHub token to have admin:org scope.
resource "github_organization_settings" "tektoncd" {
  billing_email                            = local.org_settings.billing_email
  default_repository_permission            = local.org_settings.default_repository_permission
  has_organization_projects                = local.org_settings.has_organization_projects
  has_repository_projects                  = local.org_settings.has_repository_projects
  members_can_create_repositories          = false
  members_can_create_public_repositories   = false
  members_can_create_private_repositories  = false
  members_can_create_internal_repositories = false
  members_can_create_pages                 = true
  members_can_fork_private_repositories    = false
}

# --- Org membership ---
# Manages organization members and admins.
# Sends an invitation if the user is not already a member.
resource "github_membership" "this" {
  for_each = local.all_members

  username = each.key
  role     = each.value # "admin" or "member"
}

# --- Teams ---
# Creates and manages all teams defined in org.yaml.
resource "github_team" "this" {
  for_each = local.teams

  name        = each.key
  description = try(each.value.description, "")
  privacy     = try(each.value.privacy, "closed")
}

# --- Team memberships ---
# Manages the members and maintainers of each team.
resource "github_team_membership" "this" {
  for_each = local.team_memberships

  team_id  = github_team.this[each.value.team].id
  username = each.value.user
  role     = each.value.role # "maintainer" or "member"
}

# --- Team repository permissions ---
# Grants teams access to repositories with the specified permission level.
resource "github_team_repository" "this" {
  for_each = local.team_repositories

  team_id    = github_team.this[each.value.team].id
  repository = each.value.repo
  permission = each.value.permission # "pull", "push", "maintain", "admin"
}
