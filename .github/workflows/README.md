# GitHub Actions Workflows for Tekton Nightly Releases

This directory contains the **clean, production-ready migration** from traditional Tekton cronjobs to GitHub Actions for all nightly releases and CI operations.

## 🎯 Migration Overview

### Traditional System (Legacy)
- **Location**: `tekton/cronjobs/` with complex Tekton-based triggering
- **Infrastructure**: Required persistent GCP clusters with EventListeners
- **Complexity**: Multi-step process with UUID generation, curl triggers, and manual resource management
- **Maintenance**: High overhead with cluster dependencies and custom trigger logic

### GitHub Actions System (Current)
- **Location**: `.github/workflows/` with native GitHub Actions
- **Infrastructure**: Ephemeral Kind clusters created per job
- **Simplicity**: Direct workflow dispatch with built-in GitHub integrations
- **Maintenance**: Minimal overhead with automatic cleanup and GitHub-managed runners

## 🏗️ Architecture

### Core Components

#### 1. **Reusable Setup Action** (`.github/actions/setup-tekton/`)
```yaml
- name: Setup Tekton environment
  uses: ./.github/actions/setup-tekton
  with:
    kubernetes-version: 'v1.31.0'
    enable-chains: true
    cluster-name: 'my-release'
```

**Features:**
- Production-ready Kind cluster creation
- Complete Tekton stack installation (Pipeline, Triggers, Chains)  
- Supply chain security with Tekton Chains
- Proper RBAC and namespace setup
- Comprehensive error handling and retries

#### 2. **Reusable Workflow Template** (`.github/workflows/nightly-release-template.yml`)
```yaml
jobs:
  release:
    uses: ./.github/workflows/nightly-release-template.yml
    with:
      project-name: 'pipeline'
      git-repository: 'github.com/tektoncd/pipeline'
      registry-namespace: 'tektoncd/pipeline'
    secrets:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      REGISTRY_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Features:**
- Consistent release process across all projects
- Configurable testing and build options
- GitHub OIDC integration for enhanced security
- Automatic attestation generation
- Comprehensive logging and artifact collection

## 📦 Workflow Coverage

### Tekton Core Components
| Component | Workflow | Schedule | Registry |
|-----------|----------|----------|----------|
| **Pipeline** | `nightly-pipeline.yml` | Daily 2 AM UTC | `ghcr.io/tektoncd/pipeline` |
| **Triggers** | `nightly-triggers.yml` | Daily 3 AM UTC | `ghcr.io/tektoncd/triggers` |
| **Dashboard** | `nightly-dashboard.yml` | Daily 4 AM UTC | `ghcr.io/tektoncd/dashboard` |
| **Chains** | `nightly-chains.yml` | Daily 5 AM UTC | `ghcr.io/tektoncd/chains` |
| **Operator** | `nightly-operator.yml` | Daily 6 AM UTC | `ghcr.io/tektoncd/operator` |

### Plumbing Components  
| Component | Path | Registry |
|-----------|------|----------|
| **add-pr-body** | `tekton/ci/interceptors/add-pr-body` | `ghcr.io/tektoncd/plumbing/interceptors/add-pr-body` |
| **add-pr-body-ci** | `tekton/ci/cluster-interceptors/add-pr-body` | `ghcr.io/tektoncd/plumbing/cluster-interceptors/add-pr-body` |
| **add-team-members** | `tekton/ci/interceptors/add-team-members` | `ghcr.io/tektoncd/plumbing/interceptors/add-team-members` |
| **pr-commenter** | `tekton/ci/custom-tasks/pr-commenter` | `ghcr.io/tektoncd/plumbing/custom-tasks/pr-commenter` |
| **pr-status-updater** | `tekton/ci/custom-tasks/pr-status-updater` | `ghcr.io/tektoncd/plumbing/custom-tasks/pr-status-updater` |

**Schedule**: Daily 7 AM UTC (via `nightly-plumbing-components.yml`)

## 🚀 Usage Examples

### Manual Triggering

#### Individual Tekton Projects
```bash
# Trigger specific project releases
gh workflow run nightly-pipeline.yml
gh workflow run nightly-triggers.yml -f run-tests=true
gh workflow run nightly-dashboard.yml -f run-tests=false
```

#### Plumbing Components
```bash
# Build all components
gh workflow run nightly-plumbing-components.yml

# Build specific components
gh workflow run nightly-plumbing-components.yml -f components="add-pr-body,pr-commenter"

