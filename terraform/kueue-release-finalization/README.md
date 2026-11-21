# Kueue release approval environment

This standalone module provisions `kueue-release-finalization` in
`tektoncd/tekton-kueue`. It has separate state and does not add required inputs
to the existing organization-wide branch-protection automation.

Approval is required from the existing `tekton-kueue-maintainers` team.
Self-review and administrator bypass are disabled. Only the branch `main`
may deploy; a tag named `main` is not allowed.

## Deployment prerequisites

An authorized infrastructure operator must supply a GitHub credential able to
manage repository environments and the required
`TF_VAR_kueue_release_signing_public_key`. Run this module from the approved
infrastructure runner with access to the Kubernetes state backend. Review its
plan before applying; this PR does not deploy it.

Obtain the public key from independently managed release infrastructure,
confirm its fingerprint with the maintainers, parse it with
`openssl pkey -pubin -in <key-file> -noout`, and verify a known release
signature. Terraform checks PEM shape and publishes it as the environment
variable `RELEASE_SIGNING_PUBLIC_KEY`; the finalizer also checks cryptographic
validity.

Do not obtain trusted key material from the candidate being verified. The
historical `tektoncd/chains/tekton.pub` (DER SHA-256
`ef0580597d0c3b7ff3ec827d6a02412ae4afae6ee3eac61f043ace8fab0302e2`)
does not match the OCI `v0.4.3` signer. No guessed or candidate-supplied key
is bundled here.

For rotation, independently authenticate the replacement key, verify a known
signature, update the required input, and deploy the variable change before
finalizing releases signed with it. Retain historical public keys for old
releases instead of silently changing their trust.

## Credentials and activation

The Kueue workflow uses the existing credential names `CHATOPS_TOKEN`,
`GHCR_TOKEN`, `OCI_API_KEY`, `OCI_FINGERPRINT`, `OCI_TENANCY_OCID`,
`OCI_USER_OCID`, and `OCI_REGION`. Names do not establish ownership or scopes:
the workflow must verify GitHub write tokens belong to `tekton-robot`.
Use environment-scoped overrides if existing organization credentials are
unsuitable. Limit them to this repository, its image package, and the release
storage paths; keep values out of source and Terraform inputs.

This module does not create or copy credentials. Before enabling the finalizer,
read back the environment's required reviewers, self-review and bypass settings,
main-only branch policy, and public-key variable. Merging a workflow alone can
create an unprotected environment; the finalizer must reject that state.

No finalization, release publication, or infrastructure apply is performed by
these configuration files.
