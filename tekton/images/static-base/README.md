# Tekton Static Base Image

Multi-arch static base image for all Tekton Go binaries. Built with
[apko](https://github.com/chainguard-dev/apko) from Alpine packages.

## Architectures

amd64, arm64, s390x, ppc64le

## Contents (~300KB per arch)

| Content | Why |
|---------|-----|
| CA certificates | TLS connections |
| Timezone data | `time.LoadLocation()` |
| `/etc/passwd`, `/etc/group` | nonroot user (UID 65532) |
| `/etc/nsswitch.conf` | DNS resolution |

## Build locally

```bash
# Install apko: go install chainguard.dev/apko@latest
apko build apko.yaml tekton-static-base:latest output.tar
```

## Publish

```bash
apko publish apko.yaml ghcr.io/tektoncd/plumbing/static-base:latest
```

## CI

This image is built and published by the repository's `ci` workflow
(`.github/workflows/ci.yaml`), like every other image under `tekton/images/`.
The `build-image` job detects the `apko.yaml` file and builds with apko instead
of a `Dockerfile`. It validates the build on pull requests and publishes to
`ghcr.io/tektoncd/plumbing/static-base` on push to `main` and on the daily
schedule.

## Consumers

- `tektoncd/pipeline` (.ko.yaml defaultBaseImage)
- `tektoncd/triggers` (.ko.yaml defaultBaseImage)
- `tektoncd/chains` (.ko.yaml defaultBaseImage)
- `tektoncd/results` (.ko.yaml defaultBaseImage)

## Background

See [tektoncd/pipeline#9557](https://github.com/tektoncd/pipeline/issues/9557)
for the full proposal. The previous base image (`cgr.dev/chainguard/static`)
was pinned to an EOL Alpine 3.18 digest since November 2023, and newer
Chainguard free-tier images dropped s390x and ppc64le support.
