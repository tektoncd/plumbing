# Container Images for Tekton infrastructure

This folder contains various container images used by Tekton infrastructure to
run Tekton's own CI/CD.

## Wolfi images (experimental)

Some directories include experimental support for
[Wolfi](https://github.com/wolfi-dev/) images built via
[apko](https://github.com/chainguard-dev/apko) + terraform.

These images are built declaratively from Wolfi packages and are automatically
signed + attested during publishing.

To build these images, run:

```sh
TF_VAR_target_repository=ttl.sh/path/to/registry terraform apply
```

To build a single image (for example, `ko-gcloud`):

```sh
TF_VAR_target_repository=ttl.sh/path/to/registry terraform apply -target=module.ko-gcloud
```

### Signing and attestations

If you wish to sign/attest the image locally (optional for development, but
terraform will output a warning), you can enable it by setting
`TF_COSIGN_LOCAL=1`:

```sh
TF_COSIGN_LOCAL=1 TF_VAR_target_repository=ttl.sh/path/to/registry terraform apply -target=module.ko-gcloud
```
