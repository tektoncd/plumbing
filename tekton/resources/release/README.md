# Tekton Resources for Release Automation

This folder contains various Tekton resource used to automate Tekton's own
release management. These components are written so that they can be used by
all the projects in the `tektoncd` GitHub org.
The core release pipelines are still owned by the specific project.

## Tasks and Pipelines

### OCI Release Preflight

[`prerelease-checks-oci`](./base/prerelease_checks_oci.yaml) is a read-only
gate for a new release, not a recovery workflow. It requires:

- `package`: `github.com/OWNER/REPO` (default `github.com/tektoncd/pipeline`).
- `versionTag`: a release version such as `v0.4.3` or `v0.4.3-rc.1`.
- `releaseBucket`: `<bucket-name>/<project-name>`; all three versioned objects
  `release.yaml`, `release.notags.yaml`, and `SHA256SUMS` are checked under
  `<project-name>/previous/<versionTag>/`.
- `ociNamespace`: the OCI namespace, or `""` (default) to discover it.
- `imageRepository`: `""` by default, skipping the image check. When supplied,
  it must be a public `ghcr.io/OWNER/NAME` repository without a tag or digest,
  and `versionTag` must also be a valid container tag. Callers must explicitly
  pass their image repository; it is not inferred from `package`.
- Workspaces `source-to-release` and `oci-credentials`, with the existing OCI
  files `fingerprint`, `tenancy_ocid`, `user_ocid`, `region`, and `oci_api_key.pem`.
- Namespace Secret `github-token`, key `GITHUB_TOKEN`, authenticating as
  `tekton-robot` with repository push permission so drafts are visible.
  A missing Secret/key produces a script error rather than a pending Pod.

The Task first verifies the GitHub identity and repository access, then checks
the exact tag reference and every page of authenticated releases, including
drafts. When enabled, the GHCR check obtains a public pull token and probes
the version manifest. The OCI check verifies bucket access before checking the
release objects, using temporary private credential files cleaned up on success
and script errors.

An existing tag, release (published or draft), manifest, or release object blocks
the Task. Only an HTTP 404 on the exact tag lookup, a manifest 404 with a
structured `MANIFEST_UNKNOWN` error, or a structured
OCI object error with status 404 and code `ObjectNotFound` or `NotFound` after
successful bucket access, establishes absence. Authentication, authorization,
network, other HTTP, CLI, and parse errors fail the Task; there is no anonymous
GitHub fallback. Credentials and response bodies are not logged.

If state already exists, stop and use the project's release recovery process;
this Task does not delete or repair published state. Kueue caller wiring,
shared-resource pinning, and recovery automation are separate changes.

Run the focused deterministic tests (Python with `pytest` and the repository's
`requirements.txt` installed):

```shell
python3 -m pytest -q tekton/resources/release/tests/test_prerelease_checks_oci.py
```

### Install Tekton Release

The task `install-tekton-release` installs a release of a Tekton project from the YAML file in the
release bucket.

Inputs are:

- Param `projectName`: the name of the project (pipeline, trigger,
dashboard, experimental)
- Param `version`: the version to be installed, e.g. "v0.7.0"
- Param `releaseBucket`. The release file
is expected to be at `<releaseBucket>/<projectName>/previous/<version>/release.yaml`
- Workspace `targetCluster` to be bound to a secret that holds the kubeconfig of the target cluster

The release task can use a `kustomize` overlay if available. The name of the
overlay folder is specified via the `environment` parameter.
The overlay folder must contain a `kustomize.yaml` configuration file. It may
also contain a `pre` folder. Any `*.sh` script found in the folder will be
executed before the release is installed.

### Save Release Logs

The pipeline `save-release-logs` fetches the logs from a release pipelines
and stores them in the release bucket along with the release YAML.
This pipeline is triggered automatically when a release pipeline is executed,
specifically when the "publish" task of the release is executed successfully.
The `tekton-events` event listener receives the CloudEvent, and triggers
the `save-release-logs` with the correct credentials to store the logs
in the release bucket, either the main one or the nightly one.

### Create Draft Release

The pipeline `release-draft` calculates the list of PRs merged between the
previous release and a specified revision. It also builds a list of authors and
uses PRs and authors to build a draft new release in GitHub. It also attaches
the `release.yaml` from the release bucket to the GitHub release.

This pipeline must be executed after the release images and YAML have been
produced. Running this task multiple times will create multiple drafts; old
drafts have to be pruned manually when needed.

Once the draft release is created, the release manager needs to edit the draft,
arranging PRs in the right category, highlighting important changes and creating
a release tag line.

