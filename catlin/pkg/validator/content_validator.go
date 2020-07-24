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
	"strings"

	"github.com/tektoncd/plumbing/catlin/pkg/app"
	"github.com/tektoncd/plumbing/catlin/pkg/consts"
	"github.com/tektoncd/plumbing/catlin/pkg/parser"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ContentValidator struct {
	res *parser.Resource
	log *zap.SugaredLogger
}

var _ Validator = (*ContentValidator)(nil)

func NewContentValidator(cli app.CLI, r *parser.Resource) *ContentValidator {
	return &ContentValidator{res: r, log: cli.Logger().Sugar()}
}

func (v *ContentValidator) Validate() Result {

	res := v.res.Unstructured
	gvk := res.GroupVersionKind()
	kind := res.GetKind()
	name := v.res.Name
	resName := fmt.Sprintf("%s: %s - name: %q", kind, gvk.GroupVersion(), name)

	result := Result{}

	labels := res.GetLabels()
	if _, ok := labels[consts.VersionLabel]; !ok {
		result.Error("%s must have a label %q to indicate version", resName, consts.VersionLabel)
	}

	annotations := res.GetAnnotations()
	if _, ok := annotations[consts.MinPipelineVersionAnnotation]; !ok {
		result.Error("%s is missing a mandatory annotation for minimum pipeline version(%q)", resName, consts.MinPipelineVersionAnnotation)
	}

	if _, ok := annotations[consts.DisplayNameAnnotation]; !ok {
		result.Recommend("%s is missing a readable display name annotation(%q)", resName, consts.DisplayNameAnnotation)
	}

	if _, found, err := unstructured.NestedString(res.Object, "spec", "description"); !found || err != nil {
		result.Error("%s must have a description", resName)
	}

	tags := strings.FieldsFunc(annotations[consts.TagsAnnotation], func(c rune) bool { return c == ' ' || c == ',' })
	if len(tags) == 0 {
		result.Recommend("%s is easily discoverable if it has annotation for tag %q", resName, consts.TagsAnnotation)
	}
	return result
}
