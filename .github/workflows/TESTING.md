# Testing Guide: GitHub Actions Nightly Releases

This guide covers how to test all the new GitHub Actions workflows in your fork.

## 🚀 **Quick Start**

### **1. Fork Setup**
```bash
# Fork the repository
git clone https://github.com/YOUR_USERNAME/plumbing.git
cd plumbing

# Enable Actions in fork settings:
# Settings → Actions → General → Allow all actions
```

### **2. Container Registry Setup**
```bash
# Login to GitHub Container Registry
echo $GITHUB_TOKEN | docker login ghcr.io -u YOUR_USERNAME --password-stdin

# Or create a Personal Access Token with packages:write scope
```

## 📋 **Complete Workflow Testing Matrix**

### **Tekton Core Projects**

| Workflow | Command | Expected Output |
|----------|---------|-----------------|
| **Pipeline** | `gh workflow run nightly-pipeline.yml -f run-tests=false` | `ghcr.io/tektoncd/pipeline:vYYYYMMDD-{sha}` |
| **Triggers** | `gh workflow run nightly-triggers.yml -f run-tests=false` | `ghcr.io/tektoncd/triggers:vYYYYMMDD-{sha}` |
| **Dashboard** | `gh workflow run nightly-dashboard.yml -f run-tests=false` | `ghcr.io/tektoncd/dashboard:vYYYYMMDD-{sha}` |
| **Chains** | `gh workflow run nightly-chains.yml -f run-tests=false` | `ghcr.io/tektoncd/chains:vYYYYMMDD-{sha}` |
| **Operator** | `gh workflow run nightly-operator.yml -f run-tests=false` | `ghcr.io/tektoncd/operator:vYYYYMMDD-{sha}` |

### **Plumbing Components**

| Component | Individual Test | Registry Output |
|-----------|----------------|-----------------|
| **add-pr-body** | `gh workflow run nightly-plumbing-components.yml -f components="add-pr-body"` | `ghcr.io/tektoncd/plumbing/interceptors/add-pr-body:vYYYYMMDD-{sha}` |
| **add-pr-body-ci** | `gh workflow run nightly-plumbing-components.yml -f components="add-pr-body-ci"` | `ghcr.io/tektoncd/plumbing/cluster-interceptors/add-pr-body:vYYYYMMDD-{sha}` |
| **add-team-members** | `gh workflow run nightly-plumbing-components.yml -f components="add-team-members"` | `ghcr.io/tektoncd/plumbing/interceptors/add-team-members:vYYYYMMDD-{sha}` |
| **pr-commenter** | `gh workflow run nightly-plumbing-components.yml -f components="pr-commenter"` | `ghcr.io/tektoncd/plumbing/custom-tasks/pr-commenter:vYYYYMMDD-{sha}` |
| **pr-status-updater** | `gh workflow run nightly-plumbing-components.yml -f components="pr-status-updater"` | `ghcr.io/tektoncd/plumbing/custom-tasks/pr-status-updater:vYYYYMMDD-{sha}` |

## 🧪 **Testing Scenarios**

### **Scenario 1: Single Project Test**
```bash
# Test one Tekton project
gh workflow run nightly-pipeline.yml \
  --repo YOUR_USERNAME/plumbing \
  -f run-tests=false

# Monitor progress
gh run list --workflow=nightly-pipeline.yml --limit=1
gh run view --log  # View logs of latest run
```

### **Scenario 2: Plumbing Components Test**
```bash
# Test all plumbing components
gh workflow run nightly-plumbing-components.yml \
  --repo YOUR_USERNAME/plumbing \
  -f components="all" \
  -f run-tests=false

# Test specific components only
gh workflow run nightly-plumbing-components.yml \
  --repo YOUR_USERNAME/plumbing \
  -f components="add-pr-body,pr-commenter"
```

### **Scenario 3: Full System Test**
```bash
# Run all workflows in sequence (to avoid resource conflicts)
for workflow in nightly-pipeline nightly-triggers nightly-dashboard nightly-chains nightly-operator; do
  echo "Testing $workflow..."
  gh workflow run ${workflow}.yml --repo YOUR_USERNAME/plumbing -f run-tests=false
  sleep 60  # Wait between runs
done

# Test plumbing components
gh workflow run nightly-plumbing-components.yml --repo YOUR_USERNAME/plumbing -f components="all"
```

