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
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/v29/github"
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

type triggerErrorPayload struct {
	Error string `json:"errorMessage,omitempty"`
}

const defaultRegistry = "gcr.io/tekton-releases/dogfooding"

func main() {
	secretToken := os.Getenv(envSecret)
	if secretToken == "" {
		log.Fatalf("No secret token given")
	}
	registry := os.Getenv(envRegistry)
	if registry == "" {
		registry = defaultRegistry
	}

	http.HandleFunc("/", makeMarioHandler(secretToken, registry))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", 8080), nil))
}

func makeMarioHandler(secret, registry string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//TODO: We should probably send over the EL eventID as a X-Tekton-Event-Id header as well
		payload, err := github.ValidatePayload(r, []byte(secret))
		id := github.DeliveryID(r)
		if err != nil {
			log.Printf("error handling Github Event with delivery ID %s : %q", id, err)
			marshalError(err, w)
			return
		}
		event, err := github.ParseWebHook(github.WebHookType(r), payload)
		if err != nil {
			log.Printf("error handling Github Event with delivery ID %s : %q", id, err)
			marshalError(err, w)
			return
		}

		var handlingErr error
		switch event := event.(type) {
		case *github.IssueCommentEvent:
			handlingErr = handleIssueComment(id, registry, event, w)
		default:
			handlingErr = errors.New("event type not supported")
		}

		if handlingErr != nil {
			marshalError(handlingErr, w)
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

func handleIssueComment(id string, registry string, evt *github.IssueCommentEvent, w http.ResponseWriter) error {
	if evt.GetAction() != "created" {
		return errors.New("only new comments are supported")
	}
	evtBody := evt.GetComment().GetBody()
	if !strings.HasPrefix(evtBody, "/mario") {
		return errors.New("not a Mario command")
	}
	log.Printf("handling Mario command with delivery ID: %s; Comment: %s", id, evtBody)
	commandParts := strings.Fields(evtBody)
	command := commandParts[1]
	switch command {
	case "build":
		// No validation here. Anything beyond commandParts[3] is ignored
		prID := strconv.Itoa(int(evt.GetIssue().GetNumber()))
		triggerBody := triggerPayload{
			BuildUUID:     uuid.New().String(),
			GitRepository: "github.com/" + evt.GetRepo().GetFullName(),
			GitRevision:   "pull/" + prID + "/head",
			ContextPath:   commandParts[2],
			TargetImage:   registry + "/" + commandParts[3],
			PullRequestID: prID,
		}
		tPayload, err := json.Marshal(triggerBody)
		if err != nil {
			log.Printf("Failed to marshal the trigger body. Error: %q", err)
		}
		log.Printf("Replying with payload %s", tPayload)
		n, err := w.Write(tPayload)
		if err != nil {
			log.Printf("Failed to write response for Github evt ID: %s. Bytes writted: %d. Error: %q", id, n, err)
		}
	default:
		return errors.New("unknown Mario command")
	}
	return nil
}
