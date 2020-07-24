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
	"path/filepath"
	"strings"

	"github.com/tektoncd/plumbing/catlin/pkg/app"
	"github.com/tektoncd/plumbing/catlin/pkg/parser"
	"go.uber.org/zap"
)

type PathValidator struct {
	path string
	log  *zap.SugaredLogger
	res  *parser.Resource
}

var _ Validator = (*PathValidator)(nil)

func NewPathValidator(cli app.CLI, r *parser.Resource, path string) *PathValidator {
	return &PathValidator{path: path, res: r, log: cli.Logger().Sugar()}
}

func (v *PathValidator) Validate() Result {
	r := v.res
	name := r.Name
	kind := strings.ToLower(r.Kind)
	version := r.Version()

	result := Result{}

	absPath, err := filepath.Abs(v.path)
	if err != nil {
		result.Error("invalid path: %s - %s", v.path, err)
		return result
	}

	expectedPath := filepath.Join(kind, name, version, name+".yaml")

	//v.log.Infof("name: %s | kind: %s | version: %s", name, kind, version)
	//v.log.Infof("abspath: %s", absPath)
	//v.log.Infof("expectedPath: %s", expectedPath)

	if !strings.HasSuffix(absPath, expectedPath) {
		result.Error("Resource path is invalid; expected path: %s", expectedPath)
	}
	return result
}
