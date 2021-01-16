// Copyright © 2021 The Tekton Authors.
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

package linter

import (
	"strings"
	"testing"

	"github.com/tektoncd/plumbing/catlin/pkg/parser"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

const taskScriptValidatorGood = `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: hello-moto
spec:
  steps:
  - name: s1
    image: image1
    script: |
      #!/usr/bin/env sh
      echo "Hello world"
    script: |
      #!/bin/sh
      echo "hello moto"
`

const taskScriptValidatorNoGood = `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: hello-moto
spec:
  steps:
  - name: nogood
    image: image1
    script: |
      #!/usr/bin/env sh
      '
  - name: warn
    image: image1
    script: |
      #!/bin/sh
      echo "hello world"
`

const clusterTaskTest = `
apiVersion: tekton.dev/v1beta1
kind: ClusterTask
metadata:
  name: hello-moto
spec:
  steps:
  - name: nogood
    image: image1
    script: |
      #!/usr/bin/env sh
      '
`

const pipelineWithTaskRef = `
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: pipelien-1
spec:
  tasks:
  - name: pipeline-1
    taskRef:
      name: task-1`

var configSh = []config{
	config{regexp: `(/usr/bin/env |/bin/)sh`,
		linters: []linter{
			linter{
				cmd:  "sh", // Should always be everywhere
				args: []string{"-n"},
			},
		},
	},
}

func TestTaskLint_Script_good(t *testing.T) {
	r := strings.NewReader(taskScriptValidatorGood)
	parser := parser.ForReader(r)

	res, err := parser.Parse()
	assert.NilError(t, err)

	tl := &taskLinter{
		res:     res,
		configs: configSh,
	}
	result := tl.Validate()
	assert.Equal(t, 0, result.Errors)
}

func TestTaskLint_Script_no_good(t *testing.T) {
	r := strings.NewReader(taskScriptValidatorNoGood)
	parser := parser.ForReader(r)

	res, err := parser.Parse()
	assert.NilError(t, err)

	tl := &taskLinter{
		res:     res,
		configs: configSh,
	}
	result := tl.Validate()
	assert.Equal(t, 1, result.Errors)
}

func Test_Pipeline_skip(t *testing.T) {
	r := strings.NewReader(pipelineWithTaskRef)
	parser := parser.ForReader(r)

	res, err := parser.Parse()
	assert.NilError(t, err)

	tl := &taskLinter{
		res:     res,
		configs: configSh,
	}
	result := tl.Validate()
	assert.Assert(t, is.Nil(result.Lints))
}

func Test_ClusterTaskParse(t *testing.T) {
	r := strings.NewReader(clusterTaskTest)
	parser := parser.ForReader(r)

	res, err := parser.Parse()
	assert.NilError(t, err)

	tl := &taskLinter{
		res:     res,
		configs: configSh,
	}
	result := tl.Validate()
	assert.Equal(t, 1, result.Errors)
}
