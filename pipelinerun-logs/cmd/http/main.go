package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"github.com/tektoncd/plumbing/pipelinerun-logs/pkg/config"
)

func main() {
	conf := &config.Config{}
	conf.ParseFlags()

	if err := conf.Validate(); err != nil {
		log.Printf("%v", err)
		flag.PrintDefaults()
		os.Exit(1)
		return
	}

	ctx := context.Background()

	client, err := logging.NewClient(ctx, conf.Project)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	client.OnError = func(err error) {
		log.Printf("client.OnError: %v", err)
	}

	adminClient, err := logadmin.NewClient(ctx, conf.Project)
	if err != nil {
		log.Fatalf("failed to create adminClient: %v", err)
	}

	// When building with "ko", templates is deployed under KO_DATA_PATH
	// If KO_DATA_PATH is not defined, the path will be a relatove one
	basePath := os.Getenv("KO_DATA_PATH")
	entries := path.Join(basePath, "templates/entries.html")

	server := NewServer(conf, client, adminClient, entries)
	server.Start()
}
