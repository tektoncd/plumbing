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

terraform {
  required_version = ">= 1.5"

  # Store state in a Kubernetes Secret so it persists across ephemeral
  # Tekton TaskRun pods. Without this, every run would start with empty
  # state, import all resources, and destroy+recreate them — leaving
  # repos unprotected during the ~3 minute window.
  backend "kubernetes" {
    secret_suffix     = "branch-protection"
    namespace         = "default"
    in_cluster_config = true
  }

  required_providers {
    github = {
      source  = "integrations/github"
      version = "~> 6.0"
    }
  }
}

provider "github" {
  owner = "tektoncd"
}
