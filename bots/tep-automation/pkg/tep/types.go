package tep

import (
	"bytes"
	"fmt"
	"text/template"
	"time"
)

const (
	// ActionParamName is the param name that will hold the pull request webhook action
	ActionParamName = "pullRequestAction"
	// PRNumberParamName is the param name that will hold the pull request number
	PRNumberParamName = "pullRequestNumber"
	// PRTitleParamName is the param name that will hold the pull request title
	PRTitleParamName = "pullRequestTitle"
	// PRBodyParamName is the param name that will hold the pull request body
	PRBodyParamName = "pullRequestBody"
	// PackageParamName is the param name that will hold the pull request's repository owner/name
	PackageParamName = "package"
	// PRIsMergedParamName is the param name that will hold whether the pull request is merged
	PRIsMergedParamName = "pullRequestIsMerged"
	// GitRevisionParamName is the param name that will hold the HEAD git SHA for the PR
	GitRevisionParamName = "gitRevision"

	// CommunityBlobBaseURL is the base URL for links to blobs in github.com/tektoncd/community/teps
	CommunityBlobBaseURL = "https://github.com/tektoncd/community/blob/main/teps/"

	// DefaultTrackingIssueDescription is used in the tracking issue when no description already exists in the issue body.
	DefaultTrackingIssueDescription = "(specify a one-line description of the TEP)"
	// DefaultTrackingIssueField is used in the tracking issue when no release target (for either alpha or beta)
	// or project(s) already exists in the issue body.
	DefaultTrackingIssueField = "(unspecified)"

	// TrackingIssueBodyTmpl is the template (using text/template) for the body of tracking issues.
	TrackingIssueBodyTmpl = `This issue tracks TEP-{{ .issue.TEPID }}.

Use this issue for discussion of this TEP not directly related to pull requests updating or implementing the TEP.

TEP: ({{ .tepURL }})
Description: {{ .description }}
Current status: {{ .mdStatus }}
Authors:{{ range .issue.Assignees }}
- @{{ . }}{{ end }}
Project(s): {{ .projects }}
Release Targets:
* Alpha: {{ .alphaTarget }}
* Beta: {{ .betaTarget }}

TEP PRs:{{ range .issue.TEPPRs }}
- (https://github.com/tektoncd/community/pull/{{ . }}){{ end }}
{{ if .issue.ImplementationPRs }}Implementation PRs:{{ range .issue.ImplementationPRs }}
- (https://github.com/tektoncd/{{ .Repo }}/pull/{{ .Number }}){{ end }}{{ end }}
{{ range .issue.TEPPRs }}
<!-- TEP PR: {{ . }} -->{{ end }}
{{ range .issue.ImplementationPRs }}
<!-- Implementation PR: repo: {{ .Repo }} number: {{ .Number }} -->{{ end }}
`
)

// CommentInfo stores the PR comment ID for a comment the notifier made about a TEP, the TEP's status when the
// comment was made, and the TEP ID
type CommentInfo struct {
	CommentID     int64
	TEPs          []string
	ToImplemented bool
}

// TEPInfo stores information about a TEP parsed from https://github.com/tektoncd/community/blob/main/teps/README.md
type TEPInfo struct {
	ID           string
	Title        string
	Status       Status
	Filename     string
	LastModified time.Time
	Authors      []string
}

// ImplementationPR stores a TEP implementation PR's repository and PR number
type ImplementationPR struct {
	Repo   string
	Number int
}

// TrackingIssue stores information about a TEP tracking issue in the community repo.
type TrackingIssue struct {
	IssueNumber       int
	IssueState        string
	TEPStatus         Status
	TEPID             string
	TEPPRs            []int
	ImplementationPRs []ImplementationPR
	Assignees         []string
	Description       string
	AlphaTarget       string
	BetaTarget        string
	Projects          string
}

// AddAssignee adds an assignee to the tracking issue, if it's not already present
func (ti *TrackingIssue) AddAssignee(assignee string) {
	present := false
	for _, n := range ti.Assignees {
		if n == assignee {
			present = true
			break
		}
	}

	if !present {
		ti.Assignees = append(ti.Assignees, assignee)
	}
}

// AddTEPPR adds a TEP PR to the tracking issue, if it's not already present
func (ti *TrackingIssue) AddTEPPR(number int) {
	present := false
	for _, n := range ti.TEPPRs {
		if n == number {
			present = true
			break
		}
	}

	if !present {
		ti.TEPPRs = append(ti.TEPPRs, number)
	}
}

// AddImplementationPR adds an implementation PR to the tracking issue, if it's not already present
func (ti *TrackingIssue) AddImplementationPR(repo string, number int) {
	present := false
	for _, i := range ti.ImplementationPRs {
		if i.Number == number && i.Repo == repo {
			present = true
			break
		}
	}

	if !present {
		ti.ImplementationPRs = append(ti.ImplementationPRs, ImplementationPR{
			Repo:   repo,
			Number: number,
		})
	}
}

// GetBody returns the body content for this tracking issue.
func (ti *TrackingIssue) GetBody(filename string) (string, error) {
	data := map[string]interface{}{
		"issue":    ti,
		"tepURL":   fmt.Sprintf("%s%s", CommunityBlobBaseURL, filename),
		"mdStatus": ti.TEPStatus.ForMarkdown(),
	}
	if ti.Description != "" {
		data["description"] = ti.Description
	} else {
		data["description"] = DefaultTrackingIssueDescription
	}
	if ti.AlphaTarget != "" {
		data["alphaTarget"] = ti.AlphaTarget
	} else {
		data["alphaTarget"] = DefaultTrackingIssueField
	}
	if ti.BetaTarget != "" {
		data["betaTarget"] = ti.BetaTarget
	} else {
		data["betaTarget"] = DefaultTrackingIssueField
	}
	if ti.Projects != "" {
		data["projects"] = ti.Projects
	} else {
		data["projects"] = DefaultTrackingIssueField
	}

	buf := bytes.NewBufferString("")
	if bodyTmpl, err := template.New("trackingIssueBody").Parse(TrackingIssueBodyTmpl); err != nil {
		return "", fmt.Errorf("failed to parse template for tracking issue body: %v", err)
	} else if err := bodyTmpl.Execute(buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template for tracking issue body: %v", err)
	}
	return buf.String(), nil
}
