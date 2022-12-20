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

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/apis"
)

const (
	repoKey        = "repo"
	shaKey         = "sha"
	targetURLKey   = "targetURL"
	descriptionKey = "description"
	stateKey       = "state"
	jobNameKey     = "jobName"
)

// StatusInfo defines the desired state of the status update
type StatusInfo struct {
	// Repo is the repository name.
	Repo string `json:"repo"`

	// SHA is the commit SHA the job ran against.
	SHA string `json:"sha"`

	// JobName is the name of the job whose result we're receiving.
	JobName string `json:"jobName"`

	// TargetURL is the URL for the job's logs.
	// +optional
	TargetURL string `json:"targetURL,omitempty"`

	// State is the state for the status. Must be one of `error`, `pending`, `failure`, or `success`
	State string `json:"state"`

	// Description is an optional description for the status.
	// +optional
	Description string `json:"description,omitempty"`
}

// StatusInfoFromRun reads params from the given Run and returns either a populated info or errors.
func StatusInfoFromRun(r *v1beta1.CustomRun) (*StatusInfo, *apis.FieldError) {
	statusInfo := &StatusInfo{}
	var errs *apis.FieldError

	if commit := r.Spec.GetParam(shaKey); commit != nil {
		if commit.Value.Type != v1beta1.ParamTypeString {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("should be a string, is %s", commit.Value.Type), shaKey))
		} else {
			statusInfo.SHA = commit.Value.StringVal
		}
	} else {
		errs = errs.Also(apis.ErrMissingField(shaKey))
	}

	if repo := r.Spec.GetParam(repoKey); repo != nil {
		if repo.Value.Type != v1beta1.ParamTypeString {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("should be a string, is %s", repo.Value.Type), repoKey))
		} else {
			statusInfo.Repo = repo.Value.StringVal
		}
	} else {
		errs = errs.Also(apis.ErrMissingField(repoKey))
	}

	if jobName := r.Spec.GetParam(jobNameKey); jobName != nil {
		if jobName.Value.Type != v1beta1.ParamTypeString {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("should be a string, is %s", jobName.Value.Type), jobNameKey))
		} else {
			statusInfo.JobName = jobName.Value.StringVal
		}
	} else {
		errs = errs.Also(apis.ErrMissingField(jobNameKey))
	}

	if targetURL := r.Spec.GetParam(targetURLKey); targetURL != nil {
		if targetURL.Value.Type != v1beta1.ParamTypeString {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("should be a string, is %s", targetURL.Value.Type), targetURLKey))
		} else {
			statusInfo.TargetURL = targetURL.Value.StringVal
		}
	}

	if desc := r.Spec.GetParam(descriptionKey); desc != nil {
		if desc.Value.Type != v1beta1.ParamTypeString {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("should be a string, is %s", desc.Value.Type), descriptionKey))
		} else {
			statusInfo.Description = desc.Value.StringVal
		}
	}

	if stateVal := r.Spec.GetParam(stateKey); stateVal != nil {
		if stateVal.Value.Type != v1beta1.ParamTypeString {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("should be a state, is %s", stateVal.Value.Type), stateKey))
		} else {
			stateStr := stateVal.Value.StringVal
			if stateStr != "error" && stateStr != "pending" && stateStr != "failure" && stateStr != "success" {
				errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("must be one of 'error', 'pending', 'failure', or 'success', but was %s", stateStr), stateKey))
			} else {
				statusInfo.State = stateStr
			}
		}
	} else {
		errs = errs.Also(apis.ErrMissingField(stateKey))
	}

	return statusInfo, errs
}
