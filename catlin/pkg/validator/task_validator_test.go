// Copyright © 2020 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validator

import (
	"strings"
	"testing"

	"github.com/tektoncd/plumbing/catlin/pkg/parser"
	"gotest.tools/v3/assert"
)

const taskWithValidImageRef = `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: valid-image-refs
  labels:
    app.kubernetes.io/version: a,b,c
  annotations:
    tekton.dev/tags: a,b,c
    tekton.dev/pipelines.minVersion: "0.12"
    tekton.dev/displayName: My Example Task
spec:
  description: |-
    A summary of the resource

    A para about this valid task

  params:
  - name: img_ver
    type: string

  steps:
  - name: s1
    image: docker.io/foo:bar
  - name: s2
    image: docker.io/abc/foo:baz
  - name: s3
    image: quay.io/foo:bar
  - name: s4
    image: r.j3ss.co/clisp:1.0
  - name: s5
    image: gcr.io/foo/bar/baz:bar
  - name: s6
    image: abc.io/fedora:1.0@sha256:deadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33f
  - name: s7
    image: abc.io/xyz/fedora:v123@sha256:deadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33f
  - name: s8
    image: 172.16.3.4:3000/foo/bar:baz
  - name: s9
    image: abc.io/xyz/fedora:latest-0.2@sha256:deadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33f
  - name: s10
    image: abc.io/xyz/fedora:latest0.2@sha256:deadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33f
`

const taskWithInvalidImageRef = `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: invalid-image-refs
  labels:
    app.kubernetes.io/version: a,b,c
  annotations:
    tekton.dev/tags: a,b,c
    tekton.dev/pipelines.minVersion: "0.12"
    tekton.dev/displayName: My Example Task
spec:
  description: |-
    A summary of the resource

    A para about this valid task

  params:
  - name: foo
    type: string
  - name: version
    type: string

  steps:
  - name: s1
    image: ubuntu
  - name: s2
    image: api:latest
  - name: s3
    image: docker.io/foo
  - name: s4
    image: r.j3ss.co/clisp:latest
  - name: s5
    image: docker.io/abc/foo
  - name: s6
    image: gcr.io/foo/bar/baz:latest
  - name: s7
    image: abc.io/fedora@sha256:deadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33f
  - name: s8
    image: abc.io/fedora:latest@sha256:deadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33f
  - name: s9
    image: gcr.io/k8s-staging-boskos/boskosctl@sha256:a7fc984732
  - name: s10
    image: localhost:5000/foo:bar
  - name: s11
    image: $(input.params.foo)
  - name: s12
    image: docker.io/xyz/$(params.foo)
  - name: s13
    image: docker.io/abc/foo:$(params.version)
`

const taskWithEnvFromSecret = `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: invalid-env-or-envfrom
  labels:
    app.kubernetes.io/version: a,b,c
  annotations:
    tekton.dev/tags: a,b,c
    tekton.dev/pipelines.minVersion: "0.12"
    tekton.dev/displayName: My Example Task
spec:
  description: |-
    A summary of the resource

    A para about this valid task

  params:
  - name: secret
    type: string
  - name: configmap
    type: string

  steps:
  - name: s1
    image: docker.io/foo:bar
    env:
    - name: PASS
      valueFrom:
        secretKeyRef:
          name: $(params.secret)
          key: PASS
  - name: s2
    image: docker.io/foo:bar
    env:
    - name: USER
      valueFrom:
        configMapKeyRef:
          name: $(params.configmap)
          key: USER
  - name: s3
    image: docker.io/foo:bar
    envFrom:
    - secretRef:
        name: $(params.secret)
  - name: s4
    image: docker.io/foo:bar
    envFrom:
    - configMapRef:
        name: $(params.configmap)
`

const taskWithScriptUsingParams = `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: invalid-env-or-envfrom
  labels:
    app.kubernetes.io/version: a,b,c
  annotations:
    tekton.dev/tags: a,b,c
    tekton.dev/pipelines.minVersion: "0.12"
    tekton.dev/displayName: My Example Task
spec:
  description: foo
  params:
  - name: secret
    type: string
  steps:
  - name: s1
    image: docker.io/foo:bar
    script: |
      echo "$(params.secret)"
  - name: s2
    image: docker.io/foo:bar
    script: |
      #!/usr/bin/env python
      print("""$(params.secret)""")
`

func TestTaskValidator_ValidImageRef(t *testing.T) {

	r := strings.NewReader(taskWithValidImageRef)
	parser := parser.ForReader(r)

	res, err := parser.Parse()
	assert.NilError(t, err)

	v := ForKind(res)
	result := v.Validate()
	assert.Equal(t, 0, result.Errors)

	lints := result.Lints
	assert.Equal(t, 0, len(lints))

}

