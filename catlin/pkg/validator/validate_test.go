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
    tekton.dev/categories: Build Tools
    tekton.dev/displayName: My Example Task
    tekton.dev/platforms: linux/amd64,linux/s390x
spec:
  description: |-
    A summary of the resource

    A para about this valid task

  steps:
    - name: hello
      image: abc.io/ubuntu:1.0
      command: [sleep, infinity]
    - name: foo-bar
      image: abc.io/fedora:1.0@sha256:deadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33f
`

	inValidTask = `
---
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: invalid
  labels:
    app.kubernetes.io/version: a,b,c
  annotations:
    tekton.dev/tags: a,b,c
    tekton.dev/pipelines.minVersion: "0.12"
    tekton.dev/categories: Example
    tekton.dev/displayName: My Example Task
    tekton.dev/platforms: linux/amd64,linux/s390x
spec:
  description: |-
    A summary of the resource

    A para about this valid task

  steps:
    - name: hello
      image: abc.io/ubuntu:1.0
      command: [sleep, infinity]
    - name: foo-bar
      image: abc.io/fedora:1.0@sha256:deadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33f
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
    tekton.dev/categories: Automation
    tekton.dev/displayName: My Example Task
    tekton.dev/platforms: linux/amd64,linux/s390x
spec:
  description: |-
    A summary of the resource

    A para about this valid task

  tasks:
  - name: hello
    taskRef:
      name: hello
`

	taskWithInvalidPlatforms = `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: invalid
  labels:
    app.kubernetes.io/version: a,b,c
  annotations:
    tekton.dev/tags: a,b,c
    tekton.dev/pipelines.minVersion: "0.12"
    tekton.dev/categories: Automation
    tekton.dev/displayName: My Example Task
    tekton.dev/platforms: linux,linux/amd64,something-else
spec:
  description: |-
    A summary of the resource
  steps:
    - name: hello
      image: abc.io/ubuntu:1.0
      command: [sleep, infinity]
`

	taskWithoutPlatforms = `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: valid
  labels:
    app.kubernetes.io/version: a,b,c
  annotations:
    tekton.dev/tags: a,b,c
    tekton.dev/pipelines.minVersion: "0.12"
    tekton.dev/categories: "Automation"
    tekton.dev/displayName: My Example Task
spec:
  description: |-
    A summary of the resource
  steps:
    - name: hello
      image: abc.io/ubuntu:1.0
      command: [sleep, infinity]
`
)

func TestContentValidator_Task(t *testing.T) {

	r := strings.NewReader(validTask)
	parser := parser.ForReader(r)

	res, err := parser.Parse()
	assert.NilError(t, err)

	cat, err := GetCategories()
	assert.NilError(t, err)

	v := NewContentValidator(res, cat)
	result := v.Validate()

	assert.Equal(t, 0, result.Errors)
	assert.Equal(t, 0, len(result.Lints))
}

func TestContentValidator_Pipeline(t *testing.T) {

	r := strings.NewReader(validPipeline)
	parser := parser.ForReader(r)

	res, err := parser.Parse()
	assert.NilError(t, err)

	cat, err := GetCategories()
	assert.NilError(t, err)

	v := NewContentValidator(res, cat)
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

func TestContentValidator_InvalidPlatforms(t *testing.T) {

	r := strings.NewReader(taskWithInvalidPlatforms)
	parser := parser.ForReader(r)

	res, err := parser.Parse()
	assert.NilError(t, err)

	cat, err := GetCategories()
	assert.NilError(t, err)

	v := NewContentValidator(res, cat)
	result := v.Validate()
	assert.Equal(t, 2, result.Errors)

	lints := result.Lints
	assert.Equal(t, 2, len(lints))

	assert.Equal(t, Error, lints[0].Kind)
	assert.Equal(t, `"linux" platform must be in OS/ARCH format, e.g., linux/amd64`, result.Lints[0].Message)
	assert.Equal(t, Error, lints[1].Kind)
	assert.Equal(t, `"something-else" platform must be in OS/ARCH format, e.g., linux/amd64`, result.Lints[1].Message)
}

func TestContentValidator_WithoutPlatforms(t *testing.T) {

	r := strings.NewReader(taskWithoutPlatforms)
	parser := parser.ForReader(r)

	res, err := parser.Parse()
	assert.NilError(t, err)

	cat, err := GetCategories()
	assert.NilError(t, err)

	v := NewContentValidator(res, cat)
	result := v.Validate()
	assert.Equal(t, 0, result.Errors)

	lints := result.Lints
	assert.Equal(t, 1, len(lints))

	assert.Equal(t, Recommendation, lints[0].Kind)
	assert.Equal(t, `Task: tekton.dev/v1beta1 - name: "valid" is more usable if it has "tekton.dev/platforms" annotation about platforms to run`, result.Lints[0].Message)
}

func TestCategoryValidatorInvalid_Task(t *testing.T) {

	r := strings.NewReader(inValidTask)
	parser := parser.ForReader(r)

	res, err := parser.Parse()
	assert.NilError(t, err)

	cat, err := GetCategories()
	assert.NilError(t, err)

	v := NewContentValidator(res, cat)
	result := v.Validate()

	const errMsg = `Category not defined
You can choose from the categories present at location: https://raw.githubusercontent.com/tektoncd/hub/main/config.yaml"`

	assert.Equal(t, 1, result.Errors)
	assert.Equal(t, 1, len(result.Lints))
	assert.Equal(t, result.Lints[0].Message, errMsg)
}
