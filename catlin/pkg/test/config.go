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

package test

import (
	"go.uber.org/zap"

	"github.com/tektoncd/plumbing/catlin/pkg/app"
)

type testConfig struct {
	log *zap.Logger
}

var _ app.CLI = (*testConfig)(nil)

func New() *testConfig {
	log, _ := zap.NewDevelopment()
	return &testConfig{
		log: log,
	}
}

func (t *testConfig) Logger() *zap.Logger {
	return t.log
}

func (t *testConfig) Stream() *app.Stream {
	return nil
}
