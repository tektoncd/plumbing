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
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"github.com/tektoncd/plumbing/catlin/pkg/parser"
)

const (
	parameterSubstitution = `[_a-zA-Z][_a-zA-Z0-9.-]*(\[\*\])?`
	braceMatchingRegex    = "(\\$(\\(%s.(?P<var>%s)\\)))"
)

type taskValidator struct {
	res *parser.Resource
}

var _ Validator = (*taskValidator)(nil)

func NewTaskValidator(r *parser.Resource) *taskValidator {
	return &taskValidator{res: r}
}

func (t *taskValidator) Validate() Result {

	result := Result{}

	res, err := t.res.ToType()
	if err != nil {
		result.Error("Failed to decode to a Task - %s", err)
		return result
	}

	task := res.(*v1beta1.Task)
	for _, step := range task.Spec.Steps {
		result.Append(t.validateStep(step))

	}
	return result
}

func (t *taskValidator) validateStep(s v1beta1.Step) Result {

	result := Result{}
	step := s.Name
	img := s.Image

	if _, usesVars := extractExpressionFromString(img, ""); usesVars {
		result.Warn("Step %q uses image %q that contains variables; skipping validation", step, img)
		return result
	}

	if !strings.Contains(img, "/") || !isValidRegistry(img) {
		result.Warn("Step %q uses image %q; consider using a fully qualified name - e.g. docker.io/library/ubuntu:1.0", step, img)
	}

	if strings.Contains(img, "@sha256") {
		rep, err := name.NewDigest(img, name.WeakValidation)
		if err != nil {
			result.Error("Step %q uses image %q with an invalid digest. Error: %s", step, img, err)
			return result
		}

		if !tagWithDigest(rep.String()) {
			result.Warn("Step %q uses image %q; consider using a image tagged with specific version along with digest eg. abc.io/img:v1@sha256:abcde", step, img)
		}

		return result
	}

	ref, err := name.NewTag(img, name.WeakValidation)
	if err != nil {
		result.Error("Step %q uses image %q with an invalid tag. Error: %s", step, img, err)
		return result
	}

	if strings.Contains(ref.Identifier(), "latest") {
		result.Error("Step %q uses image %q which must be tagged with a specific version", step, img)
	}

	// According to [CIS benchmarks](https://cloud.google.com/kubernetes-engine/docs/concepts/cis-benchmarks).
	// > 5.4.1 Prefer using secrets as files over secrets as environment variables
	for _, env := range s.Env {
		if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
			continue
		}
		result.Warn("Step %q uses secret to populate env %q. Prefer using secrets as files over secrets as environment variables", step, env.Name)
	}
	for _, envFrom := range s.EnvFrom {
		if envFrom.SecretRef == nil {
			continue
		}
		result.Warn("Step %q uses secret as environment variables. Prefer using secrets as files over secrets as environment variables", step)
	}

	if s.Script != "" {
		expr, _ := extractExpressionFromString(s.Script, "params")
		if expr != "" {
			result.Warn(
				"Step %q references %q directly from its script block. For reliability and security, consider putting the param into an environment variable of the Step and accessing that environment variable in your script instead.",
				step,
				expr)

		}
	}

	return result
}

// copied from tektoncd/pipeline
func extractExpressionFromString(s, prefix string) (string, bool) {
	pattern := fmt.Sprintf(braceMatchingRegex, prefix, parameterSubstitution)
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(s)
	if match == nil {
		return "", false
	}
	return match[0], true
}

func isValidRegistry(img string) bool {
	repo := strings.Split(img, "/")[0]
	return strings.Contains(repo, ".")
}

// tagWithDigest validates if image has a specific tag along with digest
func tagWithDigest(img string) bool {
	withOutDigest := strings.Split(img, "@sha256")[0]
	if strings.Contains(withOutDigest, ":") && !strings.HasSuffix(withOutDigest, ":latest") {
		return true
	}
	return false
}
