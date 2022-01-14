package tep_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/tep"
)

func TestExtractTEPsFromReadme(t *testing.T) {
	testCases := []struct {
		name     string
		body     string
		expected map[string]tep.TEPInfo
		errStr   string
	}{
		{
			name:     "no TEPs",
			body:     "there are no teps here",
			expected: make(map[string]tep.TEPInfo),
		},
		{
			name: "single TEP",
			body: `there's one tep in here
on a later line
|[TEP-1234](1234-something-or-other.md) | Some TEP Title | proposed | 2021-12-20 |
|[TEP-5678](5678-not-valid-line.md) | | proposed | 2021-12-20 |
tada, a single valid TEP and a bogus line
`,
			expected: map[string]tep.TEPInfo{
				"1234": {
					ID:           "1234",
					Title:        "Some TEP Title",
					Status:       tep.ProposedStatus,
					Filename:     "1234-something-or-other.md",
					LastModified: time.Date(2021, time.December, 20, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "multiple TEPs",
			body: `there are two teps in here
on later lines
|[TEP-1234](1234-something-or-other.md) | Some TEP Title | proposed | 2021-12-20 |
|[TEP-5678](5678-valid-line-this-time.md) | A Second TEP Title | implemented | 2021-12-29 |
tada, a single valid TEP and a bogus line
`,
			expected: map[string]tep.TEPInfo{
				"1234": {
					ID:           "1234",
					Title:        "Some TEP Title",
					Status:       tep.ProposedStatus,
					Filename:     "1234-something-or-other.md",
					LastModified: time.Date(2021, time.December, 20, 0, 0, 0, 0, time.UTC),
				},
				"5678": {
					ID:           "5678",
					Title:        "A Second TEP Title",
					Status:       tep.ImplementedStatus,
					Filename:     "5678-valid-line-this-time.md",
					LastModified: time.Date(2021, time.December, 29, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name:   "invalid date",
			body:   "|[TEP-1234](1234-something-or-other.md) | Some TEP Title | proposed | 2021-12-40 |",
			errStr: `parsing time "2021-12-40T00:00:00Z": day out of range`,
		},
		{
			name:   "invalid status",
			body:   "|[TEP-1234](1234-something-or-other.md) | Some TEP Title | invalid-status | 2021-12-20 |",
			errStr: "invalid-status is not a valid status",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			teps, err := tep.ExtractTEPsFromReadme(tc.body)
			if tc.errStr != "" {
				require.EqualError(t, err, tc.errStr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.expected, teps)
		})
	}
}

func TestGetTEPIDsFromPR(t *testing.T) {
	testCases := []struct {
		name  string
		title string
		body  string
		ids   []string
	}{
		{
			name:  "none",
			title: "Some PR title",
			body:  "Some PR body",
		},
		{
			name:  "one id in title",
			title: "This implements TEP-1234 perhaps",
			body:  "Some PR body",
			ids:   []string{"1234"},
		},
		{
			name:  "one id in body",
			title: "Some PR title",
			body:  "This implements TEP-1234 perhaps",
			ids:   []string{"1234"},
		},
		{
			name:  "one url in title",
			title: "This is for https://github.com/tektoncd/community/blob/main/teps/0002-custom-tasks.md",
			body:  "Some PR body",
			ids:   []string{"0002"},
		},
		{
			name:  "one url in body",
			title: "Some PR title",
			body:  "This is for https://github.com/tektoncd/community/blob/main/teps/0002-custom-tasks.md",
			ids:   []string{"0002"},
		},
		{
			name:  "one id in title, one url in body",
			title: "This is for TEP-1234",
			body:  "This is for https://github.com/tektoncd/community/blob/main/teps/0002-custom-tasks.md",
			ids:   []string{"0002", "1234"},
		},
		{
			name:  "two urls in body",
			title: "Some PR title",
			body:  "This is for https://github.com/tektoncd/community/blob/main/teps/0002-custom-tasks.md and https://github.com/tektoncd/community/blob/main/teps/0006-tekton-metrics.md",
			ids:   []string{"0002", "0006"},
		},
		{
			name:  "two ids in body",
			title: "Some PR title",
			body:  "This implements TEP-1234 perhaps. And what the heck, TEP-5678 as well.",
			ids:   []string{"1234", "5678"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			found := tep.GetTEPIDsFromPR(tc.title, tc.body)
			assert.ElementsMatch(t, tc.ids, found)
		})
	}
}

func TestGetTEPCommentDetails(t *testing.T) {
	testCases := []struct {
		name          string
		comment       string
		teps          map[string]string
		toImplemented bool
	}{
		{
			name:    "none",
			comment: "this has no TEPs",
			teps:    make(map[string]string),
		},
		{
			name: "one TEP",
			comment: `this comment has some text
and it also has a TEP

<!-- TEP Notifier Action: implemented -->

<!-- TEP update: TEP-1234 status: implementing -->
`,
			toImplemented: true,
			teps: map[string]string{
				"1234": "implementing",
			},
		},
		{
			name: "two TEPs",
			comment: `this comment has some text
and it also has two TEPs

<!-- TEP Notifier Action: implementing -->

<!-- TEP update: TEP-1234 status: proposed -->
<!-- TEP update: TEP-5678 status: implementable -->
`,
			teps: map[string]string{
				"1234": string(tep.ProposedStatus),
				"5678": string(tep.ImplementableStatus),
			},
			toImplemented: false,
		},
		{
			name: "duplicate TEP",
			comment: `this comment has some text
and it also has one TEP, duplicated with a different status

<!-- TEP Notifier Action: implementing -->

<!-- TEP update: TEP-1234 status: proposed -->
<!-- TEP update: TEP-1234 status: implementable -->
`,
			teps: map[string]string{
				"1234": string(tep.ImplementableStatus),
			},
			toImplemented: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			teps, toImplemented := tep.GetTEPCommentDetails(tc.comment)
			if d := cmp.Diff(tc.teps, teps); d != "" {
				t.Errorf("Wrong TEPs from comment: (-want, +got): %s", d)
			}
			assert.Equalf(t, tc.toImplemented, toImplemented, "expected toImplemented to be %t, but is %t", tc.toImplemented, toImplemented)
		})
	}
}

func TestTEPInfoFromMarkdown(t *testing.T) {
	mdFile := filepath.Join("testdata", "markdown", "0014-step-timeout.md")
	mdContent, err := ioutil.ReadFile(mdFile)
	require.NoError(t, err)

	info, err := tep.TEPInfoFromMarkdown("0014", "0014-step-timeout.md", string(mdContent))
	require.NoError(t, err)

	assert.Equal(t, "0014", info.ID)
	assert.Equal(t, "0014-step-timeout.md", info.Filename)
	assert.Equal(t, "Step Timeout", info.Title)
	assert.Equal(t, time.Date(2021, time.December, 13, 0, 0, 0, 0, time.UTC), info.LastModified)
	assert.Equal(t, tep.ImplementedStatus, info.Status)
	assert.ElementsMatch(t, []string{"Peaorl"}, info.Authors)
}

func TestPRsForTrackingIssue(t *testing.T) {
	testCases := []struct {
		name        string
		body        string
		tepPRs      []int
		implPRs     []tep.ImplementationPR
		desc        string
		alphaTarget string
		betaTarget  string
		projects    string
		expectedErr error
	}{
		{
			name: "none",
			body: "nothing to see here",
		},
		{
			name: "one of each",
			body: `<!-- TEP PR: 55 -->
<!-- Implementation PR: repo: pipeline number: 77 -->`,
			tepPRs: []int{55},
			implPRs: []tep.ImplementationPR{{
				Repo:   "pipeline",
				Number: 77,
			}},
		},
		{
			name:   "one TEP PR",
			body:   "<!-- TEP PR: 55 -->",
			tepPRs: []int{55},
		},
		{
			name: "one implementation PR",
			body: "<!-- Implementation PR: repo: pipeline number: 77 -->",
			implPRs: []tep.ImplementationPR{{
				Repo:   "pipeline",
				Number: 77,
			}},
		},
		{
			name: "multiple of each",
			body: `<!-- TEP PR: 55 -->
<!-- TEP PR: 66 -->
<!-- Implementation PR: repo: pipeline number: 77 -->
<!-- Implementation PR: repo: triggers number: 88 -->`,
			tepPRs: []int{55, 66},
			implPRs: []tep.ImplementationPR{
				{
					Repo:   "pipeline",
					Number: 77,
				},
				{
					Repo:   "triggers",
					Number: 88,
				},
			},
		},
		{
			name: "not a number for TEP PR",
			body: "<!-- TEP PR: ajaj -->",
		},
		{
			name: "not a number for implementation PR",
			body: "<!-- Implementation PR: repo: pipeline number: ajaj -->",
		},
		{
			name: "with other fields",
			body: `Some line
Description: something
* Alpha: 1.0.0
* Beta: 2.0.0
Some other line
Projects: pipeline, triggers
`,
			desc:        "something",
			alphaTarget: "1.0.0",
			betaTarget:  "2.0.0",
			projects:    "pipeline, triggers",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tepPRs, implPRs, desc, alphaTarget, betaTarget, projects, err := tep.ParseTrackingIssue(tc.body)
			if tc.expectedErr != nil {
				require.Equal(t, tc.expectedErr, err)
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, tc.tepPRs, tepPRs)
				assert.Equal(t, tc.implPRs, implPRs)
				assert.Equal(t, tc.desc, desc)
				assert.Equal(t, tc.alphaTarget, alphaTarget)
				assert.Equal(t, tc.betaTarget, betaTarget)
				assert.Equal(t, tc.projects, projects)
			}
		})
	}
}
