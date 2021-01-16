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
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/plumbing/catlin/pkg/parser"
	"github.com/tektoncd/plumbing/catlin/pkg/validator"
)

type taskLinter struct {
	res     *parser.Resource
	configs []config
}

type linter struct {
	cmd  string
	args []string
}

type config struct {
	regexp  string
	linters []linter
}

// NewConfig construct default config
func NewConfig() []config {
	return []config{
		// Default one is the first one
		config{regexp: `(/usr/bin/env |.*/bin/)sh`,
			linters: []linter{
				linter{
					cmd:  "shellcheck",
					args: []string{"-s", "sh"},
				},
				linter{
					cmd:  "sh",
					args: []string{"-n"},
				},
			}},
		config{regexp: `(/usr/bin/env |.*/bin/)bash`,
			linters: []linter{
				linter{
					cmd:  "shellcheck",
					args: []string{"-s", "bash"},
				},
				linter{
					cmd:  "bash",
					args: []string{"-n"},
				},
			}},
		config{regexp: `(/usr/bin/env\s|.*/bin/|/usr/libexec/platform-)python(23)?`,
			linters: []linter{
				linter{
					cmd:  "pylint",
					args: []string{"-dC0103"}, // Disabling C0103 which is invalid name convention
				},
			}},
	}
}

// NewScriptLinter construct a new task lister struct
func NewScriptLinter(r *parser.Resource) *taskLinter {
	return &taskLinter{res: r, configs: NewConfig()}
}

func (t *taskLinter) validateScript(taskName string, s v1beta1.Step, configs []config) validator.Result {
	result := validator.Result{}

	// use /bin/sh by default if no shbang
	if s.Script[0:2] != "#!" {
		s.Script = "#!/usr/bin/env sh\n" + s.Script
	} else { // using a shbang, check if we have /usr/bin/env
		if s.Script[0:14] != "#!/usr/bin/env" {
			result.Warn("step: %s is not using #!/usr/bin/env ", taskName)
		}
	}

	for _, config := range configs {
		matched, err := regexp.MatchString(`^#!`+config.regexp+`\n`, s.Script)

		if err != nil {
			result.Error("Invalid regexp: %s", config.regexp)
			return result
		}

		if !matched {
			continue
		}

		for _, linter := range config.linters {
			execpath, err := exec.LookPath(linter.cmd)
			if err != nil {
				result.Warn("Couldn't find the linter %s in the path", linter.cmd)
				continue
			}
			tmpfile, err := ioutil.TempFile("", "catlin-script-linter")
			if err != nil {
				result.Error("Cannot create temporary files")
				return result
			}
			defer os.Remove(tmpfile.Name()) // clean up
			if _, err := tmpfile.Write([]byte(s.Script)); err != nil {
				result.Error("Cannot write to temporary files")
				return result
			}
			if err := tmpfile.Close(); err != nil {
				result.Error("Cannot close temporary files")
				return result
			}

			// TODO: perhaps the filename is not necessary will be at the end of
			// a command, may need some variable interpolation so the linter can
			// specify where the filaname is into the command line.
			cmd := exec.Command(execpath, append(linter.args, tmpfile.Name())...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				outt := strings.ReplaceAll(string(out), tmpfile.Name(), taskName+"-"+s.Name)
				result.Error("%s, %s failed:\n%s", execpath, linter.args, outt)
			}
		}
	}

	return result
}

func (t *taskLinter) collectOverSteps(steps []v1beta1.Step, name string, result *validator.Result) {
	for _, step := range steps {
		if step.Script != "" {
			result.Append(t.validateScript(name, step, t.configs))
		}
	}
}

func (t *taskLinter) Validate() validator.Result {
	result := validator.Result{}
	res, err := t.res.ToType()
	if err != nil {
		result.Error("Failed to decode to a Task - %s", err)
		return result
	}

	switch strings.ToLower(t.res.Kind) {
	case "task":
		task := res.(*v1beta1.Task)
		t.collectOverSteps(task.Spec.Steps, task.ObjectMeta.Name, &result)
	case "clustertask":
		task := res.(*v1beta1.ClusterTask)
		t.collectOverSteps(task.Spec.Steps, task.ObjectMeta.Name, &result)
	}
	return result
}
