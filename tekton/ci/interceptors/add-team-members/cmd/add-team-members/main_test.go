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
	h := makeAddTeamMembersHandler(getTestAddTeamMemberBody, getTestAddTeamMemberBody, "token")
	w := httptest.NewRecorder()

	h(w, r)

	assertBadRequestResponse(t, w, "empty body, cannot add team members")
}

func TestNoAddTeamMemberBody(t *testing.T) {
	body := marshalEvent(t, makeTeamBody(false, "", ""))
	r := createRequest("POST", "/", "issue_comment", body)
	h := makeAddTeamMembersHandler(getTestAddTeamMemberBody, getTestAddTeamMemberBody, "token")
	w := httptest.NewRecorder()

	h(w, r)

	assertBadRequestResponse(t, w, "no 'add-team-member' found in the body")
}

func TestNoOrgUrlFound(t *testing.T) {
	body := marshalEvent(t, makeTeamBody(true, "", ""))
	r := createRequest("POST", "/", "issue_comment", body)
	h := makeAddTeamMembersHandler(getTestAddTeamMemberBody, getTestAddTeamMemberBody, "token")
	w := httptest.NewRecorder()

	h(w, r)

	assertBadRequestResponse(t, w, "no 'org_base_url' found")
}

func TestNoTeamUrlFound(t *testing.T) {
	body := marshalEvent(t, makeTeamBody(true, "foo://some_url", ""))
	r := createRequest("POST", "/", "issue_comment", body)
	h := makeAddTeamMembersHandler(getTestAddTeamMemberBody, getTestAddTeamMemberBody, "token")
	w := httptest.NewRecorder()

	h(w, r)

	assertBadRequestResponse(t, w, "no 'team' found")
}

func TestCannotFetchOrgURL(t *testing.T) {
	body := marshalEvent(t, makeTeamBody(true, "foo://some_url", ""))
	r := createRequest("POST", "/", "issue_comment", body)
	h := makeAddTeamMembersHandler(getTestAddTeamMemberBodyError, getTestAddTeamMemberBody, "token")
	w := httptest.NewRecorder()

	h(w, r)

	assertBadRequestResponse(t, w, "something went wrong")
}

func TestCannotTeamURL(t *testing.T) {
	body := marshalEvent(t, makeTeamBody(true, "foo://some_url", "team1"))
	r := createRequest("POST", "/", "issue_comment", body)
	h := makeAddTeamMembersHandler(getTestAddTeamMemberBody, getTestAddTeamMemberBodyError, "token")
	w := httptest.NewRecorder()

	h(w, r)

	assertBadRequestResponse(t, w, "something went wrong")
}

func TestFetchURL(t *testing.T) {
	body := marshalEvent(t, makeTeamBody(true, "foo://some_url", "team1"))
	r := createRequest("POST", "/", "issue_comment", body)
	h := makeAddTeamMembersHandler(getTestAddTeamMemberBody, getTestAddTeamMemberBody, "token")
	w := httptest.NewRecorder()

	h(w, r)

	want := makeTeamBody(true, "foo://some_url", "team1")
	wantTeamBody := []interface{}{string("foo"), string("bar")}
	(*want)[rootAddTeamMembersKey].(map[string]interface{})[orgMembersKey] = wantTeamBody
	(*want)[rootAddTeamMembersKey].(map[string]interface{})[teamMembersKey] = wantTeamBody

	resp := w.Result()

	assertResponsePayload(t, resp, &want)
}

func TestDecodeListBody(t *testing.T) {
	body := marshalEvent(t, []string{"foo", "bar"})
	want := []interface{}{string("foo"), string("bar")}
	got, err := decodeListBody(body)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("compare failed: %s\n", diff)
	}
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

func makeTeamBody(base bool, url, team string) *map[string]interface{} {
	teamAddBody := make(map[string]interface{})
	if !base {
		return &teamAddBody
	}
	var teamAddBodyContent map[string]interface{}
	if url == "" {
		teamAddBodyContent = map[string]interface{}{"foo": "bar"}
	} else {
		teamAddBodyContent = map[string]interface{}{orgUrlKey: url}
	}
	if team == "" {
		teamAddBodyContent["foo2"] = "bar2"
	} else {
		teamAddBodyContent[teamKey] = team
	}
	teamAddBody[rootAddTeamMembersKey] = teamAddBodyContent
	return &teamAddBody
}

func getTestAddTeamMemberBody(url, token string) ([]string, error) {
	return []string{"foo", "bar"}, nil
}

func getTestAddTeamMemberBodyError(url, token string) ([]string, error) {
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
