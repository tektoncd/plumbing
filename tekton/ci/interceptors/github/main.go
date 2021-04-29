package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/tektoncd/plumbing/tekton/ci/interceptors/github/pkg/github"
)

var (
	webhookSecretPath = flag.String("webhook_secret_path", "", "path to file containing webhook secret to validate")
)

func main() {
	flag.Parse()

	var webhookSecret []byte
	if *webhookSecretPath != "" {
		var err error
		webhookSecret, err = ioutil.ReadFile(*webhookSecretPath)
		if err != nil {
			log.Fatal(err)
		}
	}
	s := github.New(http.DefaultClient, bytes.TrimSpace(webhookSecret))

	http.ListenAndServe(":8080", s)
}
