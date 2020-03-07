/*
 Copyright 2019 The Tekton Authors

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

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEmptyBody(t *testing.T) {
	r := createRequest("POST", "/", "issue_comment", nil)
	h := makeAddPRBodyHandler(getTestPrBody)
	w := httptest.NewRecorder()

	h(w, r)

	assertBadRequestResponse(t, w, "empty body, cannot add a pull request")
}

func TestNoAddPrBody(t *testing.T) {
	body := marshalEvent(t, makePrBody(false, ""))
	r := createRequest("POST", "/", "issue_comment", body)
	h := makeAddPRBodyHandler(getTestPrBody)
	w := httptest.NewRecorder()

	h(w, r)

	assertBadRequestResponse(t, w, "no 'add-pr-body' found in the body")
}

func TestNoPullRequestUrlFound(t *testing.T) {
	body := marshalEvent(t, makePrBody(true, ""))
	r := createRequest("POST", "/", "issue_comment", body)
	h := makeAddPRBodyHandler(getTestPrBody)
	w := httptest.NewRecorder()

	h(w, r)

	assertBadRequestResponse(t, w, "no 'pull-request-url' found")
}

func TestCannotFetchURL(t *testing.T) {
	body := marshalEvent(t, makePrBody(true, "foo://some_url"))
	r := createRequest("POST", "/", "issue_comment", body)
	h := makeAddPRBodyHandler(getTestPrBodyError)
	w := httptest.NewRecorder()

	h(w, r)

	assertBadRequestResponse(t, w, "something went wrong")
}

func TestFetchURL(t *testing.T) {
	body := marshalEvent(t, makePrBody(true, "foo://some_url"))
	r := createRequest("POST", "/", "issue_comment", body)
	h := makeAddPRBodyHandler(getTestPrBody)
	w := httptest.NewRecorder()

	h(w, r)

	want := makePrBody(true, "foo://some_url")
	wantPrBody, _ := getTestPrBody("foo://some_url")
	(*want)[rootPrBodyKey].(map[string]interface{})[prBodyContentKey] = wantPrBody

	resp := w.Result()

	assertResponsePayload(t, resp, &want)
}

// creates a GitHub hook type request - no secret is provided in testing.
func createRequest(method, url, event string, body []byte, opts ...requestOption) *http.Request {
	req := httptest.NewRequest(method, url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Github-Event", event)
	req.Header.Set("X-Github-Delivery", "testing-123")
	for _, o := range opts {
		o(req)
	}
	return req
}

type requestOption func(*http.Request)

func marshalEvent(t *testing.T, evt interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(evt)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func makePrBody(base bool, url string) *map[string]interface{} {
	prAddBody := make(map[string]interface{})
	if !base {
		return &prAddBody
	}
	var prAddBodyContent map[string]interface{}
	if url == "" {
		prAddBodyContent = map[string]interface{}{"foo": "bar"}
	} else {
		prAddBodyContent = map[string]interface{}{prBodyUrlKey: url}
	}
	prAddBody[rootPrBodyKey] = prAddBodyContent
	return &prAddBody
}

func getTestPrBody(prURL string) (map[string]interface{}, error) {
	retrievedPrBody := make(map[string]interface{})
	retrievedPrBody["foo"] = map[string]interface{}{}
	return retrievedPrBody, nil
}

func getTestPrBodyError(prURL string) (map[string]interface{}, error) {
	return nil, errors.New("something went wrong")
}

func assertResponseStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("incorrect response: got %v, want %v", resp.StatusCode, want)
	}
}

func assertResponsePayload(t *testing.T, resp *http.Response, v interface{}, opts ...cmp.Option) {
	t.Helper()
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	// This assumes that v is a pointer to a type, and unmarshals to a new value
	// of that type for the purposes of comparison.
	objType := reflect.TypeOf(v).Elem()
	obj := reflect.New(objType).Interface()
	err = json.Unmarshal(body, &obj)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(obj, v, opts...); diff != "" {
		t.Fatalf("compare failed: %s\n", diff)
	}
}

func assertBadRequestResponse(t *testing.T, rr *httptest.ResponseRecorder, s string) {
	resp := rr.Result()
	assertResponseStatus(t, resp, http.StatusBadRequest)
	assertResponsePayload(t, resp, makeErrorResponse(s))
}

func makeErrorResponse(s string) *triggerErrorPayload {
	return &triggerErrorPayload{Error: s}
}
