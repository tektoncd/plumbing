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

# Repository list from prow config
locals {
  # Core Tekton repositories
  tektoncd_repos = [
    "dashboard",
    "pipeline",
    "operator",
    "mcp-server",
    "triggers",
    "cli",
    "pruner",
    "chains",
    "hub",
    "results",
    "plumbing",
  ]

  # Base status checks required for all repos
  # - tide: Prow's merge automation bot
  # - EasyCLA: CLA verification
  base_status_checks = ["tide", "EasyCLA"]

  # Repository-specific status checks from prow config
  repo_specific_checks = {
    dashboard = [
      "Build tests",
      "E2E tests (k8s-oldest, read-only)",
      "E2E tests (k8s-oldest, read-write)",
      "E2E tests (k8s-plus-one, read-only)",
      "E2E tests (k8s-plus-one, read-write)",
      "Unit tests",
    ]
    pipeline = [
      "build",
      "test",
      "lint",
      "Check generated code",
      "Multi-arch build",
      "e2e-tests / e2e tests (k8s-oldest, alpha)",
      "e2e-tests / e2e tests (k8s-oldest, beta)",
      "e2e-tests / e2e tests (k8s-oldest, stable)",
      "e2e-tests / e2e tests (k8s-latest-minus-three, alpha)",
      "e2e-tests / e2e tests (k8s-latest-minus-three, beta)",
      "e2e-tests / e2e tests (k8s-latest-minus-three, stable)",
      "e2e-tests / e2e tests (k8s-latest-minus-two, alpha)",
      "e2e-tests / e2e tests (k8s-latest-minus-two, beta)",
      "e2e-tests / e2e tests (k8s-latest-minus-two, stable)",
      "e2e-tests / e2e tests (k8s-latest-minus-one, alpha)",
      "e2e-tests / e2e tests (k8s-latest-minus-one, beta)",
      "e2e-tests / e2e tests (k8s-latest-minus-one, stable)",
      "e2e-tests / e2e tests (k8s-latest, alpha)",
      "e2e-tests / e2e tests (k8s-latest, beta)",
      "e2e-tests / e2e tests (k8s-latest, stable)",
    ]
    operator = [
      "build",
      "test",
      "lint",
      "Check generated code",
      "Multi-arch build",
      "e2e-tests / e2e tests (k8s-oldest)",
      "e2e-tests / e2e tests (k8s-plus-one)",
    ]
    "mcp-server" = [
      "build",
      "test",
      "lint",
    ]
    triggers = [
      "lint",
      "Tekton Triggers CI",
    ]
    cli = [
      "build",
      "test",
      "lint",
      "Check generated code",
      "Multi-arch build",
      "e2e-tests / e2e tests (k8s-oldest)",
      "e2e-tests / e2e tests (k8s-plus-one)",
    ]
    pruner = [
      "golangci-lint / lint (pull_request)",
      "Pruner kind E2E Tests / k8s (v1.28.x) / e2e test (pull_request)",
      "Pruner kind E2E Tests / k8s (v1.32.x) / e2e test (pull_request)",
      "Pruner kind E2E Tests / pipelines-lts (v1.0.0) / e2e test (pull_request)",
    ]
  }

  # Merge base checks with repo-specific checks
  merged_status_checks = {
    for repo in local.tektoncd_repos :
    repo => concat(
      local.base_status_checks,
      lookup(local.repo_specific_checks, repo, [])
    )
  }
}
