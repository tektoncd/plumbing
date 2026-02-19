#!/usr/bin/env bash
set -euo pipefail

# Detects what changed and outputs GitHub Actions matrix variables.
#
# Usage: detect-changes.sh <event-name> [base-ref] [base-sha] [head-sha]
#
# Arguments:
#   event-name  GitHub event name (pull_request, push, schedule, workflow_dispatch)
#   base-ref    Base branch name (required for pull_request)
#   base-sha    Base commit SHA (required for pull_request)
#   head-sha    Head commit SHA (required for pull_request)
#
# Outputs (written to $GITHUB_OUTPUT):
#   go-matrix      JSON matrix of Go projects to build
#   images-matrix  JSON matrix of container images to build
#   yaml           "true" if any YAML files changed

EVENT_NAME="${1:?event-name is required}"
BASE_REF="${2:-}"
BASE_SHA="${3:-}"
HEAD_SHA="${4:-}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

ALL_GO=$(python "${ROOT_DIR}/hack/generate-go-matrix.py" "${ROOT_DIR}" | jq -c .)
mapfile -t GO_PROJECTS < <(echo "$ALL_GO" | jq -r '.project[]')
echo "Discovered Go projects: ${GO_PROJECTS[*]}"

ALL_IMAGES=$(python "${ROOT_DIR}/tekton/images/generate-matrix.py" "${ROOT_DIR}/tekton/images" | jq -c .)

# For workflow_dispatch: run everything (manual re-run of full CI)
# For schedule: rebuild all images (pick up base image updates), skip Go CI
if [[ "${EVENT_NAME}" == "workflow_dispatch" ]]; then
  echo "go-matrix=${ALL_GO}" >> "$GITHUB_OUTPUT"
  echo "images-matrix=${ALL_IMAGES}" >> "$GITHUB_OUTPUT"
  echo "yaml=true" >> "$GITHUB_OUTPUT"
  echo "workflow_dispatch event: running all jobs"
  exit 0
fi
if [[ "${EVENT_NAME}" == "schedule" ]]; then
  echo "images-matrix=${ALL_IMAGES}" >> "$GITHUB_OUTPUT"
  echo "schedule event: building all images, skipping Go CI"
  exit 0
fi

# For PRs and push to main: detect what changed
if [[ "${EVENT_NAME}" == "pull_request" ]]; then
  git fetch origin "${BASE_REF}"
  CHANGED=$(git diff --name-only "${BASE_SHA}...${HEAD_SHA}")
else
  # Push to main: compare against parent commit(s)
  CHANGED=$(git diff --name-only HEAD~1...HEAD)
fi
echo -e "Changed files:\n${CHANGED}"

if [[ -z "${CHANGED}" ]]; then
  echo "No changed files detected"
  exit 0
fi

# --- Go project detection ---
GO_MATCHED=()
RUN_ALL_GO='false'

# Root-level Go/CI changes affect all projects
if echo "$CHANGED" | grep -qE "^(go\.(mod|sum)|vendor/|\.golangci|\.github/workflows/ci\.yaml|hack/(generate-go-matrix\.py|detect-changes\.sh))"; then
  RUN_ALL_GO='true'
fi

if [[ "$RUN_ALL_GO" == "true" ]]; then
  echo "go-matrix=${ALL_GO}" >> "$GITHUB_OUTPUT"
else
  for proj in "${GO_PROJECTS[@]}"; do
    if echo "$CHANGED" | grep -q "^${proj}/"; then
      GO_MATCHED+=("\"${proj}\"")
    fi
  done
  if [[ ${#GO_MATCHED[@]} -gt 0 ]]; then
    MATRIX=$(printf '%s,' "${GO_MATCHED[@]}" | sed 's/,$//')
    echo "go-matrix={\"project\":[${MATRIX}]}" >> "$GITHUB_OUTPUT"
  fi
fi

# --- YAML detection (for yamllint) ---
YAML='false'
while read -r file; do
  if [[ "$file" == *.yaml || "$file" == *.yml ]]; then
    YAML='true'
    break
  fi
done <<< "$CHANGED"
echo "yaml=${YAML}" >> "$GITHUB_OUTPUT"

# --- Image detection ---
# If workflow or generator changed, build all images
if echo "$CHANGED" | grep -qE "^(\.github/workflows/ci\.yaml|tekton/images/generate-matrix\.py)$"; then
  echo "images-matrix=${ALL_IMAGES}" >> "$GITHUB_OUTPUT"
  exit 0
fi

# Filter to only images with changes
CHANGED_DIRS=$(echo "$CHANGED" | grep "^tekton/images/" | cut -d/ -f3 | sort -u)
if [[ -n "$CHANGED_DIRS" ]]; then
  FILTER=$(echo "$CHANGED_DIRS" | jq -R . | jq -sc '.')
  FILTERED=$(echo "$ALL_IMAGES" | jq -c --argjson dirs "$FILTER" \
    '{include: [.include[] | select(.name as $n | $dirs | index($n))]}')
  echo "images-matrix=${FILTERED}" >> "$GITHUB_OUTPUT"
fi