# Build with tests
gh workflow run nightly-plumbing-components.yml -f run-tests=true
```

### Programmatic Usage

#### Using the Reusable Template
```yaml
name: My Custom Release
on:
  workflow_dispatch:

jobs:
  my-release:
    uses: tektoncd/plumbing/.github/workflows/nightly-release-template.yml@main
    with:
      project-name: 'my-project'
      git-repository: 'github.com/my-org/my-project'
      registry-namespace: 'my-org/my-project'
      run-tests: true
    secrets:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      REGISTRY_TOKEN: ${{ secrets.REGISTRY_TOKEN }}
```

## 🔧 Best Practices Implementation

### 🛡️ Security
- **Minimal Permissions**: Each workflow uses least-privilege access
- **OIDC Integration**: Secure authentication without long-lived credentials
- **Supply Chain Security**: Tekton Chains integration for artifact signing
- **GitHub Attestations**: Automated provenance generation

### ⚡ Performance
- **Parallel Execution**: Matrix strategy for plumbing components
- **Efficient Caching**: Proper Docker layer caching with Buildx
- **Resource Optimization**: Right-sized Kind clusters with appropriate timeouts
- **Fast Feedback**: Early failure detection with proper error handling

### 🔍 Observability
- **Rich Logging**: Emojis and structured output for easy debugging
- **GitHub Summaries**: Comprehensive release summaries with metadata
- **Artifact Collection**: Proper retention and organization of build outputs
- **Status Tracking**: Clear success/failure indicators and rollback capabilities

### 🧪 Testing
- **Integration Tests**: Optional but configurable test execution
- **Setup Validation**: Comprehensive setup action testing
- **Isolated Environments**: Each job gets a fresh Kind cluster
- **Component Testing**: Individual component validation

## 📊 Migration Benefits

| Aspect | Traditional Cronjobs | GitHub Actions |
|--------|---------------------|----------------|
| **Infrastructure** | Persistent GCP clusters | Ephemeral runners |
| **Cost** | 24/7 cluster costs | Pay-per-use model |
| **Security** | Cluster-based secrets | GitHub OIDC + attestations |
| **Maintenance** | Manual cluster updates | GitHub-managed updates |
| **Observability** | Custom logging setup | Built-in GitHub insights |
| **Debugging** | kubectl + cluster access | Web UI + downloadable logs |
| **Testing** | Complex test environments | Isolated Kind clusters |
| **Scaling** | Manual cluster scaling | Automatic runner scaling |

## 🔄 Migration Status

### ✅ Completed
- [x] All 5 Tekton core components migrated
- [x] All 5 plumbing components migrated  
- [x] Reusable action and template created
- [x] Security and best practices implemented
- [x] Comprehensive documentation
- [x] Testing workflows established

### 📋 Traditional Coverage Mapping

| Traditional Cronjob | GitHub Actions Workflow | Status |
|-------------------|------------------------|--------|
| `tekton/cronjobs/releases_azure/releases/pipeline-nightly/` | `nightly-pipeline.yml` | ✅ |
| `tekton/cronjobs/releases_azure/releases/triggers-nightly/` | `nightly-triggers.yml` | ✅ |
| `tekton/cronjobs/releases_azure/releases/dashboard-nightly/` | `nightly-dashboard.yml` | ✅ |
| `tekton/cronjobs/releases_azure/releases/chains-nightly/` | `nightly-chains.yml` | ✅ |
| `tekton/cronjobs/releases_azure/releases/operator-nightly/` | `nightly-operator.yml` | ✅ |
| `tekton/cronjobs/releases_azure/releases/add-pr-body-nightly/` | `nightly-plumbing-components.yml` | ✅ |
| `tekton/cronjobs/releases_azure/releases/add-pr-body-ci-nightly/` | `nightly-plumbing-components.yml` | ✅ |
| `tekton/cronjobs/releases_azure/releases/add-team-members-nightly/` | `nightly-plumbing-components.yml` | ✅ |
| `tekton/cronjobs/releases_azure/releases/pr-commenter-nightly/` | `nightly-plumbing-components.yml` | ✅ |
| `tekton/cronjobs/releases_azure/releases/pr-status-updater-nightly/` | `nightly-plumbing-components.yml` | ✅ |

## 🔗 See Also

- **[Testing Guide](TESTING.md)** - Comprehensive testing documentation
- **[Setup Action Documentation](.github/actions/setup-tekton/action.yml)** - Reusable action details
- **[Release Template](.github/workflows/nightly-release-template.yml)** - Reusable workflow template
- **[Migration History](https://github.com/tektoncd/plumbing/issues)** - Background and decision rationale 