#!/bin/sh
# Check CronJob and Job health in the cluster.
#
# Detects:
# - CronJobs with stuck active jobs (blocking concurrencyPolicy=Forbid)
# - CronJobs that haven't succeeded in a long time
# - Pods with ImagePullBackOff errors
# - Failed Jobs
#
# Usage:
#   ./check-cronjobs.sh <report-dir> [stale-threshold-hours]
#
#   stale-threshold-hours: flag CronJobs with no success in this many hours (default: 48)
#
# Outputs:
#   <report-dir>/health-report.md  — markdown report (created/appended)
#   <report-dir>/status            — "healthy" or "unhealthy"
#
# Requirements: kubectl
set -e

REPORT_DIR="${1:?Usage: $0 <report-dir> [stale-threshold-hours]}"
STALE_HOURS="${2:-48}"

REPORT_FILE="${REPORT_DIR}/health-report.md"
STATUS_FILE="${REPORT_DIR}/status"

# Initialize report if it doesn't exist
if [ ! -f "${REPORT_FILE}" ]; then
  echo "# Cluster Health Report" > "${REPORT_FILE}"
  echo "" >> "${REPORT_FILE}"
  echo "**Generated:** $(date -u '+%Y-%m-%d %H:%M:%S UTC')" >> "${REPORT_FILE}"
  echo "" >> "${REPORT_FILE}"
  echo "healthy" > "${STATUS_FILE}"
fi

# Parse ISO 8601 timestamp to epoch seconds.
# Compatible with GNU date and busybox/alpine date.
parse_ts() {
  date -u -d "$1" +%s 2>/dev/null ||
    date -u -D "%Y-%m-%dT%H:%M:%SZ" -d "$1" +%s 2>/dev/null ||
    echo "0"
}

NOW=$(date +%s)
STALE_THRESHOLD=$((STALE_HOURS * 3600))

# =========================================================
# 1. CronJobs with stuck active jobs
# =========================================================
echo "## CronJob Health" >> "${REPORT_FILE}"
echo "" >> "${REPORT_FILE}"

CRONJOB_ISSUES=""

STUCK_CJS=""
for cj in $(kubectl get cronjobs -n default -o jsonpath='{range .items[?(@.status.active)]}{.metadata.name}{"\n"}{end}' 2>/dev/null); do
  POLICY=$(kubectl get cronjob "${cj}" -n default -o jsonpath='{.spec.concurrencyPolicy}' 2>/dev/null)
  LAST_SUCCESS=$(kubectl get cronjob "${cj}" -n default -o jsonpath='{.status.lastSuccessfulTime}' 2>/dev/null)
  LAST_SCHEDULE=$(kubectl get cronjob "${cj}" -n default -o jsonpath='{.status.lastScheduleTime}' 2>/dev/null)
  STUCK_CJS="${STUCK_CJS}- \`${cj}\` (policy: ${POLICY}, last success: ${LAST_SUCCESS:-never}, last scheduled: ${LAST_SCHEDULE:-never})\n"
  echo "unhealthy" > "${STATUS_FILE}"
done

if [ -n "${STUCK_CJS}" ]; then
  CRONJOB_ISSUES="${CRONJOB_ISSUES}### CronJobs with Active (Stuck) Jobs\n\n"
  CRONJOB_ISSUES="${CRONJOB_ISSUES}These CronJobs have active jobs that haven't completed. If \`concurrencyPolicy=Forbid\`, no new runs can be scheduled.\n\n"
  CRONJOB_ISSUES="${CRONJOB_ISSUES}${STUCK_CJS}\n"
fi

# =========================================================
# 2. CronJobs that haven't succeeded in a long time
# =========================================================
STALE_CJS=""
# Get all non-suspended CronJobs with their lastSuccessfulTime and creation time
CRONJOBS_DATA=$(kubectl get cronjobs -n default -o jsonpath='{range .items[?(@.spec.suspend!=true)]}{.metadata.name}{"\t"}{.status.lastSuccessfulTime}{"\t"}{.metadata.creationTimestamp}{"\n"}{end}' 2>/dev/null)

