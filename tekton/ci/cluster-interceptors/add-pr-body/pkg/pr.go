/*
 Copyright 2022 The Tekton Authors

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

package pkg

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"google.golang.org/grpc/codes"
)

const (
	prExtensionsKey        = "add_pr_body"
	prExtensionsUrlKey     = "pull_request_url"
	prExtensionsContentKey = "pull_request_body"
)

var _ triggersv1.InterceptorInterface = (*Interceptor)(nil)

type Interceptor struct {
	// AuthToken is an OAuth token used to connect to the GitHub API
	AuthToken string
}

func (w Interceptor) Process(ctx context.Context, r *triggersv1.InterceptorRequest) *triggersv1.InterceptorResponse {
	// Assumption - there is an extension key called "add_pr_body.pull_request_url")
	// Get the URL from the body
	prUrl, err := getPrUrlFromExtension(r.Extensions)
	if err != nil {
		return interceptors.Fail(codes.FailedPrecondition, err.Error())
	}
	// TODO: Refactor this into its own struct field?
	prBody, err := getPrBody(prUrl, w.AuthToken)
	if err != nil {
		// TODO: Refactor getPrBody to map errors better to error codes
		return interceptors.Fail(codes.Internal, err.Error())
	}
	return &triggersv1.InterceptorResponse{
		Extensions: map[string]interface{}{
			prExtensionsKey: map[string]interface{}{
				prExtensionsContentKey: prBody,
			},
		},
		Continue: true,
	}
}

// getPrUrlFromExtensions gets the PrUrl value from the InterceptorRequest's extensions field.
func getPrUrlFromExtension(extensions map[string]interface{}) (string, error) {
	addPrBody, ok := extensions[prExtensionsKey]
	if !ok {
		return "", errors.New("no 'add_pr_body' found in the extensions")
	}
	prUrl, ok := addPrBody.(map[string]interface{})[prExtensionsUrlKey]
	if !ok {
		return "", errors.New("no 'pull_request_url' found")
	}
	prUrlString, ok := prUrl.(string)
	if !ok {
		return "", errors.New("'pull_request_url' found, but not a string")
	}
	return prUrlString, nil
}

func decodeBody(body []byte) (map[string]interface{}, error) {
	var jsonMap map[string]interface{}
	err := json.Unmarshal(body, &jsonMap)
	if err != nil {
		return nil, err
	}
	return jsonMap, nil
}

func getPrBody(prUrl string, token string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", prUrl, nil)
	if err != nil {
		return nil, err
	}

	// If token isn't an empty string add GitHub Enterprise OAuth header
	if token != "" {
		req.Header.Add("Authorization", "token "+token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return decodeBody(body)
}
