package main

import (
	"flag"
	"github.com/tektoncd/plumbing/tep-notifier/pkg/reconciler"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"
	"log"
)

var (
	ghToken = flag.String("github-token", "", "GitHub OAuth token for interacting with GitHub")
)

func main() {
	flag.Parse()

	if ghToken == nil {
		log.Fatal("no github-token specified")
	}
	sharedmain.MainWithContext(signals.NewContext(), reconciler.ControllerName, reconciler.NewController(*ghToken))
}
