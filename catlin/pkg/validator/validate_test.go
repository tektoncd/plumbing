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

	"gotest.tools/v3/assert"

	"github.com/tektoncd/plumbing/catlin/pkg/parser"
)

const (
	validTask = `
---
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: valid
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

  steps:
    - name: hello
      image: abc.io/ubuntu:1.0
      command: [sleep, infinity]
    - name: foo-bar
      image: abc.io/fedora@sha256:deadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33f
`

	validPipeline = `
---
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: valid
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

  tasks:
  - name: hello
    taskRef:
      name: hello
`
)

func TestContentValidator_Task(t *testing.T) {

	r := strings.NewReader(validTask)
	parser := parser.ForReader(r)

	res, err := parser.Parse()
	assert.NilError(t, err)

	v := NewContentValidator(res)
	result := v.Validate()

	assert.Equal(t, 0, result.Errors)
	assert.Equal(t, 0, len(result.Lints))
}

func TestContentValidator_Pipeline(t *testing.T) {

	r := strings.NewReader(validPipeline)
	parser := parser.ForReader(r)

	res, err := parser.Parse()
	assert.NilError(t, err)

	v := NewContentValidator(res)
	result := v.Validate()

	assert.Equal(t, 0, result.Errors)
	assert.Equal(t, 0, len(result.Lints))
}

func TestValidatorForKind_Task(t *testing.T) {

	r := strings.NewReader(validTask)
	parser := parser.ForReader(r)

	res, err := parser.Parse()
	assert.NilError(t, err)

	v := ForKind(res)
	result := v.Validate()

	assert.Equal(t, 0, result.Errors)
	assert.Equal(t, 0, len(result.Lints))
}
