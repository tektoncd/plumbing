#!/bin/sh
# Report cluster health status via GitHub issues.
#
# Behavior:
# - Unhealthy + no open issue  → create a new issue
# - Unhealthy + open issue     → add comment with latest report
# - Healthy   + open issue     → comment "resolved" and close it
# - Healthy   + no open issue  → nothing to do
#
# Usage:
#   ./report.sh <report-dir>
#
# Environment:
#   GITHUB_TOKEN — GitHub API token with issue creation permissions
#
# Requirements: wget
set -e

REPORT_DIR="${1:?Usage: $0 <report-dir>}"
REPORT_FILE="${REPORT_DIR}/health-report.md"
STATUS_FILE="${REPORT_DIR}/status"
STATUS=$(cat "${STATUS_FILE}")

: "${GITHUB_TOKEN:?GITHUB_TOKEN must be set}"

REPO="tektoncd/plumbing"
API="https://api.github.com/repos/${REPO}"
LABELS="area/infra,kind/monitoring"
AUTH_HEADER="Authorization: token ${GITHUB_TOKEN}"
ACCEPT_HEADER="Accept: application/vnd.github.v3+json"

# Find existing open monitoring issue (returns issue number or empty)
find_open_issue() {
  wget -q -O- \
    --header="${AUTH_HEADER}" \
    --header="${ACCEPT_HEADER}" \
    "${API}/issues?labels=${LABELS}&state=open&per_page=1" \
    | sed -n 's/.*"number": *\([0-9]*\).*/\1/p' | head -1
}

# Encode text as a JSON string value (with surrounding quotes).
# Reads from stdin, writes to stdout.
json_encode() {
  sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g' | awk '{printf "%s\\n", $0}' | sed 's/^/"/; s/$/"/'
}

# Add a comment to an issue
add_comment() {
  issue_number="$1"
  body_file="$2"

  PAYLOAD_FILE="/tmp/comment-payload.json"
  printf '{"body": ' > "${PAYLOAD_FILE}"
  json_encode < "${body_file}" >> "${PAYLOAD_FILE}"
  printf '}' >> "${PAYLOAD_FILE}"

  wget -q -O- --post-file="${PAYLOAD_FILE}" \
    --header="${AUTH_HEADER}" \
    --header="${ACCEPT_HEADER}" \
    --header="Content-Type: application/json" \
    "${API}/issues/${issue_number}/comments" > /dev/null
}

# Close an issue via PATCH. Tries curl first (supports PATCH natively),
# falls back to wget --method=PATCH (busybox wget may not support it).
close_issue() {
  issue_number="$1"

  if command -v curl > /dev/null 2>&1; then
    curl -s -X PATCH \
      -H "${AUTH_HEADER}" \
      -H "${ACCEPT_HEADER}" \
      -H "Content-Type: application/json" \
      -d '{"state": "closed", "state_reason": "completed"}' \
      "${API}/issues/${issue_number}" > /dev/null
  else
    PAYLOAD_FILE="/tmp/close-payload.json"
    printf '{"state": "closed", "state_reason": "completed"}' > "${PAYLOAD_FILE}"
    wget -q -O- --post-file="${PAYLOAD_FILE}" \
      --header="${AUTH_HEADER}" \
      --header="${ACCEPT_HEADER}" \
      --header="Content-Type: application/json" \
      --method=PATCH \
      "${API}/issues/${issue_number}" > /dev/null 2>&1 || {
        echo "⚠️  Could not auto-close issue #${issue_number}"
        echo "   Please close it manually."
      }
  fi
}

EXISTING_ISSUE=$(find_open_issue)

TIMESTAMP=$(date -u '+%Y-%m-%d %H:%M UTC')
TMPDIR="/tmp/health-monitor"
mkdir -p "${TMPDIR}"

if [ "${STATUS}" = "healthy" ]; then
  if [ -n "${EXISTING_ISSUE}" ]; then
    echo "✅ Cluster healthy — closing issue #${EXISTING_ISSUE}"
    cat > "${TMPDIR}/resolved.md" <<EOF
## ✅ Resolved

Cluster health checks are passing as of ${TIMESTAMP}.

Auto-closing this issue.
EOF
    add_comment "${EXISTING_ISSUE}" "${TMPDIR}/resolved.md"
    close_issue "${EXISTING_ISSUE}"
  else
    echo "✅ Cluster is healthy, nothing to do."
  fi
  exit 0
fi

# --- Cluster is unhealthy ---

if [ -n "${EXISTING_ISSUE}" ]; then
  echo "⚠️  Cluster unhealthy — updating issue #${EXISTING_ISSUE}"
  cat > "${TMPDIR}/update.md" <<EOF
## Health Report Update — ${TIMESTAMP}

$(cat "${REPORT_FILE}")
EOF
  add_comment "${EXISTING_ISSUE}" "${TMPDIR}/update.md"
  echo "✅ Comment added to issue #${EXISTING_ISSUE}"
else
  echo "⚠️  Cluster unhealthy — creating new issue"

  TITLE="[Cluster Health Monitor] Issues detected on $(date -u '+%Y-%m-%d')"

  cat > "${TMPDIR}/body.md" <<EOF
$(cat "${REPORT_FILE}")

---
*This issue was automatically created by the cluster-health-monitor Task.*
*It will be updated with new reports and auto-closed when the cluster is healthy.*
EOF

  PAYLOAD_FILE="${TMPDIR}/payload.json"
  printf '{"title": "%s", "body": ' "${TITLE}" > "${PAYLOAD_FILE}"
  json_encode < "${TMPDIR}/body.md" >> "${PAYLOAD_FILE}"
  printf ', "labels": ["area/infra", "kind/monitoring"]}' >> "${PAYLOAD_FILE}"

  RESPONSE=$(wget -q -O- --post-file="${PAYLOAD_FILE}" \
    --header="${AUTH_HEADER}" \
    --header="${ACCEPT_HEADER}" \
    --header="Content-Type: application/json" \
    "${API}/issues")

  ISSUE_NUM=$(echo "${RESPONSE}" | sed -n 's/.*"number": *\([0-9]*\).*/\1/p' | head -1)
  echo "✅ Created issue #${ISSUE_NUM}"
fi
