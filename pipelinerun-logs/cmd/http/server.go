package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"github.com/tektoncd/plumbing/pipelinerun-logs/pkg/config"
	"golang.org/x/xerrors"
	"google.golang.org/api/iterator"
)

type Server struct {
	conf        *config.Config
	client      *logging.Client
	adminClient *logadmin.Client
	entriesTmpl *template.Template
}

type EntriesTemplateContext struct {
	LogsJSON     []RenderableEntry
	BuildID      string
	PipelineName string
}

const (
	EntryParseErrorMessage           = "unable to parse entry"
	MaxFetchedLogEntries             = 100000
	StackdriverContainerNameLabel    = "container_name"
	StackdriverContainerResourceType = "k8s_container"
	TektonPipelineNameLabel          = "k8s-pod/tekton_dev/pipeline"
	TektonTaskNameLabel              = "k8s-pod/tekton_dev/task"
)

var (
	prowBuildIDPattern = regexp.MustCompile(`[0-9]+`)
	uuidBuildIDPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
)

// NewServer returns an instance of Server configured with provided params.
func NewServer(conf *config.Config, client *logging.Client, adminClient *logadmin.Client, templatePath string) *Server {
	return &Server{
		conf:        conf,
		client:      client,
		adminClient: adminClient,
		entriesTmpl: template.Must(template.ParseFiles(templatePath)),
	}
}

// Start begins serving logs over http
func (s *Server) Start() {
	http.HandleFunc("/", s.serveLog)
	addr := fmt.Sprintf("%s:%s", s.conf.Hostname, s.conf.Port)
	log.Printf("Serving %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// serveLog serves up an html page with log entries rendered as a JSON object
// in the head of the document.
func (s *Server) serveLog(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s?%s", r.URL.Path, r.URL.RawQuery)

	buildID := r.URL.Query().Get("buildid")
	if err := s.validateBuildID(buildID); err != nil {
		log.Printf("%v", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	query := &Query{
		Project:   s.conf.Project,
		Cluster:   s.conf.Cluster,
		Namespace: s.conf.Namespace,
		BuildID:   buildID,
	}

	if err := query.Validate(); err != nil {
		log.Printf("%v", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ctx := context.Background()

	entries, err := s.fetchAllEntries(ctx, query)
	if err != nil {
		log.Printf("%v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	j, err := s.logsToJSON(ctx, entries)
	if err != nil {
		log.Printf("error building json for logs: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tc := &EntriesTemplateContext{
		LogsJSON:     j,
		BuildID:      query.BuildID,
		PipelineName: getPipelineName(entries),
	}

	if err := s.entriesTmpl.Execute(w, tc); err != nil {
		log.Printf("error executing template: %v", err)
	}
}

// logsToJSON converts a list of logging entries into a format suitable to
// provide to a frontend.
func (s *Server) logsToJSON(ctx context.Context, entries []*logging.Entry) ([]RenderableEntry, error) {
	out := make([]RenderableEntry, 0)
	for _, entry := range entries {
		s, err := s.structureEntry(entry)
		if err != nil {
			return nil, xerrors.Errorf("error structuring log entry: %w", err)
		}
		out = append(out, *s)
	}
	return out, nil
}

// fetchAllEntries iterates over paginated log entries from stackdriver
// and returns the complete list.
func (s *Server) fetchAllEntries(ctx context.Context, query *Query) ([]*logging.Entry, error) {
	filter := query.ToFilter()
	iter := s.adminClient.Entries(ctx, logadmin.Filter(filter))
	var entries []*logging.Entry
	var err error
	var count int
	for err != iterator.Done && count < MaxFetchedLogEntries {
		var entry *logging.Entry
		if entry, err = iter.Next(); entry != nil {
			entries = append(entries, entry)
		}
		count++
	}
	if err != nil && err != iterator.Done {
		return nil, xerrors.Errorf("error iterating log entries: %w", err)
	}
	return entries, nil
}

// structureEntry takes a logging Entry and extracts the fields necessary
// to provide to the frontend for rendering.
func (s *Server) structureEntry(entry *logging.Entry) (*RenderableEntry, error) {
	ep, err := parseEntryPayload(entry)
	if err != nil {
		return nil, err
	}
	entryPrefix := fmt.Sprintf("projects/%s/logs/", s.conf.Project)
	logName := strings.TrimPrefix(entry.LogName, entryPrefix)
	return &RenderableEntry{
		TaskName:      entry.Labels[TektonTaskNameLabel],
		LogName:       logName,
		Message:       ep.Fields.Msg.Kind.StringValue,
		Caller:        ep.Fields.Caller.Kind.StringValue,
		ContainerName: extractContainerName(entry),
		TimeStamp:     entry.Timestamp.UTC().Format(time.RFC3339),
	}, nil
}

func (s *Server) validateBuildID(buildID string) error {
	if uuidBuildIDPattern.MatchString(buildID) || prowBuildIDPattern.MatchString(buildID) {
		return nil
	}
	return fmt.Errorf("build id not formatted as prow id or uuid: %q", buildID)
}

// parseEntryPayload takes a stackdriver logging entry and parses out the payload
// fields we're interested in returning to the user.
func parseEntryPayload(entry *logging.Entry) (*EntryPayload, error) {
	content, isString := entry.Payload.(string)
	var ep EntryPayload
	if isString {
		if err := json.Unmarshal([]byte(content), &ep); err != nil {
			// payload is just a vanilla log string with no json content
			ep.Fields.Msg.Kind.StringValue = content
		}
	} else {
		if contentBytes, err := json.Marshal(entry.Payload); err != nil {
			return nil, err
		} else if err := json.Unmarshal(contentBytes, &ep); err != nil {
			return nil, err
		}
	}
	return &ep, nil
}

// extractContainerName returns the container name from the labels of the
// stackdriver resource if one is available or an empty string.
func extractContainerName(entry *logging.Entry) string {
	if entry.Resource.Type == StackdriverContainerResourceType {
		return entry.Resource.GetLabels()[StackdriverContainerNameLabel]
	}
	return ""
}

// getPipelineName fetches the pipeline name from a list of logging entries if any of them
// provide it in their labels.
func getPipelineName(entries []*logging.Entry) string {
	for _, e := range entries {
		p := e.Labels[TektonPipelineNameLabel]
		if p != "" {
			return p
		}
	}
	return ""
}
