#!/usr/bin/env bash

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

# verify-crd-compat.sh — fail on breaking CRD OpenAPI changes vs a git baseline.
#
# Shared helper for Tekton component repos (operator, pipeline, …).
# Uses sigs.k8s.io/crdify (KEP-5000). Policy matches openshift/api crdify defaults.
#
# Required:
#   CRD_DIRS             space-separated dirs of CRD YAML (relative to repo root)
#
# Optional:
#   COMPAT_BASELINE_REF  git ref for old schemas
#                        (default: PULL_BASE_SHA, else upstream/main, else main)
#   CRDIFY_VERSION       pinned crdify version (default: v0.6.0)
#   CRDIFY_CONFIG        crdify --config path
#                        (default: crdify-compat.yaml next to this script)
#   CRDIFY_BIN           path to crdify binary (default: <repo>/.bin/crdify-<version>)
#
# PR CI: set COMPAT_BASELINE_REF to the PR base SHA (or rely on PULL_BASE_SHA).
# Run it, don't source it (it exits and sets shell options), e.g.:
#   CRD_DIRS=... bash vendor/github.com/tektoncd/plumbing/scripts/verify-crd-compat.sh

set -o errexit
set -o nounset
set -o pipefail

ROOT="$(git rev-parse --show-toplevel)"
cd "${ROOT}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

CRDIFY_VERSION="${CRDIFY_VERSION:-v0.6.0}"
CRDIFY_BIN="${CRDIFY_BIN:-${ROOT}/.bin/crdify-${CRDIFY_VERSION}}"   # version in path => pin is real
CRDIFY_CONFIG="${CRDIFY_CONFIG:-${SCRIPT_DIR}/crdify-compat.yaml}"

if [[ -z "${CRD_DIRS:-}" ]]; then
  echo "error: CRD_DIRS is required (space-separated dirs of CRD YAML)" >&2
  echo "example: CRD_DIRS=config/base/generated-crds $0" >&2
  exit 1
fi

resolve_baseline() {
  if [[ -n "${COMPAT_BASELINE_REF:-}" ]]; then
    echo "${COMPAT_BASELINE_REF}"
    return
  fi
  if [[ -n "${PULL_BASE_SHA:-}" ]]; then
    echo "${PULL_BASE_SHA}"
    return
  fi
  if git rev-parse --verify --quiet upstream/main >/dev/null; then
    echo "upstream/main"
    return
  fi
  if git rev-parse --verify --quiet main >/dev/null; then
    echo "main"
    return
  fi
  echo "error: set COMPAT_BASELINE_REF (no PULL_BASE_SHA / upstream/main / main)" >&2
  exit 1
}

ensure_crdify() {
  if [[ -x "${CRDIFY_BIN}" ]]; then
    return
  fi
  local bindir
  bindir="$(dirname "${CRDIFY_BIN}")"
  mkdir -p "${bindir}"
  echo "installing sigs.k8s.io/crdify@${CRDIFY_VERSION} -> ${CRDIFY_BIN}"
  # go install always names the binary "crdify"; move to the versioned path.
  GOBIN="${bindir}" go install "sigs.k8s.io/crdify@${CRDIFY_VERSION}"
  if [[ "${CRDIFY_BIN}" != "${bindir}/crdify" ]]; then
    mv "${bindir}/crdify" "${CRDIFY_BIN}"
  fi
  if [[ ! -x "${CRDIFY_BIN}" ]]; then
    echo "error: crdify install failed; expected ${CRDIFY_BIN}" >&2
    exit 1
  fi
}

# Reads YAML on stdin. No -q: grep must drain stdin or `git show | is_crd` can SIGPIPE under pipefail.
is_crd() {
  grep -E '^kind:[[:space:]]*"?CustomResourceDefinition"?([[:space:]]|$)' >/dev/null
}

BASELINE="$(resolve_baseline)"
if ! git rev-parse --verify --quiet "${BASELINE}^{commit}" >/dev/null; then
  echo "error: baseline ref ${BASELINE} not found locally; fetch it first" >&2
  exit 1
fi

ensure_crdify

if [[ ! -f "${CRDIFY_CONFIG}" ]]; then
  echo "error: crdify config not found: ${CRDIFY_CONFIG}" >&2
  exit 1
fi

echo "CRD compatibility: baseline=${BASELINE} dirs=${CRD_DIRS} crdify=${CRDIFY_VERSION} config=${CRDIFY_CONFIG}"

failed=0
checked=0
skipped_new=0

# CRDs present at baseline but gone from the PR (deleted, or renamed/moved) are breaking.
while IFS= read -r -d '' f; do
  git show "${BASELINE}:${f}" | is_crd || continue
  echo "FAIL (removed/renamed vs baseline): ${f}" >&2
  failed=1
done < <(git diff --no-renames --name-only -z --diff-filter=D "${BASELINE}" -- ${CRD_DIRS})

for dir in ${CRD_DIRS}; do
  if [[ ! -d "${dir}" ]]; then
    echo "error: CRD dir not found: ${dir}" >&2
    exit 1
  fi

  while IFS= read -r -d '' file; do
    rel="${file#./}"
    is_crd < "${rel}" || { echo "skip (not a CRD): ${rel}"; continue; }

    if ! git cat-file -e "${BASELINE}:${rel}" 2>/dev/null; then
      echo "skip (new at HEAD): ${rel}"
      skipped_new=$((skipped_new + 1))
      continue
    fi

    checked=$((checked + 1))
    echo "check: ${rel}"
    if ! "${CRDIFY_BIN}" --config "${CRDIFY_CONFIG}" \
      "git://${BASELINE}?path=${rel}" \
      "file://${ROOT}/${rel}"; then
      failed=1
    fi
  done < <(find "${dir}" \( -name '*.yaml' -o -name '*.yml' \) -print0 | sort -z)
done

echo "summary: checked=${checked} skipped_new=${skipped_new}"

if [[ "${checked}" -eq 0 ]]; then
  echo "error: no CRDs compared against ${BASELINE}" >&2
  exit 1
fi

if [[ "${failed}" -ne 0 ]]; then
  echo "FAIL: breaking CRD changes vs ${BASELINE}" >&2
  echo "Intentional? adjust CRDIFY_CONFIG or document the break for reviewers." >&2
  exit 1
fi

echo "PASS: CRD schemas compatible with ${BASELINE}"
