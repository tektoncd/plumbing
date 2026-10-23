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

# Parse the peribolos-format org.yaml from tektoncd/community.
#
# The peribolos format uses:
#   orgs.<org>.admins       — list of admin usernames
#   orgs.<org>.members      — list of member usernames
#   orgs.<org>.teams.<name> — team with maintainers, members, repos, privacy, description
#
# We transform these into flat maps suitable for Terraform for_each.

locals {
  org_config = yamldecode(file(var.org_config_path))
  tektoncd   = local.org_config.orgs.tektoncd

  # --- Org membership ---
  # Peribolos has separate "admins" and "members" lists.
  # Transform into a single map: { username => role }
  # Use distinct() to handle duplicate entries in the source YAML.
  org_admins  = { for user in distinct(local.tektoncd.admins) : user => "admin" }
  org_members = { for user in distinct(local.tektoncd.members) : user => "member" }
  all_members = merge(local.org_admins, local.org_members)

  # --- Org settings ---
  org_settings = {
    billing_email                 = local.tektoncd.billing_email
    default_repository_permission = local.tektoncd.default_repository_permission
    has_organization_projects     = local.tektoncd.has_organization_projects
    has_repository_projects       = local.tektoncd.has_repository_projects
  }

  # --- Teams ---
  teams = local.tektoncd.teams

  # --- Team memberships ---
  # Build flat map: { "team-name:username" => { team, user, role } }
  # Peribolos has separate "maintainers" and "members" lists per team.
  team_memberships = merge([
    for team_name, team in local.teams : merge(
      {
        for user in coalesce(try(team.maintainers, null), []) :
        "${team_name}:${user}" => {
          team = team_name
          user = user
          role = "maintainer"
        }
      },
      {
        for user in coalesce(try(team.members, null), []) :
        "${team_name}:${user}" => {
          team = team_name
          user = user
          role = "member"
        }
      }
    )
  ]...)

  # --- Permission mapping ---
  # Peribolos uses "read" and "write" while the GitHub API / Terraform
  # provider expects "pull" and "push". Other values (maintain, admin,
  # triage) are the same in both.
  permission_map = {
    read     = "pull"
    write    = "push"
    pull     = "pull"
    push     = "push"
    triage   = "triage"
    maintain = "maintain"
    admin    = "admin"
  }

  # --- Team repository permissions ---
  # Build flat map: { "team-name:repo" => { team, repo, permission } }
  team_repositories = merge([
    for team_name, team in local.teams : {
      for repo, permission in coalesce(try(team.repos, null), {}) :
      "${team_name}:${repo}" => {
        team       = team_name
        repo       = repo
        permission = lookup(local.permission_map, permission, permission)
      }
    }
  ]...)
}
