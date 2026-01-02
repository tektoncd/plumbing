#!/usr/bin/env bash
set -e -o pipefail

declare TEKTON_PIPELINE_VERSION TEKTON_TRIGGERS_VERSION TEKTON_DASHBOARD_VERSION CONTAINER_RUNTIME

# This script deploys Tekton on a local kind cluster
# It creates a kind cluster and deploys pipeline, triggers and dashboard

# Prerequisites:
# - go 1.14+
# - podman or docker (recommended 8GB memory config)
# - kind

# Notes:
# - Latest versions will be installed if not specified
# - If a kind cluster named "tekton" already exists this will fail
# - Local access to the dashboard requires port 9097 to be locally available

# if script is ran with source, i.e. `. tekton_in_kind.sh` - a warning is displayed 
# and suggested to run it in it's own shell process. 
if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
  echo "This script is not intended to be sourced. Please run it as ./tekton_in_kind.sh"
  return 1
fi

get_latest_release() {
  curl --silent "https://api.github.com/repos/$1/releases/latest" | # Get latest release from GitHub api
    grep '"tag_name":' |                                            # Get tag line
    sed -E 's/.*"([^"]+)".*/\1/'                                    # Pluck JSON value
}

info() {
  echo -e "[\e[93mINFO\e[0m] $1"
}

# Read command line options
while getopts ":c:p:t:d:k" opt; do
  case ${opt} in
    c )
      CLUSTER_NAME=$OPTARG
      ;;
    p )
      TEKTON_PIPELINE_VERSION=$OPTARG
      ;;
    t )
      TEKTON_TRIGGERS_VERSION=$OPTARG
      ;;
    d )
      TEKTON_DASHBOARD_VERSION=$OPTARG
      ;;
    k )
      CONTAINER_RUNTIME="docker"
      ;;
    \? )
      echo "Invalid option: $OPTARG" 1>&2
      echo 1>&2
      echo "Usage: tekton_in_kind.sh [-c cluster-name -p pipeline-version -t triggers-version -d dashboard-version [-k]"
      ;;
    : )
      echo "Invalid option: $OPTARG requires an argument" 1>&2
      ;;
  esac
done
shift $((OPTIND -1))

# Check and set default input params
export KIND_CLUSTER_NAME=${CLUSTER_NAME:-"tekton"}

if [ -z "$TEKTON_PIPELINE_VERSION" ]; then
  TEKTON_PIPELINE_VERSION=$(get_latest_release tektoncd/pipeline)
fi
if [ -z "$TEKTON_TRIGGERS_VERSION" ]; then
  TEKTON_TRIGGERS_VERSION=$(get_latest_release tektoncd/triggers)
fi
if [ -z "$TEKTON_DASHBOARD_VERSION" ]; then
  TEKTON_DASHBOARD_VERSION=$(get_latest_release tektoncd/dashboard)
fi
if [ -z "$CONTAINER_RUNTIME" ]; then
  CONTAINER_RUNTIME="podman"
fi

info "Using container runtime: $CONTAINER_RUNTIME"

info "Checking if registry exists..."
reg_name='kind-registry'
reg_port='5000'
running="$(${CONTAINER_RUNTIME} inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
if [ "${running}" != 'true' ]; then
  info "Registry does not exist, creating..."
  # It may exist and not be running, so cleanup just in case
  "$CONTAINER_RUNTIME" rm "${reg_name}" 2> /dev/null || true
  "$CONTAINER_RUNTIME" run \
    -d \
    --restart=always \
    -p "${reg_port}:5000" \
    --network bridge \
    --name "${reg_name}" \
    registry:2
fi
info "Registry ready..."

info "Checking if kind cluster '$KIND_CLUSTER_NAME' exists..."
# Tell kind to use the same container runtime as the registry
# https://kind.sigs.k8s.io/docs/user/quick-start/#creating-a-cluster
export KIND_EXPERIMENTAL_PROVIDER=$CONTAINER_RUNTIME
running_cluster=$(kind get clusters | grep "$KIND_CLUSTER_NAME" || true)
if [ "${running_cluster}" != "$KIND_CLUSTER_NAME" ]; then
  info "Kind cluster '$KIND_CLUSTER_NAME' does not exist, creating with the local registry enabled in containerd..."
  cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
  - role: worker
  - role: worker
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${reg_port}"]
    endpoint = ["http://${reg_name}:${reg_port}"]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."${reg_name}:${reg_port}"]
    endpoint = ["http://${reg_name}:${reg_port}"]
EOF
  info "Waiting for the nodes to be ready..."
  kubectl wait --for=condition=ready node --all --timeout=600s
fi
info "Kind cluster '$KIND_CLUSTER_NAME' ready..."

info "Connect the registry to the cluster network..."
# (the network may already be connected)
"$CONTAINER_RUNTIME" network connect "kind" "${reg_name}" || true
info "Connection established..."

info "Install Tekton Pipeline, Triggers and Dashboard..."
kubectl apply -f https://infra.tekton.dev/tekton-releases/pipeline/previous/${TEKTON_PIPELINE_VERSION}/release.yaml
kubectl apply -f https://infra.tekton.dev/tekton-releases/triggers/previous/${TEKTON_TRIGGERS_VERSION}/release.yaml
kubectl wait --for=condition=Established --timeout=30s crds/clusterinterceptors.triggers.tekton.dev || true # Starting from triggers v0.13
kubectl apply -f https://infra.tekton.dev/tekton-releases/triggers/previous/${TEKTON_TRIGGERS_VERSION}/interceptors.yaml || true
kubectl apply -f https://infra.tekton.dev/tekton-releases/dashboard/previous/${TEKTON_DASHBOARD_VERSION}/release-full.yaml

info "Wait until all pods are ready..."
kubectl wait -n tekton-pipelines --for=condition=ready pods --all --timeout=600s
kubectl port-forward service/tekton-dashboard -n tekton-pipelines 9097:9097 &> kind-tekton-dashboard.log &
info "Tekton Dashboard available at http://localhost:9097"
