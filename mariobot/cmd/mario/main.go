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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"github.com/google/uuid"
)

const (
	// Environment variable containing GitHub secret token
	envSecret = "GITHUB_SECRET_TOKEN"
	// Environment variable containing the target container registry
	envRegistry = "CONTAINER_REGISTRY"
)

type triggerPayload struct {
	BuildUUID     string `json:"buildUUID,omitempty"`
	GitRepository string `json:"gitRepository,omitempty"`
	GitRevision   string `json:"gitRevision,omitempty"`
	ContextPath   string `json:"contextPath,omitempty"`
	TargetImage   string `json:"targetImage,omitempty"`
	PullRequestID string `json:"pullRequestID,omitempty"`
}

func main() {
	errorMessage := ""
	secretToken := os.Getenv(envSecret)
	if secretToken == "" {
		log.Fatalf("No secret token given")
	}
	registry := os.Getenv(envRegistry)
	if secretToken == "" {
		registry = "gcr.io/tekton-releases/dogfooding"
	}

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		//TODO: We should probably send over the EL eventID as a X-Tekton-Event-Id header as well
		payload, err := github.ValidatePayload(request, []byte(secretToken))
		id := github.DeliveryID(request)
		if err != nil {
			log.Printf("Error handling Github Event with delivery ID %s : %q", id, err)
			http.Error(writer, fmt.Sprint(err), http.StatusBadRequest)
		}
		event, err := github.ParseWebHook(github.WebHookType(request), payload)
		if err != nil {
			log.Printf("Error handling Github Event with delivery ID %s : %q", id, err)
			http.Error(writer, fmt.Sprint(err), http.StatusBadRequest)
		}
		switch event := event.(type) {
		case *github.IssueCommentEvent:
			if event.GetAction() == "created" {
				eventBody := event.GetComment().GetBody()
				if strings.HasPrefix(eventBody, "/mario") {
					log.Printf("Handling Mario command with delivery ID: %s; Comment: %s", id, eventBody)
					commandParts := strings.Fields(eventBody)
					command := commandParts[1]
					if command == "build" {
						// No validation here. Anything beyond commandParts[3] is ignored
						prID := strconv.Itoa(int(event.GetIssue().GetNumber()))
						triggerBody := triggerPayload{
							BuildUUID:     uuid.New().String(),
							GitRepository: "github.com/" + event.GetRepo().GetFullName(),
							GitRevision:   "pull/" + prID + "/head",
							ContextPath:   commandParts[2],
							TargetImage:   registry + commandParts[3],
							PullRequestID: prID,
						}
						tPayload, err := json.Marshal(triggerBody)
						if err != nil {
							log.Printf("Failed to marshal the trigger body. Error: %q", err)
						}
						log.Printf("Replying with payload %s", payload)
						n, err := writer.Write(tPayload)
						if err != nil {
							log.Printf("Failed to write response for Github event ID: %s. Bytes writted: %d. Error: %q", id, n, err)
						}
					} else {
						errorMessage = "Unknown Mario command"
					}
				} else {
					errorMessage = "Not a Mario command"
				}
			} else {
				errorMessage = "Only new comments are supported"
			}
		default:
			errorMessage = "Event type not supported"
		}
		if errorMessage != "" {
			log.Printf(errorMessage)
			http.Error(writer, fmt.Sprint(errorMessage), http.StatusBadRequest)
		}
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", 8080), nil))
}
