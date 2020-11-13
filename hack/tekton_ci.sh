#!/bin/bash 
set -e -o pipefail

declare GITHUB_USER GITHUB_TOKEN GITHUB_ORG GITHUB_REPO 

# This script deploys Tekton CI on a local kind cluster
# It create GitHub webhook and deploys Tekton CI pipeline and tasks

USAGE="Usage:  tekton_ci.sh -u <github-user> -t <github-token> -o <github-org> -r <github-repo> 

Options:
 -u <github-user>         Your GitHub username
 -t <github-token>        Your GitHub token
 -o <github-org>          The org or user where your fork is hosted
 -r <github-repo>         The name of the fork, typically \"plumbing\"
"

# Read command line options
while getopts ":u:t:o:r:" opt; do
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


# Deploy plumbing resources. Run from the root of your local clone
# The sed command injects your fork GitHub org in the CEL filters
kustomize build tekton/ci | \
  sed -E 's/tektoncd(\/p[^i]+|\(|\/'\'')/'$GITHUB_ORG'\1/g' | \
  kubectl create -f -

# Create the secret used by the GitHub interceptor
kubectl create secret generic ci-webhook -n tektonci --from-literal=secret=$GITHUB_SECRET

# Wait until all pods are ready
kubectl wait -n tektonci --for=condition=ready pods --all --timeout=120s

# Expose the event listener via Smee
kubectl port-forward service/el-tekton-ci-webhook -n tektonci 9999:8080 &> ./el-tekton-ci-webhook-pf.log &
smee --target http://127.0.0.1:9999/ &> ./smee.log &

# Wait smee target ready
sleep 5 
SMEE_TARGET=$(tail -1 ./smee.log | cut -d'/' -f3-)

# Install a Task to create the webhook, create a secret used by it
kubectl apply -f https://raw.githubusercontent.com/tektoncd/triggers/master/docs/getting-started/create-webhook.yaml

kubectl create secret generic github --from-literal=token=$GITHUB_TOKEN --from-literal=secret=$GITHUB_SECRET

# Setup the webhook in your fork that points to the smee service
tkn task start create-webhook -p ExternalDomain=$SMEE_TARGET -p GitHubUser=$GITHUB_USER -p GitHubRepo=$GITHUB_REPO -p GitHubOrg=$GITHUB_ORG -p GitHubSecretName=github -p GitHubAccessTokenKey=token -p GitHubSecretStringKey=secret
