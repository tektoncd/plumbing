#!/usr/bin/env python3
"""Fetch security advisories across all tektoncd/* repos.

Outputs structured JSON suitable for generating a weekly VMT digest.
Requires `gh` CLI with org admin/security access.

Usage:
    python3 vmt/fetch-advisories.py [--org ORG] [--repo REPO] [--state triage|draft|published|closed|all] [--since YYYY-MM-DD]
"""

import argparse
import json
import subprocess
import sys
from datetime import datetime, timezone


def gh_api(endpoint, repo=None):
    """Call gh api with pagination, return parsed JSON.

    Returns [] for 404 (no advisories enabled). Raises on other errors.
    """
    result = subprocess.run(
        ["gh", "api", "--paginate", endpoint],
        capture_output=True, text=True
    )
    if result.returncode != 0:
        stderr = result.stderr.strip()
        # 404 = advisories not enabled for this repo — expected, skip
        if "404" in stderr or "Not Found" in stderr:
            return []
        # Real error (auth, rate limit, network) — surface it
        label = repo or endpoint
        print(f"Warning: gh api failed for {label}: {stderr}", file=sys.stderr)
        return []
    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError:
        return []


def list_org_repos(org):
    """List all repos in an org."""
    result = subprocess.run(
        ["gh", "api", "--paginate", f"/orgs/{org}/repos", "-q", ".[].name"],
        capture_output=True, text=True
    )
    if result.returncode != 0:
        print(f"Error listing repos for {org}: {result.stderr}", file=sys.stderr)
        sys.exit(1)
    return [name.strip() for name in result.stdout.strip().split("\n") if name.strip()]


def days_since(iso_date):
    """Return days since an ISO 8601 date string."""
    if not iso_date:
        return None
    dt = datetime.fromisoformat(iso_date.replace("Z", "+00:00"))
    return (datetime.now(timezone.utc) - dt).days


def extract_advisory(repo, adv):
    """Extract relevant fields from a raw advisory."""
    return {
        "repo": repo,
        "ghsa_id": adv.get("ghsa_id"),
        "cve_id": adv.get("cve_id"),
        "summary": adv.get("summary", ""),
        "severity": adv.get("severity"),
        "state": adv.get("state"),
        "html_url": adv.get("html_url"),
        "created_at": adv.get("created_at"),
        "updated_at": adv.get("updated_at"),
        "published_at": adv.get("published_at"),
        "closed_at": adv.get("closed_at"),
        "withdrawn_at": adv.get("withdrawn_at"),
        "days_open": days_since(adv.get("created_at") or adv.get("published_at")),
        "days_since_update": days_since(adv.get("updated_at")),
        "submission_accepted": (adv.get("submission") or {}).get("accepted"),
        "author": (adv.get("author") or {}).get("login"),
        "credits": [
            {"login": c["login"], "type": c["type"]}
            for c in (adv.get("credits") or [])
        ],
        "collaborators": [
            u["login"]
            for u in (adv.get("collaborating_users") or [])
        ],
        "collaborating_teams": [
            t["slug"]
            for t in (adv.get("collaborating_teams") or [])
        ],
        "cvss_score": (adv.get("cvss") or {}).get("score"),
        "cwes": adv.get("cwe_ids", []),
        "vulnerabilities": [
            {
                "package": (v.get("package") or {}).get("name"),
                "ecosystem": (v.get("package") or {}).get("ecosystem"),
                "vulnerable_range": v.get("vulnerable_version_range"),
                "patched_versions": v.get("patched_versions"),
            }
            for v in (adv.get("vulnerabilities") or [])
        ],
    }


def main():
    parser = argparse.ArgumentParser(description="Fetch tektoncd security advisories")
    parser.add_argument("--org", default="tektoncd", help="GitHub org (default: tektoncd)")
    parser.add_argument("--state", default="all",
                        choices=["triage", "draft", "published", "closed", "all"],
                        help="Filter by advisory state (triage/draft are client-side filtered)")
    parser.add_argument("--since", default=None, help="Only include advisories updated since YYYY-MM-DD")
    parser.add_argument("--repo", default=None, help="Single repo instead of whole org")
    args = parser.parse_args()

    repos = [args.repo] if args.repo else list_org_repos(args.org)

    since_dt = None
    if args.since:
        since_dt = datetime.fromisoformat(args.since).replace(tzinfo=timezone.utc)

    all_advisories = []
    for repo in repos:
        endpoint = f"/repos/{args.org}/{repo}/security-advisories"
        # API only supports open/closed/published; triage/draft are client-side
        api_state = args.state if args.state in ("closed", "published") else None
        if api_state:
            endpoint += f"?state={api_state}"

        advisories = gh_api(endpoint, repo=repo)
        if not isinstance(advisories, list):
            continue

        for adv in advisories:
            if since_dt and adv.get("updated_at"):
                updated = datetime.fromisoformat(
                    adv["updated_at"].replace("Z", "+00:00")
                )
                if updated < since_dt:
                    continue
            # Client-side state filter for triage/draft
            if args.state in ("triage", "draft") and adv.get("state") != args.state:
                continue
            all_advisories.append(extract_advisory(repo, adv))

    # Sort: open/triage first, then by staleness
    state_order = {"triage": 0, "draft": 1, "published": 2, "closed": 3}
    all_advisories.sort(key=lambda a: (
        state_order.get(a["state"], 99),
        -(a["days_open"] or 0),
    ))

    output = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "org": args.org,
        "total": len(all_advisories),
        "by_state": {},
        "advisories": all_advisories,
    }
    for adv in all_advisories:
        state = adv["state"] or "unknown"
        output["by_state"][state] = output["by_state"].get(state, 0) + 1

    json.dump(output, sys.stdout, indent=2)
    print()  # trailing newline


if __name__ == "__main__":
    main()
