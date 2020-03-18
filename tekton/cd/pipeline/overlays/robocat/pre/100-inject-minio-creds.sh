#!/bin/bash
#
# Grab Minio credentials and injects them in the storage secret used by
# Tekton Pipeline
set -exo pipefail

echo "=== Start Pre-script Inject Minio Creds"
BASE_DIR="$( cd "$( dirname "$0" )" >/dev/null 2>&1 && pwd )"

MINIO_NAMESPACE=${MINIO_NAMESPACE:-eu-geo}
MINIO_SECRET=${MINIO_SECRET:-s3}

MINIO_ACCESS_KEY=$(kubectl get -n $MINIO_NAMESPACE secret/$MINIO_SECRET \
  -o jsonpath='{ .data.accesskey }' | base64 -d)
MINIO_SECRET_KEY=$(kubectl get -n $MINIO_NAMESPACE secret/$MINIO_SECRET \
  -o jsonpath='{ .data.secretkey }' | base64 -d)

sed -e 's,__MINIO_ACCESS_KEY__,'$MINIO_ACCESS_KEY',g' \
  -e 's,__MINIO_SECRET_KEY__/,'$MINIO_SECRET_KEY',g' \
  "${BASE_DIR}/tekton-storage-secret.yaml.tpl" > "${BASE_DIR}/../tekton-storage-secret.yaml"

echo "=== End Pre-script Inject Minio Creds"
