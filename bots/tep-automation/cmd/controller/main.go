package main

import (
	"flag"
	"log"

	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/performers"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/reconciler"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"
)

var (
	ghToken = flag.String("github-token", "", "GitHub OAuth token for interacting with GitHub")
)

func main() {
	flag.Parse()

	if ghToken == nil {
		log.Fatal("no github-token specified")
	}

	sharedmain.MainWithContext(signals.NewContext(), reconciler.ControllerName,
		reconciler.NewController(
			*ghToken,
			performers.NewPRNotifierAsPerformer,
			performers.NewIssueCreatorAsPerformer,
		))
}
