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

import "fmt"

type Status int

const (
	// represents Queued state
	Passed Status = iota
	Failed
	Unknown
)

func (s Status) String() string {
	return [...]string{"passed", "failed", "unknown"}[s]
}

type Kind int

const (
	// represents Queued state
	Error Kind = iota
	Warning
	Recommendation
	Info
)

func (t Kind) String() string {
	return [...]string{"error", "warning", "recommendation", "info"}[t]
}

type Lint struct {
	Kind    Kind
	Message string
}

type Result struct {
	//Status Status
	Lints    []Lint
	Errors   int
	Warnings int
}

func (r *Result) add(k Kind, format string, args ...interface{}) {
	if r.Lints == nil {
		r.Lints = []Lint{}
	}
	r.Lints = append(r.Lints, Lint{Kind: k, Message: fmt.Sprintf(format, args...)})

}

func (r *Result) Error(format string, args ...interface{}) {
	r.Errors++
	r.add(Error, format, args...)
}

func (r *Result) Warn(format string, args ...interface{}) {
	r.Warnings++
	r.add(Warning, format, args...)
}

func (r *Result) Recommend(format string, args ...interface{}) {
	r.add(Recommendation, format, args...)
}

func (r *Result) Info(format string, args ...interface{}) {
	r.add(Info, format, args...)
}

func (r *Result) Append(other Result) {
	r.Errors += other.Errors
	r.Lints = append(r.Lints, other.Lints...)
}
