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

# Import existing main branch protections into Terraform state
# These protections already exist on GitHub and need to be imported
# so Terraform can manage them going forward.
#
# After successful import, these blocks can be removed.

import {
  to = github_branch_protection.main["dashboard"]
  id = "dashboard:main"
}

import {
  to = github_branch_protection.main["pipeline"]
  id = "pipeline:main"
}

import {
  to = github_branch_protection.main["operator"]
  id = "operator:main"
}

import {
  to = github_branch_protection.main["mcp-server"]
  id = "mcp-server:main"
}

import {
  to = github_branch_protection.main["triggers"]
  id = "triggers:main"
}

import {
  to = github_branch_protection.main["cli"]
  id = "cli:main"
}

import {
  to = github_branch_protection.main["pruner"]
  id = "pruner:main"
}

import {
  to = github_branch_protection.main["chains"]
  id = "chains:main"
}

import {
  to = github_branch_protection.main["hub"]
  id = "hub:main"
}

import {
  to = github_branch_protection.main["results"]
  id = "results:main"
}

import {
  to = github_branch_protection.main["plumbing"]
  id = "plumbing:main"
}
