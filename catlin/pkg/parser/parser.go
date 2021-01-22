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

package parser

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	decoder "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/pkg/apis"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/plumbing/catlin/pkg/consts"
)

func registerSchema() {
	beta1 := runtime.NewSchemeBuilder(v1beta1.AddToScheme)
	_ = beta1.AddToScheme(scheme.Scheme)
}

type Resource struct {
	Name         string
	Kind         string
	Unstructured *unstructured.Unstructured
	Object       runtime.Object
}

func (r *Resource) Version() string {
	if r.Unstructured == nil {
		return ""
	}

	labels := r.Unstructured.GetLabels()
	v, ok := labels[consts.VersionLabel]
	if !ok {
		return ""
	}
	return v
}

func (r *Resource) ToType() (interface{}, error) {
	return convertToTyped(r.Unstructured)
}

type Parser interface {
	Parse() (*Resource, error)
}

type TektonParser struct {
	reader io.Reader
}

var once sync.Once

func ForReader(r io.Reader) *TektonParser {
	once.Do(registerSchema)
	return &TektonParser{reader: r}
}

func (t *TektonParser) Parse() (*Resource, error) {

	// both UniversalDeserializer and NewYAMLToJSONDecoder need to
	var dup bytes.Buffer
	r := io.TeeReader(t.reader, &dup)
	contents, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	object, gvk, err := scheme.Codecs.UniversalDeserializer().Decode(contents, nil, nil)
	if err != nil || !isTektonKind(gvk) {
		return nil, fmt.Errorf("parse error: invalid resource %+v:\n%s", err, contents)
	}

	decoder := decoder.NewYAMLToJSONDecoder(&dup)

	var res *unstructured.Unstructured
	for {
		res = &unstructured.Unstructured{}
		if err := decoder.Decode(res); err != nil {
			return nil, fmt.Errorf("failed to decode: %w", err)
		}

		if len(res.Object) == 0 {
			continue
		}

		if res.GetKind() != "" {
			break
		}
	}

	if _, err := convertToTyped(res); err != nil {
		return nil, err
	}

	return &Resource{
		Unstructured: res,
		Object:       object,
		Name:         res.GetName(),
		Kind:         gvk.Kind,
	}, nil
}

func convertToTyped(u *unstructured.Unstructured) (interface{}, error) {
	t, err := typeForKind(u.GetKind())
	if err != nil {
		return nil, err
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, t); err != nil {
		return nil, err
	}

	t.SetDefaults(context.Background())
	if fe := t.Validate(context.Background()); !isEmptyFieldError(fe) {
		return nil, fe
	}

	return t, nil
}

func isEmptyFieldError(fe *apis.FieldError) bool {
	if fe == nil {
		return true
	}
	return fe.Message == "" && fe.Details == "" && len(fe.Paths) == 0
}

type tektonResource interface {
	apis.Validatable
	apis.Defaultable
}

func typeForKind(kind string) (tektonResource, error) {
	switch kind {
	case "Task":
		return &v1beta1.Task{}, nil
	case "ClusterTask":
		return &v1beta1.ClusterTask{}, nil
	case "Pipeline":
		return &v1beta1.Pipeline{}, nil
	}

	return nil, fmt.Errorf("unknown kind %s", kind)
}

func isTektonKind(gvk *schema.GroupVersionKind) bool {
	id := gvk.GroupVersion().Identifier()
	return id == v1beta1.SchemeGroupVersion.Identifier()
}
