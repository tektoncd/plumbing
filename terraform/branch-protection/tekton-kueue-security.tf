# Copyright 2026 The Tekton Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

resource "github_repository" "tekton_kueue" {
  name         = "tekton-kueue"
  description  = "Kueue extension for Tekton"
  homepage_url = "https://tekton.dev"
  visibility   = "public"

  has_issues      = true
  has_projects    = true
  has_wiki        = true
  has_discussions = false
  is_template     = false

  allow_merge_commit     = true
  allow_squash_merge     = true
  allow_rebase_merge     = true
  allow_auto_merge       = false
  allow_update_branch    = false
  delete_branch_on_merge = false

  merge_commit_title          = "MERGE_MESSAGE"
  merge_commit_message        = "PR_TITLE"
  squash_merge_commit_title   = "COMMIT_OR_PR_TITLE"
  squash_merge_commit_message = "COMMIT_MESSAGES"

  web_commit_signoff_required = false
  archived                    = false

  topics = [
    "hacktoberfest",
    "kueue",
    "pipelines",
    "tekton",
  ]

  security_and_analysis {
    secret_scanning {
      status = "enabled"
    }

    secret_scanning_push_protection {
      status = "enabled"
    }
  }

  archive_on_destroy = true

  lifecycle {
    prevent_destroy = true
  }
}

resource "github_repository_vulnerability_alerts" "tekton_kueue" {
  repository = github_repository.tekton_kueue.name
  enabled    = true
}

resource "github_repository_dependabot_security_updates" "tekton_kueue" {
  repository = github_repository.tekton_kueue.name
  enabled    = true

  depends_on = [github_repository_vulnerability_alerts.tekton_kueue]
}

resource "github_workflow_repository_permissions" "tekton_kueue" {
  repository                       = github_repository.tekton_kueue.name
  default_workflow_permissions     = "read"
  can_approve_pull_request_reviews = false
}

import {
  to = github_repository.tekton_kueue
  id = "tekton-kueue"
}

import {
  to = github_repository_vulnerability_alerts.tekton_kueue
  id = "tekton-kueue"
}

import {
  to = github_workflow_repository_permissions.tekton_kueue
  id = "tekton-kueue"
}

# The GitHub provider does not currently expose private vulnerability
# reporting or secret-scanning validity checks. Its v6.13 implementation also
# does not write non-provider patterns despite exposing that schema. Those
# controls remain administrator settings until provider support is available.
