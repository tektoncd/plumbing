# Signing Releases

Any Tekton releases run in the [dogfooding](dogfooding.md) cluster can be automatically signed by [Tekton Chains](https://github.com/tektoncd/chains), which is already set up in the cluster.
Users can then verify released images with the [Tekton public key](https://raw.githubusercontent.com/tektoncd/chains/main/tekton.pub) before deployment. 
Tekton Chains can also automatically generate build provenance for the release and store it in the Rekor transparency log, where users can easily find build provenance for an image based on the SHA256 digest.


## Configuring Released Images to be Signed
Tekton Chains will sign all images built by a `TaskRun` in the [dogfooding](dogfooding.md) cluster that have at least one of the following properties:
* The image is specified as an output Pipeline Resource in the TaskRun
* The image name, qualified by digest, is included as a Result called `IMAGES`

For more details on how Tekton Chains extracts built images from TaskRuns, see [Chains type hinting docs](https://github.com/tektoncd/chains/blob/main/docs/config.md#chains-type-hinting).
If Chains can recognize that an OCI image was built, it will try to sign it.

To see an example of how to configure the `IMAGES` result with `ko`, see the [Tekton Chains release TaskRun](https://github.com/tektoncd/chains/blob/main/release/publish.yaml#L35).


## Generating Build Provenance for a Release
Tekton Chains can automatically generate build provenance for a release and upload it to the Rekor transparency log.
Users can easily query the log and find this build provenance.

To enable build provenance generation for your release, add this annotation to your `Task` or `Pipeline`:

```yaml
annotations:
    chains.tekton.dev/transparency-upload: "true"
```

Users can then easily find provenance for an image with the [rekor-cli](https://github.com/sigstore/rekor/releases/) tool:

```
$ rekor-cli search –sha sha256:1189a2207be3e93e91aaeff323dc2804576f188527afe3cc2e9a9a0c688344df
Found matching entries (listed by UUID):
60a1e4f9c78ae76b2b2a06745340b8ed74c8b2ea2c124b8520ba319d03957906
3873a54462deab6320d1cac993b31b36bb28ff5c2f0d16993909b61907235ec6

$ rekor-cli get --uuid 3873a54462deab6320d1cac993b31b36bb28ff5c2f0d16993909b61907235ec6 --format json | jq -r .Attestation | base64 --decode | jq

<your attestation here>
```


## Verifying a Signed Image
Users can verify an image with [cosign](https://github.com/sigstore/cosign) and the Tekton public key:

```
$ cat tekton.pub
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEnLNw3RYx9xQjXbUEw8vonX3U4+tB
kPnJq+zt386SCoG0ewIH5MB8+GjIDGArUULSDfjfM31Eae/71kavAUI0OA==
-----END PUBLIC KEY-----

$ cosign verify -key tekton.pub YOUR-IMAGE-NAME

<verification here>
```