func TestTaskValidator_InvalidImageRef(t *testing.T) {

	r := strings.NewReader(taskWithInvalidImageRef)
	parser := parser.ForReader(r)

	res, err := parser.Parse()
	assert.NilError(t, err)

	v := ForKind(res)
	result := v.Validate()
	assert.Equal(t, 7, result.Errors)

	lints := result.Lints
	assert.Equal(t, 15, len(lints))

	// image without full reference
	assert.Equal(t, Warning, lints[0].Kind)
	assert.Equal(t, `Step "s1" uses image "ubuntu"; consider using a fully qualified name - e.g. docker.io/library/ubuntu:1.0`, lints[0].Message)
	assert.Equal(t, Error, lints[1].Kind)
	assert.Equal(t, `Step "s1" uses image "ubuntu" which must be tagged with a specific version`, result.Lints[1].Message)

	// image without full reference but with latest tag
	assert.Equal(t, Warning, lints[2].Kind)
	assert.Equal(t, `Step "s2" uses image "api:latest"; consider using a fully qualified name - e.g. docker.io/library/ubuntu:1.0`, lints[2].Message)
	assert.Equal(t, Error, lints[3].Kind)
	assert.Equal(t, `Step "s2" uses image "api:latest" which must be tagged with a specific version`, result.Lints[3].Message)

	// image without a tag
	assert.Equal(t, Error, lints[4].Kind)
	assert.Equal(t, `Step "s3" uses image "docker.io/foo" which must be tagged with a specific version`, result.Lints[4].Message)

	// image with latest tag
	assert.Equal(t, Error, lints[5].Kind)
	assert.Equal(t, `Step "s4" uses image "r.j3ss.co/clisp:latest" which must be tagged with a specific version`, result.Lints[5].Message)

	// image without a tag
	assert.Equal(t, Error, lints[6].Kind)
	assert.Equal(t, `Step "s5" uses image "docker.io/abc/foo" which must be tagged with a specific version`, result.Lints[6].Message)

	// image with latest tag
	assert.Equal(t, Error, lints[7].Kind)
	assert.Equal(t, `Step "s6" uses image "gcr.io/foo/bar/baz:latest" which must be tagged with a specific version`, result.Lints[7].Message)

	// image with digest and a tag
	assert.Equal(t, Warning, lints[8].Kind)
	assert.Equal(t, `Step "s7" uses image "abc.io/fedora@sha256:deadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33f"; consider using a image tagged with specific version along with digest eg. abc.io/img:v1@sha256:abcde`, result.Lints[8].Message)

	// image with digest and latest tag
	assert.Equal(t, Warning, lints[9].Kind)
	assert.Equal(t, `Step "s8" uses image "abc.io/fedora:latest@sha256:deadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33f"; consider using a image tagged with specific version along with digest eg. abc.io/img:v1@sha256:abcde`, result.Lints[9].Message)

	// image with invalid digest
	assert.Equal(t, Error, lints[10].Kind)
	assert.Equal(t, `Step "s9" uses image "gcr.io/k8s-staging-boskos/boskosctl@sha256:a7fc984732" with an invalid digest. Error: digest must be between 71 and 71 runes in length: sha256:a7fc984732`, result.Lints[10].Message)

	// image with invalid registry
	assert.Equal(t, Warning, lints[11].Kind)
	assert.Equal(t, `Step "s10" uses image "localhost:5000/foo:bar"; consider using a fully qualified name - e.g. docker.io/library/ubuntu:1.0`, result.Lints[11].Message)

	// image with variable
	assert.Equal(t, Warning, lints[12].Kind)
	assert.Equal(t, `Step "s11" uses image "$(input.params.foo)" that contains variables; skipping validation`, result.Lints[12].Message)

	// image with variable
	assert.Equal(t, Warning, lints[13].Kind)
	assert.Equal(t, `Step "s12" uses image "docker.io/xyz/$(params.foo)" that contains variables; skipping validation`, result.Lints[13].Message)

	// image with variable
	assert.Equal(t, Warning, lints[14].Kind)
	assert.Equal(t, `Step "s13" uses image "docker.io/abc/foo:$(params.version)" that contains variables; skipping validation`, result.Lints[14].Message)
}

func TestTaskValidator_ValidEnvFromSecret(t *testing.T) {

	r := strings.NewReader(taskWithEnvFromSecret)
	parser := parser.ForReader(r)

	res, err := parser.Parse()
	assert.NilError(t, err)

	v := ForKind(res)
	result := v.Validate()
	assert.Equal(t, 0, result.Errors)

	assert.Equal(t, 2, len(result.Lints))

	// env.secretKeyRef generates a warning
	assert.Equal(t, Warning, result.Lints[0].Kind)
	assert.Equal(t, `Step "s1" uses secret to populate env "PASS". Prefer using secrets as files over secrets as environment variables`, result.Lints[0].Message)

	// envFrom.secretRef generates a warning
	assert.Equal(t, Warning, result.Lints[1].Kind)
	assert.Equal(t, `Step "s3" uses secret as environment variables. Prefer using secrets as files over secrets as environment variables`, result.Lints[1].Message)
}

func TestTaskValidator_ScriptUsingParams(t *testing.T) {
	r := strings.NewReader(taskWithScriptUsingParams)
	parser := parser.ForReader(r)

	res, err := parser.Parse()
	assert.NilError(t, err)

	v := ForKind(res)
	result := v.Validate()
	assert.Equal(t, 0, result.Errors)

	assert.Equal(t, 2, len(result.Lints))

	assert.Equal(t, Warning, result.Lints[0].Kind)
	assert.Equal(t, `Step "s1" references "$(params.secret)" directly from its script block. For reliability and security, consider putting the param into an environment variable of the Step and accessing that environment variable in your script instead.`, result.Lints[0].Message)

	assert.Equal(t, Warning, result.Lints[1].Kind)
	assert.Equal(t, `Step "s2" references "$(params.secret)" directly from its script block. For reliability and security, consider putting the param into an environment variable of the Step and accessing that environment variable in your script instead.`, result.Lints[1].Message)
}