### **Scenario 4: Fork-Specific Configuration**
```bash
# Edit workflows to use your fork's registry
sed -i 's/tektoncd/YOUR_USERNAME/g' .github/workflows/nightly-*.yml

# Commit and test
git add .github/workflows/
git commit -m "Update registry paths for fork testing"
git push

# Run test
gh workflow run nightly-pipeline.yml -f run-tests=false
```

## 🔍 **Verification Steps**

### **1. Check Workflow Status**
```bash
# List recent runs
gh run list --limit=10

# View specific run details
gh run view RUN_ID

# Download artifacts
gh run download RUN_ID
```

### **2. Verify Container Images**
```bash
# Check if images were published
docker pull ghcr.io/tektoncd/pipeline:vYYYYMMDD-{sha}

# List your fork's packages
gh api user/packages?package_type=container

# View package details
gh api users/YOUR_USERNAME/packages/container/pipeline
```

### **3. Validate Tekton Resources**
```bash
# If you have a test cluster, verify the release works
kubectl apply -f https://github.com/YOUR_USERNAME/plumbing/releases/download/vYYYYMMDD-{sha}/pipeline-release.yaml
```

## 🐛 **Debugging Common Issues**

### **Permission Errors**
```yaml
# Ensure your fork has correct permissions in Settings → Actions → General
permissions:
  id-token: write
  contents: read
  attestations: write
  packages: write
```

### **Registry Access Issues**
```bash
# Create GitHub token with packages:write
gh auth refresh -s write:packages

# Or use a Personal Access Token
export GITHUB_TOKEN=ghp_your_token_here
```

### **Resource Constraints**
```bash
# If workflows timeout, reduce resource usage
# Edit workflow timeout from 180 to 60 minutes
timeout-minutes: 60
```

### **Kind Cluster Issues**
```bash
# If Kind fails to start, check Docker daemon
docker version

# Check available resources
df -h
free -m
```

## 📊 **Success Criteria**

A successful test run should produce:

### **✅ Expected Outputs**
- [ ] Workflow completes successfully (green checkmark)
- [ ] Container image published to registry
- [ ] Tekton release YAML generated
- [ ] Pipeline logs available in artifacts
- [ ] No timeout or resource errors

### **✅ Artifact Verification**
- [ ] Download and inspect pipeline logs
- [ ] Verify container image tags match expected format
- [ ] Check release information is correct
- [ ] Validate multi-architecture support (if enabled)

### **✅ Performance Benchmarks**
- [ ] Total runtime < 60 minutes for basic test
- [ ] Cluster setup < 10 minutes
- [ ] Pipeline execution matches traditional system timing
- [ ] Resource cleanup completes properly

## 🎯 **Testing Checklist**

Before declaring workflows production-ready:

### **Individual Components**
- [ ] Test each Tekton project workflow individually
- [ ] Test each plumbing component individually
- [ ] Verify all container images are published correctly
- [ ] Check that release artifacts are properly formatted

### **Integration Testing**
- [ ] Run multiple workflows in parallel (if resources allow)
- [ ] Test with different input parameters
- [ ] Verify failure handling and cleanup
- [ ] Test manual triggers via GitHub UI

### **Production Validation**
- [ ] Deploy generated releases to test cluster
- [ ] Compare artifacts with traditional cronjob outputs
- [ ] Verify signing and attestations work correctly
- [ ] Test rollback procedures if needed

## 🔄 **Continuous Validation**

### **Automated Testing**
```yaml
# Consider adding a weekly validation workflow
name: Weekly Release Validation
on:
  schedule:
    - cron: '0 0 * * 0'  # Weekly on Sunday
  workflow_dispatch:

jobs:
  validate-all:
    runs-on: ubuntu-latest
    steps:
      - name: Test all workflows
        run: |
          # Script to trigger and validate all workflows
```

### **Monitoring**
- Set up GitHub notifications for workflow failures
- Monitor container registry for published images  
- Track workflow execution times and success rates
- Compare output quality with traditional system

This comprehensive testing approach ensures the GitHub Actions workflows can fully replace the traditional Tekton cronjob system. 