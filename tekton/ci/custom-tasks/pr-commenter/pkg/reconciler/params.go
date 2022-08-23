/*
Copyright 2020 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package reconciler

import (
	"fmt"
	"strconv"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/apis"
)

const (
	repoKey     = "repo"
	prNumberKey = "prNumber"
	shaKey      = "sha"
	jobNameKey  = "jobName"
	successKey  = "isSuccess"
	optionalKey = "isOptional"
	logURLKey   = "logURL"

	defaultIsOptional = false
)

// ReportInfo defines the desired state of the PRCommenter
type ReportInfo struct {
	// Repo is the repository name.
	Repo string `json:"repo"`

	// PRNumber is the PR number to comment on.
	PRNumber int `json:"prNumber"`

	// SHA is the commit SHA the job ran against.
	SHA string `json:"sha"`

	// JobName is the name of the job whose result we're receiving.
	JobName string `json:"jobName"`

	// IsSuccess is whether the job whose result we're receiving failed or succeeded.
	IsSuccess bool `json:"isSuccess"`

	// LogURL is the URL for the job's logs.
	// +optional
	LogURL string `json:"logURL,omitempty"`

	// IsOptional is true if the job is optional.
	// +optional
	IsOptional bool `json:"isOptional,omitempty"`
}

// ReportInfoFromRun reads params from the given Run and returns either a populated info or errors.
func ReportInfoFromRun(r *v1alpha1.Run) (*ReportInfo, *apis.FieldError) {
	report := &ReportInfo{}
	var errs *apis.FieldError

	if prNum := r.Spec.GetParam(prNumberKey); prNum != nil {
		if prNum.Value.Type != v1beta1.ParamTypeString {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("should be a number, is %s", prNum.Value.Type), prNumberKey))
		} else if numVal, err := strconv.Atoi(prNum.Value.StringVal); err != nil {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("%s should be a number", prNum.Value.StringVal), prNumberKey))
		} else {
			report.PRNumber = numVal
		}
	} else {
		errs = errs.Also(apis.ErrMissingField(prNumberKey))
	}

	if commit := r.Spec.GetParam(shaKey); commit != nil {
		if commit.Value.Type != v1beta1.ParamTypeString {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("should be a string, is %s", commit.Value.Type), shaKey))
		} else {
			report.SHA = commit.Value.StringVal
		}
	} else {
		errs = errs.Also(apis.ErrMissingField(shaKey))
	}

	if repo := r.Spec.GetParam(repoKey); repo != nil {
		if repo.Value.Type != v1beta1.ParamTypeString {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("should be a string, is %s", repo.Value.Type), repoKey))
		} else {
			report.Repo = repo.Value.StringVal
		}
	} else {
		errs = errs.Also(apis.ErrMissingField(repoKey))
	}

	if jobName := r.Spec.GetParam(jobNameKey); jobName != nil {
		if jobName.Value.Type != v1beta1.ParamTypeString {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("should be a string, is %s", jobName.Value.Type), jobNameKey))
		} else {
			report.JobName = jobName.Value.StringVal
		}
	} else {
		errs = errs.Also(apis.ErrMissingField(jobNameKey))
	}

	if logURL := r.Spec.GetParam(logURLKey); logURL != nil {
		if logURL.Value.Type != v1beta1.ParamTypeString {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("should be a string, is %s", logURL.Value.Type), logURLKey))
		} else {
			report.LogURL = logURL.Value.StringVal
		}
	} else {
		errs = errs.Also(apis.ErrMissingField(logURLKey))
	}

	if successVal := r.Spec.GetParam(successKey); successVal != nil {
		if successVal.Value.Type != v1beta1.ParamTypeString {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("should be a bool, is %s", successVal.Value.Type), successKey))
		} else if boolVal, err := strconv.ParseBool(successVal.Value.StringVal); err != nil {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("%s should be a bool", successVal.Value.StringVal), successKey))
		} else {
			report.IsSuccess = boolVal
		}
	} else {
		errs = errs.Also(apis.ErrMissingField(successKey))
	}

	if optionalVal := r.Spec.GetParam(optionalKey); optionalVal != nil {
		if optionalVal.Value.Type != v1beta1.ParamTypeString {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("should be a bool, is %s", optionalVal.Value.Type), optionalKey))
		} else if boolVal, err := strconv.ParseBool(optionalVal.Value.StringVal); err != nil {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("%s should be a bool", optionalVal.Value.StringVal), optionalKey))
		} else {
			report.IsOptional = boolVal
		}
	} else {
		report.IsOptional = defaultIsOptional
	}

	return report, errs
}
