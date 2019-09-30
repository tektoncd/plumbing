package main

import (
	"errors"
	"fmt"
	"regexp"
)

const (
	StackdriverBuildIDLabel = "k8s-pod/prow_k8s_io/build-id"
)

var (
	buildIDRegex = regexp.MustCompile(`^[0-9]+$`)
)

type Query struct {
	Project   string
	Cluster   string
	Namespace string
	BuildID   string
}

// Validate ensures that required information for a query is provided
// and returns an error otherwise.
func (q *Query) Validate() error {
	if q.Project == "" {
		return errors.New("Invalid query: missing project")
	}
	if q.Cluster == "" {
		return errors.New("Invalid query: missing cluster")
	}
	if q.Namespace == "" {
		return errors.New("Invalid query: missing namespace")
	}
	if q.BuildID == "" {
		return errors.New("Invalid query: missing build id")
	}
	if !buildIDRegex.MatchString(q.BuildID) {
		return errors.New("Invalid query: build id does not match expected pattern")
	}
	return nil
}

// ToFilter returns a stackdriver filter string that is populated
// with data from the query.
func (q *Query) ToFilter() string {
	return fmt.Sprintf(`
resource.type=k8s_container
AND (
	logName=projects/%s/logs/stderr
	OR logName=projects/%s/logs/stdout
)
AND resource.labels.cluster_name=%q
AND resource.labels.namespace_name=%q
AND labels.%q=%q
`,
		q.Project,
		q.Project,
		q.Cluster,
		q.Namespace,
		StackdriverBuildIDLabel,
		q.BuildID,
	)
}
