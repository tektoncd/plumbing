#!/bin/bash
set -eu -o pipefail

declare TEKTON_PIPELINE_VERSION TEKTON_TRIGGERS_VERSION TEKTON_DASHBOARD_VERSION

# This script deploys Tekton on a local kind cluster
# It creates a kind cluster and deploys pipeline, triggers and dashboard

# Prerequisites:
# - go 1.14+
# - docker (recommended 8GB memory config)
# - kind

# Notes:
# - Latest versions will be installed if not specified
# - If a kind cluster named "tekton" already exists this will fail
# - Local access to the dashboard requires port 9097 to be locally available

get_latest_release() {
  curl --silent "https://api.github.com/repos/$1/releases/latest" | # Get latest release from GitHub api
    grep '"tag_name":' |                                            # Get tag line
    sed -E 's/.*"([^"]+)".*/\1/'                                    # Pluck JSON value
}

# Read command line options
while getopts ":p:t:d:" opt; do
  case ${opt} in
    p )
      TEKTON_PIPELINE_VERSION=$OPTARG
      ;;
    t )
      TEKTON_TRIGGERS_VERSION=$OPTARG
      ;;
    d )
      TEKTON_DASHBOARD_VERSION=$OPTARG
      ;;
    \? )
      echo "Invalid option: $OPTARG" 1>&2
      echo 1>&2
      echo "Usage:  tekton_in_kind.sh [-p pipeline-version -t triggers-version -d dashboard-version]"
      ;;
    : )
      echo "Invalid option: $OPTARG requires an argument" 1>&2
      ;;
  esac
done
shift $((OPTIND -1))

# Check and defaults input params
if [ -z "$TEKTON_PIPELINE_VERSION" ]; then
  TEKTON_PIPELINE_VERSION=$(get_latest_release tektoncd/pipeline)
fi
if [ -z "$TEKTON_TRIGGERS_VERSION" ]; then
  TEKTON_TRIGGERS_VERSION=$(get_latest_release tektoncd/triggers)
fi
if [ -z "$TEKTON_DASHBOARD_VERSION" ]; then
  TEKTON_DASHBOARD_VERSION=$(get_latest_release tektoncd/dashboard)
fi

# Setup the kind cluster
kind create cluster --name tekton

# Install Tekton
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/previous/${TEKTON_PIPELINE_VERSION}/release.yaml
kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/previous/${TEKTON_TRIGGERS_VERSION}/release.yaml 
kubectl apply -f https://github.com/tektoncd/dashboard/releases/download/${TEKTON_DASHBOARD_VERSION}/tekton-dashboard-release.yaml

# Wait just a bit, to wait for the service to be running
sleep 10
kubectl port-forward service/tekton-dashboard -n tekton-pipelines 9097:9097 &> kind-tekton-dashboard.log &
echo “Tekton Dashboard available at http://localhost:9097”