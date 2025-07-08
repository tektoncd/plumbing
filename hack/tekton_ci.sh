#!/usr/bin/env bash
set -e -o pipefail

declare GITHUB_USER GITHUB_TOKEN GITHUB_ORG GITHUB_REPO GITHUB_SECRET

# This script deploys Tekton CI on a local kind cluster
# It create GitHub webhook and deploys Tekton CI pipeline and tasks

USAGE="Usage:  tekton_ci.sh -u <github-user> -t <github-token> -o <github-org> -r <github-repo> [-s <github-secret>]

Options:
 -u <github-user>         Your GitHub username
 -t <github-token>        Your GitHub token
 -o <github-org>          The org or user where your fork is hosted
 -r <github-repo>         The name of the fork, typically \"plumbing\"
 -s <github-secret>       GitHub webhook secret (optional, will be generated if not provided)
"

# Function to check if required tools are available
check_dependencies() {
  local missing_tools=()
  
  for tool in kubectl kustomize smee tkn ko; do
    if ! command -v "$tool" &> /dev/null; then
      missing_tools+=("$tool")
    fi
  done
  
  # Check for container runtime (docker or podman)
  if ! command -v docker &> /dev/null && ! command -v podman &> /dev/null; then
    missing_tools+=("docker or podman")
  fi
  
  if [ ${#missing_tools[@]} -ne 0 ]; then
    echo "Error: Missing required tools: ${missing_tools[*]}"
    echo "Please install these tools before running this script."
    exit 1
  fi
}

# Function to generate a random secret
generate_secret() {
  openssl rand -hex 20
}

# Function to cleanup background processes
cleanup() {
  echo "Cleaning up background processes..."
  if [[ -n "$KUBECTL_PID" ]]; then
    kill "$KUBECTL_PID" 2>/dev/null || true
  fi
  if [[ -n "$SMEE_PID" ]]; then
    kill "$SMEE_PID" 2>/dev/null || true
  fi
  rm -f ./el-tekton-ci-pf.log ./smee.log
}

# Set up cleanup trap
trap cleanup EXIT

# Check dependencies first
check_dependencies

# Read command line options
while getopts ":u:t:o:r:s:" opt; do
  case ${opt} in
    u )
      GITHUB_USER=$OPTARG
      ;;
    t )
      GITHUB_TOKEN=$OPTARG
      ;;
    o )
      GITHUB_ORG=$OPTARG
      ;;
    r )
      GITHUB_REPO=$OPTARG
      ;;
    s )
      GITHUB_SECRET=$OPTARG
      ;;
    \? )
      echo "Invalid option: $OPTARG" 1>&2
      echo 1>&2
      echo "$USAGE"
      exit 1
      ;;
    : )
      echo "Invalid option: $OPTARG requires an argument" 1>&2
      exit 1
      ;;
  esac
done
shift $((OPTIND -1))

# Check input params
if [ -z "$GITHUB_USER" ] || [ -z "$GITHUB_TOKEN" ] || [ -z "$GITHUB_ORG" ] || [ -z "$GITHUB_REPO" ] ; then
  echo "Missing parameters"
  echo "$USAGE"
  exit 1
fi

# Generate GitHub secret if not provided
if [ -z "$GITHUB_SECRET" ]; then
  GITHUB_SECRET=$(generate_secret)
  echo "Generated GitHub webhook secret: $GITHUB_SECRET"
fi

echo "Deploying Tekton CI resources..."

# Deploy plumbing resources. Run from the root of your local clone
# The sed command injects your fork GitHub org in the CEL filters
if ! kustomize build tekton/ci | \
  sed -E "s/tektoncd(\/p[^i]+|\(|\/')/$GITHUB_ORG\1/g" | \
  kubectl create -f -; then
  echo "Error: Failed to deploy plumbing resources"
  exit 1
fi

# Install build-id cluster interceptor (required by CI resources)
echo "Installing build-id cluster interceptor..."

# Set environment variables for ko
export KO_DOCKER_REPO=ghcr.io/${GITHUB_ORG}/${GITHUB_REPO}

# login to ghcr.io (GitHub Container Registry) to allow ko to push the built container image there
echo "$GITHUB_TOKEN" | docker login ghcr.io -u "${GITHUB_USER}" --password-stdin

