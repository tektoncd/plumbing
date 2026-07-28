# VMT (Vulnerability Management Team) Tools

Tools for the Tekton VMT to manage and track security advisories across the
`tektoncd` GitHub organization.

## fetch-advisories.py

Fetches all GitHub Security Advisories (GHSAs) across `tektoncd/*` repos and
outputs structured JSON. Used to generate weekly VMT digest emails.

### Requirements

- `gh` CLI, authenticated with access to security advisories
- Python 3.10+

### Usage

```bash
# All advisories across the org
python3 vmt/fetch-advisories.py

# Only published advisories
python3 vmt/fetch-advisories.py --state published

# Advisories needing triage
python3 vmt/fetch-advisories.py --state triage

# Single repo
python3 vmt/fetch-advisories.py --repo pipeline

# Updated since a date
python3 vmt/fetch-advisories.py --since 2026-07-01
```

### Output

JSON with:
- `generated_at`: timestamp
- `org`: GitHub org
- `total`: count
- `by_state`: breakdown by state (triage, draft, published, closed)
- `advisories[]`: array of advisory objects with:
  - `repo`, `ghsa_id`, `cve_id`, `summary`, `severity`, `state`
  - `days_open`, `days_since_update` (staleness indicators)
  - `credits[]`, `collaborators[]` (who's working on it)
  - `vulnerabilities[]` (affected packages/versions)

### Weekly Digest Workflow

1. Run `fetch-advisories.py` to get current state
2. Feed JSON to an AI agent skill to generate the digest email
3. Review and send to `tekton-vmt` mailing list
