#!/bin/sh
# Check PipelineRun health across namespaces.
#
# Uses smart filtering to avoid noise from flaky tests:
# 1. Consistently failing: all last N runs of a pipeline failed
# 2. Infrastructure failures: ImagePullBackOff, timeouts, etc — always flagged
# 3. Regressions: pipeline was succeeding but is now all-failing
#
# Pipelines with mixed success/failure (flaky) are NOT flagged.
#
# Usage:
#   ./check-runs.sh <report-dir> [namespaces] [window-size]
#
#   namespaces:  comma-separated (default: default,tekton-ci,tekton-nightly,bastion-p,bastion-z)
#   window-size: number of recent runs to consider (default: 3)
#
# Outputs:
#   <report-dir>/health-report.md  — markdown report (appended)
#   <report-dir>/status            — "healthy" or "unhealthy"
#
# Requirements: kubectl
set -e

REPORT_DIR="${1:?Usage: $0 <report-dir> [namespaces] [window-size]}"
NAMESPACES="${2:-default,tekton-ci,tekton-nightly,bastion-p,bastion-z}"
WINDOW="${3:-3}"

REPORT_FILE="${REPORT_DIR}/health-report.md"
STATUS_FILE="${REPORT_DIR}/status"

# Infrastructure failure reasons that always indicate a platform problem,
# regardless of failure rate. These are never caused by user code.
INFRA_REASONS="TaskRunImagePullFailed|ImagePullBackOff|PipelineRunTimeout|TaskRunTimeout|CouldntGetTask|CouldntGetPipeline|CreateContainerConfigError|ExceededResourceQuota|ExceededNodeResources"

echo "## PipelineRun Health" >> "${REPORT_FILE}"
echo "" >> "${REPORT_FILE}"

HAS_ISSUES=false

for ns in $(echo "${NAMESPACES}" | tr ',' ' '); do
  # Get all PipelineRuns: pipeline_name, failure_reason, succeeded (True/False/Unknown)
  # Sorted by creation time (oldest first), so `tail` gives us the most recent.
  ALL_RUNS=$(kubectl get pipelineruns -n "${ns}" \
    --sort-by=.metadata.creationTimestamp \
    -o custom-columns='PIPELINE:.metadata.labels.tekton\.dev/pipeline,REASON:.status.conditions[0].reason,STATUS:.status.conditions[0].status' \
    --no-headers 2>/dev/null || true)

  [ -z "${ALL_RUNS}" ] && continue

  # Get unique pipeline names
  PIPELINES=$(echo "${ALL_RUNS}" | awk '{print $1}' | sort -u)

  NS_INFRA=""
  NS_CONSISTENT=""
  NS_REGRESSION=""

  for pipeline in ${PIPELINES}; do
    [ "${pipeline}" = "<none>" ] && continue

    # All runs for this pipeline
    P_ALL=$(echo "${ALL_RUNS}" | awk -v p="${pipeline}" '$1 == p')
    # Last N runs (the window we evaluate)
    P_RECENT=$(echo "${P_ALL}" | tail -n "${WINDOW}")

    TOTAL=$(echo "${P_RECENT}" | wc -l | tr -d ' ')
    FAILED=$(echo "${P_RECENT}" | awk '$3 != "True"' | wc -l | tr -d ' ')

    # --- Check 1: Infrastructure failures (always flag) ---
    INFRA_HITS=$(echo "${P_RECENT}" | grep -cE "${INFRA_REASONS}" || true)
    if [ "${INFRA_HITS}" -gt 0 ]; then
      INFRA_DETAILS=$(echo "${P_RECENT}" | grep -E "${INFRA_REASONS}" | awk '{print $2}' | sort | uniq -c | sort -rn | awk '{printf "%s (x%s), ", $2, $1}' | sed 's/, $//')
      NS_INFRA="${NS_INFRA}- \`${pipeline}\` — ${INFRA_DETAILS}\n"
    fi

    # --- Check 2: Consistently failing (all N runs failed) ---
    if [ "${FAILED}" -eq "${TOTAL}" ] && [ "${TOTAL}" -ge "${WINDOW}" ]; then
      # Get the most common failure reason
      TOP_REASON=$(echo "${P_RECENT}" | awk '{print $2}' | sort | uniq -c | sort -rn | head -1 | awk '{print $2}')

      # Skip if already flagged as infra
      if ! echo "${NS_INFRA}" | grep -q "\`${pipeline}\`"; then
        # --- Check 3: Is this a regression? ---
        # Look at runs before the window for any successes
        P_OLDER=$(echo "${P_ALL}" | head -n -"${WINDOW}")
        HAD_SUCCESS=false
        if [ -n "${P_OLDER}" ]; then
          OLD_SUCCESS=$(echo "${P_OLDER}" | awk '$3 == "True"' | wc -l | tr -d ' ')
          [ "${OLD_SUCCESS}" -gt 0 ] && HAD_SUCCESS=true
        fi

        if [ "${HAD_SUCCESS}" = "true" ]; then
          NS_REGRESSION="${NS_REGRESSION}- \`${pipeline}\` — last ${TOTAL} runs failed (${TOP_REASON}), was previously succeeding\n"
        else
          NS_CONSISTENT="${NS_CONSISTENT}- \`${pipeline}\` — last ${TOTAL} runs failed (${TOP_REASON})\n"
        fi
      fi
    fi
  done

  # Write findings for this namespace
  NS_HAS_ISSUES=false

  if [ -n "${NS_INFRA}" ]; then
    echo "### Infrastructure Failures in \`${ns}\`" >> "${REPORT_FILE}"
    echo "" >> "${REPORT_FILE}"
    printf "%b" "${NS_INFRA}" >> "${REPORT_FILE}"
    echo "" >> "${REPORT_FILE}"
    NS_HAS_ISSUES=true
  fi

  if [ -n "${NS_REGRESSION}" ]; then
    echo "### Regressions in \`${ns}\`" >> "${REPORT_FILE}"
    echo "" >> "${REPORT_FILE}"
    echo "Pipelines that were succeeding but are now consistently failing:" >> "${REPORT_FILE}"
    echo "" >> "${REPORT_FILE}"
    printf "%b" "${NS_REGRESSION}" >> "${REPORT_FILE}"
    echo "" >> "${REPORT_FILE}"
    NS_HAS_ISSUES=true
  fi

  if [ -n "${NS_CONSISTENT}" ]; then
    echo "### Consistently Failing in \`${ns}\`" >> "${REPORT_FILE}"
    echo "" >> "${REPORT_FILE}"
    printf "%b" "${NS_CONSISTENT}" >> "${REPORT_FILE}"
    echo "" >> "${REPORT_FILE}"
    NS_HAS_ISSUES=true
  fi

  if [ "${NS_HAS_ISSUES}" = "true" ]; then
    HAS_ISSUES=true
    echo "unhealthy" > "${STATUS_FILE}"
  fi
done

if [ "${HAS_ISSUES}" = "false" ]; then
  echo "✅ No actionable PipelineRun failures" >> "${REPORT_FILE}"
fi
echo "" >> "${REPORT_FILE}"

# =========================================================
# Summary
# =========================================================
STATUS=$(cat "${STATUS_FILE}")
echo "---" >> "${REPORT_FILE}"
echo "" >> "${REPORT_FILE}"
echo "**Overall Status:** ${STATUS}" >> "${REPORT_FILE}"

echo "=== Full report ==="
cat "${REPORT_FILE}"
echo ""
echo "Status: ${STATUS}"
