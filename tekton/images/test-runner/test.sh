#!/usr/bin/env bash
set -x
echo This is a test script for plumbing to test test-runner

export KUBETEST_IN_DOCKER="true"

source ./scripts/e2e-tests.sh

initialize

kubectl version

kubectl get all --all-namespaces
