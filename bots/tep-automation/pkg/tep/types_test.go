package tep_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/tep"
)

func TestTrackingIssue_GetBody(t *testing.T) {
	ti := tep.TrackingIssue{
		IssueNumber: 5,
		IssueState:  "open",
		TEPStatus:   tep.ProposedStatus,
		TEPID:       "1234",
		TEPPRs:      []int{100, 120},
		ImplementationPRs: []tep.ImplementationPR{
			{
				Repo:   "pipeline",
				Number: 77,
			},
			{
				Repo:   "triggers",
				Number: 88,
			},
		},
		Assignees: []string{"abayer", "vdemeester"},
	}

	mdFilename := "1234-some-tep.md"

	expected := `This issue tracks TEP-1234.

Use this issue for discussion of this TEP not directly related to pull requests updating or implementing the TEP.

TEP: (https://github.com/tektoncd/community/blob/main/teps/1234-some-tep.md)
Description: (specify a one-line description of the TEP)
Current status: ` + "`proposed`" + `
Authors:
- @abayer
- @vdemeester
Project(s): (unspecified)
Release Targets:
* Alpha: (unspecified)
* Beta: (unspecified)

TEP PRs:
- (https://github.com/tektoncd/community/pull/100)
- (https://github.com/tektoncd/community/pull/120)
Implementation PRs:
- (https://github.com/tektoncd/pipeline/pull/77)
- (https://github.com/tektoncd/triggers/pull/88)

<!-- TEP PR: 100 -->
<!-- TEP PR: 120 -->

<!-- Implementation PR: repo: pipeline number: 77 -->
<!-- Implementation PR: repo: triggers number: 88 -->
`

	body, err := ti.GetBody(mdFilename)
	require.NoError(t, err)
	assert.Equal(t, expected, body)
}