Parameters are: `package`, `release-name`, `release-tag`, `previous-release-tag`
, `git-revision`, `bucket` and `rekor-uuid`.

The OCI variant ([release-draft-oci](./base/github_release_oci.yaml)) also
accepts an optional `draft` parameter (default `"true"`, matching the
existing behavior above). Set it to `"false"` to publish the release
directly instead of creating a draft pre-release — useful for routine
releases (e.g. patch releases) that don't need manual curation. Either way,
the release title is always set to a clean `Tekton <Project> release
<version>` (with the release name quoted alongside it, only when one is
given), instead of being left blank for the release manager to fill in.

An example using `tkn`. Start defining a few environment variables, obtain the
REKOR_UUID and then run the pipeline:

```shell
export TEKTON_RELEASE_GIT_SHA=9c884fb3d3bf35c0a251936626f2ca9f17b5c183
export TEKTON_PACKAGE=tektoncd/pipeline
export TEKTON_VERSION=v0.9.0
export TEKTON_OLD_VERSION=v0.8.0
export TEKTON_RELEASE_NAME="Bengal Bender"

RELEASE_FILE=https://infra.tekton.dev/tekton-releases/pipeline/previous/${TEKTON_VERSION}/release.yaml
CONTROLLER_IMAGE_SHA=$(curl $RELEASE_FILE | egrep 'gcr.io.*controller' | cut -d'@' -f2)
REKOR_UUID=$(rekor-cli search --sha $CONTROLLER_IMAGE_SHA | grep -v Found | head -1)
echo -e "CONTROLLER_IMAGE_SHA: ${CONTROLLER_IMAGE_SHA}\nREKOR_UUID: ${REKOR_UUID}"

tkn pipeline start \
  --workspace name=shared,volumeClaimTemplateFile=workspace-template.yaml \
  --workspace name=credentials,secret=release-secret \
  -p package="${TEKTON_PACKAGE}" \
  -p git-revision="$TEKTON_RELEASE_GIT_SHA" \
  -p release-tag="${TEKTON_VERSION}" \
  -p previous-release-tag="${TEKTON_OLD_VERSION}" \
  -p release-name="${TEKTON_RELEASE_NAME}" \
  -p bucket="gs://tekton-releases/pipeline" \
  -p rekor-uuid="$REKOR_UUID" \
  release-draft
```

#### Using Oracle Cloud Storage

To draft a release by downloading the release manifests directly from Oracle Cloud Storage buckets, use the [release-draft-oci](./base/github_release_oci.yaml) pipeline instead.

If Oracle Cloud login credentials are managed in a secret called `oci-release-secret`, then set the workspace secret addordingly.

Please note the additional inputs `--pod-template` and the new parameter `repo-name` that are specific to release pipelines with oci related tags.

Create a pod template file:

```shell
cat <<EOF > tekton/pod-template.yaml
securityContext:
  fsGroup: 65532
  runAsUser: 65532
  runAsNonRoot: true
EOF
```
```shell
export TEKTON_RELEASE_GIT_SHA=9c884fb3d3bf35c0a251936626f2ca9f17b5c183
export TEKTON_PACKAGE=tektoncd/pipeline
export TEKTON_VERSION=v0.9.0
export TEKTON_OLD_VERSION=v0.8.0
export TEKTON_RELEASE_NAME="Bengal Bender"
export TEKON_REPO_NAME=pipeline

RELEASE_FILE=https://infra.tekton.dev/tekton-releases/pipeline/previous/${TEKTON_VERSION}/release.yaml
CONTROLLER_IMAGE_SHA=$(curl -L $RELEASE_FILE | egrep 'gcr.io.*controller' | cut -d'@' -f2)
REKOR_UUID=$(rekor-cli search --sha $CONTROLLER_IMAGE_SHA | grep -v Found | head -1)
echo -e "CONTROLLER_IMAGE_SHA: ${CONTROLLER_IMAGE_SHA}\nREKOR_UUID: ${REKOR_UUID}"

tkn pipeline start \
  --workspace name=shared,volumeClaimTemplateFile=workspace-template.yaml \
  --workspace name=credentials,secret=oci-release-secret \
  --pod-template pod-template.yaml \
  -p package="${TEKTON_PACKAGE}" \
  -p git-revision="$TEKTON_RELEASE_GIT_SHA" \
  -p release-tag="${TEKTON_VERSION}" \
  -p previous-release-tag="${TEKTON_OLD_VERSION}" \
  -p release-name="${TEKTON_RELEASE_NAME}" \
  -p repo-name="${TEKTON_REPO_NAME}" \
  -p bucket="tekton-releases" \
  -p rekor-uuid="$REKOR_UUID" \
  release-draft-oci
```