echo "${CRONJOBS_DATA}" | while IFS="$(printf '\t')" read -r name last_success created; do
  [ -z "${name}" ] && continue

  if [ -z "${last_success}" ]; then
    # Never succeeded — only flag if the CronJob is older than the threshold
    CREATED_EPOCH=$(parse_ts "${created}")
    AGE=$((NOW - CREATED_EPOCH))
    if [ "${AGE}" -gt "${STALE_THRESHOLD}" ]; then
      printf -- "- \`%s\` — **never succeeded** (created: %s)\n" "${name}" "${created}" >> "${REPORT_DIR}/stale.tmp"
      echo "unhealthy" > "${STATUS_FILE}"
    fi
  else
    LAST_EPOCH=$(parse_ts "${last_success}")
    AGE=$((NOW - LAST_EPOCH))
    if [ "${AGE}" -gt "${STALE_THRESHOLD}" ]; then
      HOURS_AGO=$((AGE / 3600))
      printf -- "- \`%s\` — last success **%dh ago** (%s)\n" "${name}" "${HOURS_AGO}" "${last_success}" >> "${REPORT_DIR}/stale.tmp"
      echo "unhealthy" > "${STATUS_FILE}"
    fi
  fi
done

if [ -f "${REPORT_DIR}/stale.tmp" ]; then
  CRONJOB_ISSUES="${CRONJOB_ISSUES}### CronJobs Without Recent Success (>${STALE_HOURS}h)\n\n"
  CRONJOB_ISSUES="${CRONJOB_ISSUES}$(cat "${REPORT_DIR}/stale.tmp")\n\n"
  rm -f "${REPORT_DIR}/stale.tmp"
fi

if [ -z "${CRONJOB_ISSUES}" ]; then
  echo "✅ All CronJobs healthy" >> "${REPORT_FILE}"
else
  printf "%b" "${CRONJOB_ISSUES}" >> "${REPORT_FILE}"
fi
echo "" >> "${REPORT_FILE}"

# =========================================================
# 3. Pods with image pull failures
# =========================================================
echo "## Job Health" >> "${REPORT_FILE}"
echo "" >> "${REPORT_FILE}"

JOB_ISSUES=""

PULL_FAILURES=$(kubectl get pods -n default \
  --field-selector=status.phase!=Succeeded,status.phase!=Running \
  -o custom-columns='POD:.metadata.name,STATUS:.status.containerStatuses[0].state.waiting.reason,IMAGE:.spec.containers[0].image' \
  --no-headers 2>/dev/null | grep -i "ImagePull\|ErrImage" || true)
if [ -n "${PULL_FAILURES}" ]; then
  JOB_ISSUES="${JOB_ISSUES}### Pods with Image Pull Failures\n\n"
  JOB_ISSUES="${JOB_ISSUES}\`\`\`\n${PULL_FAILURES}\n\`\`\`\n\n"
  echo "unhealthy" > "${STATUS_FILE}"
fi

# =========================================================
# 4. Failed jobs
# =========================================================
FAILED_JOBS=$(kubectl get jobs -n default \
  -o custom-columns='NAME:.metadata.name,FAILED:.status.failed,CREATED:.metadata.creationTimestamp' \
  --no-headers 2>/dev/null | awk '$2 ~ /^[0-9]+$/ && $2 > 0 {print}' || true)
if [ -n "${FAILED_JOBS}" ]; then
  JOB_ISSUES="${JOB_ISSUES}### Failed Jobs\n\n"
  JOB_ISSUES="${JOB_ISSUES}\`\`\`\n${FAILED_JOBS}\n\`\`\`\n\n"
  echo "unhealthy" > "${STATUS_FILE}"
fi

if [ -z "${JOB_ISSUES}" ]; then
  echo "✅ All Jobs healthy" >> "${REPORT_FILE}"
else
  printf "%b" "${JOB_ISSUES}" >> "${REPORT_FILE}"
fi
echo "" >> "${REPORT_FILE}"

echo "=== CronJob/Job check complete ==="
cat "${REPORT_FILE}"