# create a secret to allow the cluster to pull the image when deploying
kubectl create secret docker-registry ghcr-creds \
  --docker-server=ghcr.io \
  --docker-username="$GITHUB_USER" \
  --docker-password="$GITHUB_TOKEN" \
  --namespace=tekton-ci

# Change to the build-id directory since ko needs to be run from where go.mod is located
pushd tekton/ci/cluster-interceptors/build-id > /dev/null

# Create service account and add image pull secret
kubectl apply -f config/100-serviceaccount.yaml --namespace=tekton-ci

# add the secret to the service account
kubectl patch serviceaccount build-id-bot \
  --namespace tekton-ci \
  --patch '{"imagePullSecrets": [{"name": "ghcr-creds"}]}'

# Apply build-id cluster interceptor configuration directly
echo "Building and applying build-id cluster interceptor..."
if ! ko apply -f config/ -- --namespace=tekton-ci; then
  echo "Error: Failed to install build-id cluster interceptor"
  popd > /dev/null
  exit 1
fi

popd > /dev/null

echo "Creating secrets..."

# Create the secret used by the GitHub interceptor
if ! kubectl create secret generic ci-webhook -n tekton-ci --from-literal=secret="$GITHUB_SECRET"; then
  echo "Error: Failed to create ci-webhook secret"
  exit 1
fi

echo "Waiting for pods to be ready..."

# Wait until all pods are ready
if ! kubectl wait -n tekton-ci --for=condition=ready pods --all --timeout=120s; then
  echo "Error: Pods failed to become ready within timeout"
  exit 1
fi

echo "Setting up port forwarding and smee..."

# Expose the event listener via Smee
kubectl port-forward service/el-tekton-ci -n tekton-ci 9999:8080 &> ./el-tekton-ci-pf.log &
KUBECTL_PID=$!

smee --target http://127.0.0.1:9999/ &> ./smee.log &
SMEE_PID=$!

# Wait smee target ready and extract URL more robustly
echo "Waiting for smee to start..."
sleep 5

# Wait for smee log to contain the URL
for i in {1..10}; do
  if [[ -f "./smee.log" ]] && grep -q "https://smee.io/" "./smee.log"; then
    break
  fi
  sleep 2
done

if [[ ! -f "./smee.log" ]] || ! grep -q "https://smee.io/" "./smee.log"; then
  echo "Error: Failed to get smee URL from log"
  exit 1
fi

SMEE_TARGET=$(tail -1 ./smee.log | cut -d'/' -f3-)

if [[ -z "$SMEE_TARGET" ]]; then
  echo "Error: Failed to extract smee target URL"
  exit 1
fi

echo "Smee target: $SMEE_TARGET"

echo "Installing webhook creation task..."

# Install a Task to create the webhook, create a secret used by it
if ! kubectl apply -f https://raw.githubusercontent.com/tektoncd/triggers/main/docs/getting-started/create-webhook.yaml; then
  echo "Error: Failed to install webhook creation task"
  exit 1
fi

if ! kubectl create secret generic github --from-literal=token="$GITHUB_TOKEN" --from-literal=secret="$GITHUB_SECRET"; then
  echo "Error: Failed to create github secret"
  exit 1
fi

echo "Creating webhook in GitHub repository..."

# Setup the webhook in your fork that points to the smee service
if ! tkn task start create-webhook \
  -p ExternalDomain="$SMEE_TARGET" \
  -p GitHubUser="$GITHUB_USER" \
  -p GitHubRepo="$GITHUB_REPO" \
  -p GitHubOrg="$GITHUB_ORG" \
  -p GitHubDomain="github.com" \
  -p WebhookEvents='[\"push\",\"pull_request\"]' \
  -p GitHubSecretName=github \
  -p GitHubAccessTokenKey=token \
  -p GitHubSecretStringKey=secret; then
  echo "Error: Failed to create webhook"
  exit 1
fi

echo "Tekton CI deployment completed successfully!"
echo "GitHub webhook secret: $GITHUB_SECRET"
echo "Smee URL: https://$SMEE_TARGET"
echo ""
echo "Background processes are running. Press Ctrl+C to stop them."

# Keep the script running to maintain the port-forward and smee processes
wait
