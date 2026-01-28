# Community Roadmap Sync

This document describes the automated workflow that syncs issues from all Tekton
repositories to the [Community Roadmap project](https://github.com/orgs/tektoncd/projects/34).

## Overview

The Community Roadmap provides a unified view of roadmap items across all Tekton
repositories. Instead of checking individual repo projects, maintainers and
contributors can see the full picture in one place.

**Project URL:** https://github.com/orgs/tektoncd/projects/34

## How It Works

```
┌─────────────────────────────────────────────────────────────────────┐
│  Every 6 hours (or manual trigger)                                  │
│                                                                     │
│  1. Search: org:tektoncd label:area/roadmap is:open                 │
│     └─► Finds all issues with 'area/roadmap' label                  │
│                                                                     │
│  2. For each issue: add to Community Roadmap                        │
│     └─► Idempotent: safe to run repeatedly                          │
│                                                                     │
│  3. Built-in project workflows handle status changes                │
│     └─► Issue closed → Done, PR merged → Done, etc.                 │
└─────────────────────────────────────────────────────────────────────┘
```

## Adding Items to the Community Roadmap

To get an issue on the Community Roadmap:

1. Add the `area/roadmap` label to any issue in a tektoncd repository
2. Wait up to 6 hours for the next sync (or ask a maintainer to trigger manually)

That's it! The workflow will automatically pick it up.

## Status Management

Status is managed by the project's **built-in workflows**, not by this sync workflow.

When you enable these workflows on the project (Settings → Workflows):
- **Item closed** → Status set to "Done"
- **Item reopened** → Status set to "In Progress"
- **PR merged** → Status set to "Done"

Manual status changes can be made directly in the project board.

### Status Options

| Status | Description |
|--------|-------------|
| Todo | Planned work, not yet started |
| In Progress | Actively being worked on |
| Review | Implementation complete, awaiting review |
| Done | Completed and closed |
| Hold | Paused or blocked |

## Workflow Details

**File:** `.github/workflows/sync-project-status.yml`

### Schedule

- Runs automatically every 6 hours (00:00, 06:00, 12:00, 18:00 UTC)
- Can be triggered manually via GitHub Actions UI

### Manual Trigger

To trigger the workflow manually:

```bash
# Dry run (no changes, just logs what would happen)
gh workflow run sync-project-status.yml -f dry_run=true

# Actual sync
gh workflow run sync-project-status.yml -f dry_run=false
```

Or use the GitHub UI:
1. Go to Actions → "Sync Community Roadmap"
2. Click "Run workflow"
3. Optionally enable "Dry run" for testing

### Required Secrets

| Secret | Description |
|--------|-------------|
| `PROJECT_SYNC_APP_ID` | GitHub App ID |
| `PROJECT_SYNC_PRIVATE_KEY` | GitHub App private key |

#### GitHub App Setup

1. Create a GitHub App at `https://github.com/organizations/tektoncd/settings/apps/new`
2. Configure permissions:
   - **Repository permissions:**
     - `Discussions: Read` - To add discussions to projects
     - `Issues: Read` - To search issues across repositories
     - `Pull requests: Read` - To add PRs to projects
     - `Metadata: Read` - Required for API access
   - **Organization permissions:**
     - `Projects: Read and write` - To add items to organization projects
3. Install the App on the tektoncd organization
4. Generate a private key and store it as a secret:

```bash
gh secret set PROJECT_SYNC_APP_ID --repo tektoncd/plumbing
gh secret set PROJECT_SYNC_PRIVATE_KEY --repo tektoncd/plumbing < private-key.pem
```

## Repositories Included

The workflow searches ALL repositories in the `tektoncd` organization. Any repo
can participate by using the `area/roadmap` label.

Currently contributing repositories include:
- pipeline
- triggers
- cli
- dashboard
- operator
- results
- chains
- catalog
- plumbing
- community
- actions

## Troubleshooting

### Issue not appearing in Community Roadmap

1. Verify the issue has the `area/roadmap` label
2. Verify the issue is **open** (closed issues are not synced)
3. Check the workflow run logs in GitHub Actions
4. Trigger a manual sync with `dry_run=false`

### Workflow failing

Check the workflow logs for:
- Authentication errors → Verify `PROJECT_SYNC_APP_ID` and `PROJECT_SYNC_PRIVATE_KEY` are set correctly
- Rate limiting → The workflow includes rate limiting, but heavy usage may hit limits
- API errors → Check GitHub status page

## Appendix: Creating the GitHub App

Step-by-step instructions for creating the GitHub App used by this workflow.

### Step 1: Create the App

1. Go to https://github.com/organizations/tektoncd/settings/apps/new
2. Fill in the basic information:
   - **GitHub App name:** `Tekton Community Roadmap Sync`
   - **Homepage URL:** `https://github.com/tektoncd/plumbing`
3. Under **Webhook**:
   - Uncheck "Active" (webhooks are not needed)

### Step 2: Configure Permissions

Set the following permissions:

**Repository permissions:**

| Permission | Access |
|------------|--------|
| Discussions | Read-only |
| Issues | Read-only |
| Pull requests | Read-only |
| Metadata | Read-only |

**Organization permissions:**

| Permission | Access |
|------------|--------|
| Projects | Read and write |

### Step 3: Set Installation Access

- Select "Only on this account"
- Click "Create GitHub App"

### Step 4: Install the App

After the App is created:

1. In the App settings, click "Install App" in the left sidebar
2. Click "Install" next to the `tektoncd` organization
3. Select "All repositories" (required to search issues across all repos)
4. Click "Install"

### Step 5: Generate a Private Key

1. Go back to the App's settings page
2. Scroll down to "Private keys"
3. Click "Generate a private key"
4. Save the downloaded `.pem` file securely

### Step 6: Store Secrets in the Repository

Note the **App ID** from the App's settings page (shown near the top).

```bash
# Store the App ID
gh secret set PROJECT_SYNC_APP_ID --repo tektoncd/plumbing
# Enter the App ID when prompted

# Store the private key
gh secret set PROJECT_SYNC_PRIVATE_KEY --repo tektoncd/plumbing < /path/to/downloaded-private-key.pem
```

### Step 7: Verify Setup

Trigger a dry run to verify everything is configured correctly:

```bash
gh workflow run sync-project-status.yml -f dry_run=true --repo tektoncd/plumbing
```

Check the workflow run in GitHub Actions to ensure it completes successfully.
