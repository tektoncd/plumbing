package tep

import (
	"fmt"
	"strings"
)

const (
	// NewStatus is used for unmerged TEPs.
	NewStatus Status = "new"
	// ProposedStatus is the "proposed" TEP status.
	ProposedStatus Status = "proposed"
	// ImplementableStatus is the "implementable" status.
	ImplementableStatus Status = "implementable"
	// ImplementingStatus is the "implementing" status.
	ImplementingStatus Status = "implementing"
	// ImplementedStatus is the "implemented" status.
	ImplementedStatus Status = "implemented"
	// WithdrawnStatus is the "withdrawn" status.
	WithdrawnStatus Status = "withdrawn"
	// ReplacedStatus is the "replaced" status.
	ReplacedStatus Status = "replaced"

	// TrackingIssueStatusLabelPrefix is the prefix added to the Status to generate the label on GitHub for tracking issues.
	TrackingIssueStatusLabelPrefix = "tep-status/"
)

var (
	// Statuses is a list of all valid TEP statuses
	Statuses = []Status{
		NewStatus,
		ProposedStatus,
		ImplementableStatus,
		ImplementingStatus,
		ImplementedStatus,
		WithdrawnStatus,
		ReplacedStatus,
	}
)

// Status is a valid TEP status
type Status string

// TrackingLabel returns "tep-status/[status string]" for a status.
func (s Status) TrackingLabel() string {
	return fmt.Sprintf("%s%s", TrackingIssueStatusLabelPrefix, s)
}

// ForMarkdown returns the status surrounded by backticks for use in GitHub comments, issue bodies, etc.
func (s Status) ForMarkdown() string {
	return fmt.Sprintf("`%s`", s)
}

// FromTrackingIssueLabel extracts the status from the "tep-status/whatever" label and returns the appropriate Status.
// If the result isn't a valid status, it returns nil.
func FromTrackingIssueLabel(label string) *Status {
	withoutPrefix := strings.TrimPrefix(label, TrackingIssueStatusLabelPrefix)
	if IsValidStatus(withoutPrefix) {
		asStatus := Status(withoutPrefix)
		return &asStatus
	}

	return nil
}

// IsValidStatus returns true if the given string is a known status type
func IsValidStatus(s string) bool {
	for _, status := range Statuses {
		if s == string(status) {
			return true
		}
	}

	return false
}

// GetTEPsWithStatus filters a map of TEP ID to TEPInfo for all TEPs in a given status
func GetTEPsWithStatus(input map[string]TEPInfo, desiredStatus Status) map[string]TEPInfo {
	teps := make(map[string]TEPInfo)

	for k, v := range input {
		if v.Status == desiredStatus {
			teps[k] = v
		}
	}

	return teps
}
