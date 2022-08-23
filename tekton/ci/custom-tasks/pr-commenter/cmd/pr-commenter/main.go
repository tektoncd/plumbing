package main

import (
	"log"
	"os"

	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/tektoncd/plumbing/tekton/ci/custom-tasks/pr-commenter/pkg/reconciler"
	"knative.dev/pkg/injection/sharedmain"
)

func main() {
	scmClient, err := factory.NewClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	owner := os.Getenv("GIT_OWNER")
	if owner == "" {
		log.Fatal("GIT_OWNER env var required")
	}
	botUser := os.Getenv("GIT_USER")
	if botUser == "" {
		botUser = "tekton-robot"
	}

	sharedmain.Main(reconciler.ControllerName, reconciler.NewController(scmClient, botUser, owner))
}
