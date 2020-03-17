apiVersion: v1
kind: Secret
metadata:
  name: tekton-storage
  namespace: tekton-pipelines
type: kubernetes.io/opaque
stringData:
  boto-config: |
    [Credentials]
    aws_access_key_id = __MINIO_ACCESS_KEY__
    aws_secret_access_key = __MINIO_SECRET_KEY__
    [s3]
    host = s3.eu-geo.svc.cluster.local
    use-sigv4 = True
    [Boto]
    https_validate_certificates = True
    [GSUtil]
    prefer_api = xml
