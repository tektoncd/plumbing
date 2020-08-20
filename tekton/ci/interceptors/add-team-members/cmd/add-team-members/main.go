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

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type triggerErrorPayload struct {
	Error string `json:"errorMessage,omitempty"`
}

type urlToList func(string, string) ([]string, error)

const (
	rootAddTeamMembersKey = "add_team_members"
	orgUrlKey             = "org_base_url"
	teamKey               = "team"
	orgMembersKey         = "public_org_members"
	teamMembersKey        = "maintainers_team_members"
	publicMembersPath     = "%s/public_members"
	teamPath              = "%s/teams/%s-maintainers/members"
)

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("missing required environment variable GITHUB_TOKEN")
		os.Exit(1)
	}
	http.HandleFunc("/", makeAddTeamMembersHandler(fetchMembers, fetchMembers, token))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", 8080), nil))
}

func makeAddTeamMembersHandler(orgMembersFetcher, teamMembersFetcher urlToList, token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var payload []byte
		var err error

		// Get the payload
		if r.Body != nil {
			defer r.Body.Close()
			payload, err = ioutil.ReadAll(r.Body)
			if err != nil {
				log.Printf("failed to read request body: %q", err)
				marshalError(err, w)
				return
			}
			if len(payload) == 0 {
				bodyError := errors.New("empty body, cannot add team members")
				log.Printf("No body received: %q", bodyError)
				marshalError(bodyError, w)
				return
			}
		} else {
			bodyError := errors.New("empty body, cannot add team members")
			log.Printf("failed to read request body: %q", err)
			marshalError(bodyError, w)
			return
		}

		// Get the json body
		jsonBody, err := decodeMapBody(payload)
		if err != nil {
			log.Printf("failed to decode the body: %q", err)
			marshalError(err, w)
			return
		}
		// Get the org URL from the body
		orgUrl, err := getOrgUrl(jsonBody)
		if err != nil {
			log.Printf("failed to extract the Org URL from the body: %q", err)
			marshalError(err, w)
			return
		}
		// Get the list of org members from the URL
		orgMembers, err := orgMembersFetcher(orgUrl, "")
		if err != nil {
			log.Printf("failed to get the list of org members: %q", err)
			marshalError(err, w)
			return
		}
		// Add the PR body to the original body
		jsonBody[rootAddTeamMembersKey].(map[string]interface{})[orgMembersKey] = orgMembers
		// Get the team URL from the body
		teamUrl, err := getTeamUrl(jsonBody)
		if err != nil {
			log.Printf("failed to extract the Team URL from the body: %q", err)
			marshalError(err, w)
			return
		}
		// Get the list of org members from the URL
		teamMembers, err := teamMembersFetcher(teamUrl, token)
		if err != nil {
			log.Printf("failed to get the list of team members: %q", err)
			marshalError(err, w)
			return
		}
		// Add the PR body to the original body
		jsonBody[rootAddTeamMembersKey].(map[string]interface{})[teamMembersKey] = teamMembers

		// Marshal the body
		responseBytes, err := json.Marshal(jsonBody)
		if err != nil {
			log.Printf("failed marshal the response body: %q", err)
			marshalError(err, w)
			return
		}
		// Set all the original headers
		for k, values := range r.Header {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}
		// Write the response
		n, err := w.Write(responseBytes)
		if err != nil {
			log.Printf("Failed to write response. Bytes written: %d. Error: %q", n, err)
		}
	}
}

func marshalError(err error, w http.ResponseWriter) {
	if err != nil {
		triggerBody := triggerErrorPayload{
			Error: err.Error(),
		}
		tPayload, err := json.Marshal(triggerBody)
		if err != nil {
			log.Printf("Failed to marshal the trigger body. Error: %q", err)
			http.Error(w, "{}", http.StatusBadRequest)
			return
		}
		http.Error(w, string(tPayload[:]), http.StatusBadRequest)
	}
}

func decodeMapBody(body []byte) (map[string]interface{}, error) {
	var jsonMap map[string]interface{}
	err := json.Unmarshal(body, &jsonMap)
	if err != nil {
		return nil, err
	}
	return jsonMap, nil
}

func decodeListBody(body []byte) ([]interface{}, error) {
	var jsonList []interface{}
	err := json.Unmarshal(body, &jsonList)
	if err != nil {
		return nil, err
	}
	return jsonList, nil
}

func getTeamMemberValue(key string, body map[string]interface{}) (string, error) {
	addTeamMember, ok := body[rootAddTeamMembersKey]
	if !ok {
		return "", errors.New("no 'add-team-member' found in the body")
	}
	value, ok := addTeamMember.(map[string]interface{})[key]
	if !ok {
		return "", fmt.Errorf("no '%s' found", key)
	}
	valueString, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("'%s' found, but not a string", key)
	}
	return valueString, nil
}

func getOrgBaseUrl(body map[string]interface{}) (string, error) {
	return getTeamMemberValue(orgUrlKey, body)
}

func getTeam(body map[string]interface{}) (string, error) {
	return getTeamMemberValue(teamKey, body)
}

func getOrgUrl(body map[string]interface{}) (string, error) {
	baseUrl, err := getOrgBaseUrl(body)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(publicMembersPath, baseUrl), nil
}

func getTeamUrl(body map[string]interface{}) (string, error) {
	baseUrl, err := getOrgBaseUrl(body)
	if err != nil {
		return "", err
	}
	teamName, err := getTeam(body)
	if err != nil {
		return "", err
	}
	// Pipeline's maintainer team is called "core-maintainers"
	if teamName == "pipeline" {
		teamName = "core"
	}
	return fmt.Sprintf(teamPath, baseUrl, teamName), nil
}

func fetchMembers(membersUrl, token string) ([]string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", membersUrl, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("token %s", token))
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	decodedBody, err := decodeListBody(body)
	if err != nil {
		return nil, fmt.Errorf("Error decoding body %v to a list: %q", string(body[:]), err)
	}

	members := []string{}
	for _, member := range decodedBody {
		value, ok := member.(map[string]interface{})["login"]
		if !ok {
			return nil, fmt.Errorf("no \"login\" found in %v", member)
		}
		login, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("\"login\" %s found in %v, but not a string", value, member)
		}
		members = append(members, login)
	}
	return members, nil
}
