package performers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v41/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/ghclient"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/performers"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/tep"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/testutil"
	corev1 "k8s.io/api/core/v1"
	kreconciler "knative.dev/pkg/reconciler"
)

type expectedIssue struct {
	tep.TrackingIssue
	filename string
}

type closedIssue struct {
	number  int
	comment string
}

func (ei expectedIssue) toIssueRequest(t *testing.T) *github.IssueRequest {
	body, err := ei.GetBody(ei.filename)
	require.NoError(t, err)

	return &github.IssueRequest{
		Title: github.String(fmt.Sprintf(ghclient.TrackingIssueTitleFmt, ei.TEPID)),
		Body:  github.String(body),
		Labels: &[]string{
			ghclient.TrackingIssueLabel,
			ei.TEPStatus.TrackingLabel(),
		},
		Assignees: &ei.Assignees,
	}
}

func TestIssueCreator_Perform(t *testing.T) {
	contentsRef := "some-ref"

	testCases := []struct {
		name              string
		prRepo            string
		prNumber          int
		prAction          string
		prIsMerged        bool
		listFilesResponse []*github.CommitFile
		existingIssues    []*github.Issue
		closedIssues      []closedIssue
		createdIssues     []expectedIssue
		modifiedIssues    []expectedIssue
		doesNothing       bool
		expectedEventType string
		expectedReason    string
		expectedErr       error
	}{
		{
			name:        "not a community PR",
			prRepo:      "pipeline",
			doesNothing: true,
		},
		{
			name: "not a TEP PR",
			listFilesResponse: []*github.CommitFile{{
				SHA:      github.String("some-sha"),
				Filename: github.String("some-other-file"),
				Status:   github.String("added"),
			}},
			doesNothing: true,
		},
		{
			name: "new TEP",
			listFilesResponse: []*github.CommitFile{{
				SHA:      github.String("some-sha"),
				Filename: github.String("teps/1234-some-proposal.md"),
				Status:   github.String("added"),
			}},
			createdIssues: []expectedIssue{{
				TrackingIssue: tep.TrackingIssue{
					TEPStatus: tep.NewStatus,
					TEPID:     "1234",
					TEPPRs:    []int{1},
					Assignees: []string{"abayer", "vdemeester"},
				},
				filename: "1234-some-proposal.md",
			}},
			expectedEventType: corev1.EventTypeNormal,
			expectedReason:    "TrackingIssuesUpdatedOrCreated",
		},
		{
			name:     "modified TEP",
			prAction: performers.SynchronizeAction,
			listFilesResponse: []*github.CommitFile{{
				SHA:      github.String("some-sha"),
				Filename: github.String("teps/1234-some-proposal.md"),
				Status:   github.String("modified"),
			}},
			existingIssues: []*github.Issue{{
				Title:  github.String("TEP-1234 Tracking Issue"),
				Number: github.Int(1),
				State:  github.String("open"),
				Assignees: []*github.User{
					{
						Login: github.String("abayer"),
					},
					{
						Login: github.String("vdemeester"),
					},
				},
				Labels: []*github.Label{
					{
						Name: github.String(ghclient.TrackingIssueLabel),
					},
					{
						Name: github.String(tep.NewStatus.TrackingLabel()),
					},
				},
				Body: github.String(`<!-- TEP PR: 55 -->
<!-- Implementation PR: repo: pipeline number: 77 -->
<!-- Implementation PR: repo: triggers number: 88 -->`),
			}},
			modifiedIssues: []expectedIssue{{
				TrackingIssue: tep.TrackingIssue{
					IssueNumber: 1,
					TEPStatus:   tep.NewStatus,
					TEPID:       "1234",
					TEPPRs:      []int{55, 1},
					Assignees:   []string{"abayer", "vdemeester"},
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
				},
				filename: "1234-some-proposal.md",
			}},
			expectedEventType: corev1.EventTypeNormal,
			expectedReason:    "TrackingIssuesUpdatedOrCreated",
		},
		{
			name: "one new, one modified but not in new state",
			listFilesResponse: []*github.CommitFile{
				{
					SHA:      github.String("some-sha"),
					Filename: github.String("teps/1234-some-proposal.md"),
					Status:   github.String("added"),
				},
				{
					SHA:      github.String("some-sha"),
					Filename: github.String("teps/5678-just-cats.md"),
					Status:   github.String("modified"),
				},
			},
			existingIssues: []*github.Issue{{
				Title:  github.String("TEP-5678 Tracking Issue"),
				Number: github.Int(5),
				State:  github.String("open"),
				Assignees: []*github.User{
					{
						Login: github.String("abayer"),
					},
				},
				Labels: []*github.Label{
					{
						Name: github.String(ghclient.TrackingIssueLabel),
					},
					{
						Name: github.String(tep.ImplementableStatus.TrackingLabel()),
					},
				},
				Body: github.String(`<!-- TEP PR: 55 -->
<!-- Implementation PR: repo: pipeline number: 77 -->
<!-- Implementation PR: repo: triggers number: 88 -->`),
			}},
			createdIssues: []expectedIssue{{
				TrackingIssue: tep.TrackingIssue{
					TEPStatus: tep.NewStatus,
					TEPID:     "1234",
					TEPPRs:    []int{1},
					Assignees: []string{"abayer", "vdemeester"},
				},
				filename: "1234-some-proposal.md",
			}},
			expectedEventType: corev1.EventTypeNormal,
			expectedReason:    "TrackingIssuesUpdatedOrCreated",
		},
		{
			name:       "merged PR",
			prAction:   performers.ClosedAction,
			prIsMerged: true,
			listFilesResponse: []*github.CommitFile{{
				SHA:      github.String("some-sha"),
				Filename: github.String("teps/1234-some-proposal.md"),
				Status:   github.String("added"),
			}},
			existingIssues: []*github.Issue{{
				Title:  github.String("TEP-1234 Tracking Issue"),
				Number: github.Int(1),
				State:  github.String("open"),
				Assignees: []*github.User{
					{
						Login: github.String("abayer"),
					},
					{
						Login: github.String("vdemeester"),
					},
				},
				Labels: []*github.Label{
					{
						Name: github.String(ghclient.TrackingIssueLabel),
					},
					{
						Name: github.String(tep.NewStatus.TrackingLabel()),
					},
				},
				Body: github.String(`<!-- TEP PR: 55 -->
<!-- Implementation PR: repo: pipeline number: 77 -->
<!-- Implementation PR: repo: triggers number: 88 -->`),
			}},
			modifiedIssues: []expectedIssue{{
				TrackingIssue: tep.TrackingIssue{
					IssueNumber: 1,
					TEPStatus:   tep.ProposedStatus,
					TEPID:       "1234",
					TEPPRs:      []int{55, 1},
					Assignees:   []string{"abayer", "vdemeester"},
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
				},
				filename: "1234-some-proposal.md",
			}},
			expectedEventType: corev1.EventTypeNormal,
			expectedReason:    "TrackingIssuesUpdatedOrCreated",
		},
		{
			name:     "closed and unmerged PR",
			prAction: performers.ClosedAction,
			listFilesResponse: []*github.CommitFile{{
				SHA:      github.String("some-sha"),
				Filename: github.String("teps/1234-some-proposal.md"),
				Status:   github.String("added"),
			}},
			existingIssues: []*github.Issue{{
				Title:  github.String("TEP-1234 Tracking Issue"),
				Number: github.Int(1),
				State:  github.String("open"),
				Assignees: []*github.User{
					{
						Login: github.String("abayer"),
					},
					{
						Login: github.String("vdemeester"),
					},
				},
				Labels: []*github.Label{
					{
						Name: github.String(ghclient.TrackingIssueLabel),
					},
					{
						Name: github.String(tep.NewStatus.TrackingLabel()),
					},
				},
				Body: github.String(`<!-- TEP PR: 55 -->
<!-- Implementation PR: repo: pipeline number: 77 -->
<!-- Implementation PR: repo: triggers number: 88 -->`),
			}},
			doesNothing: true,
		},
		{
			name:       "merged PR in terminal status",
			prAction:   performers.ClosedAction,
			prIsMerged: true,
			listFilesResponse: []*github.CommitFile{{
				SHA:      github.String("some-sha"),
				Filename: github.String("teps/4321-another-proposal.md"),
				Status:   github.String("modified"),
			}},
			existingIssues: []*github.Issue{{
				Title:  github.String("TEP-4321 Tracking Issue"),
				Number: github.Int(1),
				State:  github.String("open"),
				Assignees: []*github.User{
					{
						Login: github.String("abayer"),
					},
				},
				Labels: []*github.Label{
					{
						Name: github.String(ghclient.TrackingIssueLabel),
					},
					{
						Name: github.String(tep.ImplementingStatus.TrackingLabel()),
					},
				},
				Body: github.String(`<!-- TEP PR: 55 -->
<!-- Implementation PR: repo: pipeline number: 77 -->
<!-- Implementation PR: repo: triggers number: 88 -->`),
			}},
			closedIssues: []closedIssue{{
				comment: "Closing tracking issue for TEP-4321 because it has reached the terminal status `implemented`",
				number:  1,
			}},
			modifiedIssues: []expectedIssue{{
				TrackingIssue: tep.TrackingIssue{
					IssueNumber: 1,
					TEPStatus:   tep.ImplementedStatus,
					TEPID:       "4321",
					TEPPRs:      []int{55, 1},
					Assignees:   []string{"abayer"},
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
				},
				filename: "4321-another-proposal.md",
			}},
			expectedEventType: corev1.EventTypeNormal,
			expectedReason:    "TrackingIssuesUpdatedOrCreated",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			ghClient, mux, closeFunc := testutil.SetupFakeGitHub()
			defer closeFunc()

			tgc := ghclient.NewTEPGHClient(ghClient)

			prRepo := tc.prRepo
			if prRepo == "" {
				prRepo = "community"
			}
			prNumber := tc.prNumber
			if prNumber == 0 {
				prNumber = 1
			}
			prAction := tc.prAction
			if prAction == "" {
				prAction = "opened"
			}

			expectedClosedIssues := make(map[int]bool)

			for _, closed := range tc.closedIssues {
				// Mark that we expect to see this issue closed by adding it to the map with a false value
				expectedClosedIssues[closed.number] = false

				mux.HandleFunc(fmt.Sprintf("/repos/tektoncd/community/issues/%d/comments", closed.number),
					func(w http.ResponseWriter, r *http.Request) {
						v := new(github.IssueComment)
						require.NoError(t, json.NewDecoder(r.Body).Decode(v))

						require.Equal(t, "POST", r.Method)

						expected := &github.IssueComment{
							Body: github.String(closed.comment),
						}

						if d := cmp.Diff(expected, v); d != "" {
							t.Errorf("difference in PATCH body: %s", d)
						}

						_, _ = fmt.Fprint(w, `{"number":1}`)
					})
			}

			mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/pulls/%d/files", ghclient.TEPsOwner, ghclient.TEPsRepo, prNumber),
				func(w http.ResponseWriter, r *http.Request) {
					respBody, err := json.Marshal(tc.listFilesResponse)
					if err != nil {
						t.Fatal("marshalling GitHub file list")
					}
					_, _ = fmt.Fprint(w, string(respBody))
				})

			for _, f := range tc.listFilesResponse {
				if strings.HasPrefix(f.GetFilename(), "teps/") {
					fn := strings.TrimPrefix(f.GetFilename(), "teps/")
					mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/contents/teps/%s", ghclient.TEPsOwner, ghclient.TEPsRepo, fn),
						func(w http.ResponseWriter, r *http.Request) {
							if !strings.HasSuffix(r.RequestURI, fmt.Sprintf("?ref=%s", contentsRef)) {
								t.Errorf("expected request for ref %s, but URI was %s", contentsRef, r.RequestURI)
							}

							fileContent, err := ioutil.ReadFile(filepath.Join("..", "ghclient", "testdata", fn))
							if err != nil {
								t.Fatalf("reading ../ghclient/testdata/%s", fn)
							}

							_, _ = fmt.Fprint(w, testutil.GHContentJSON(string(fileContent)))
						})
				}
			}

			createdIssueCalls := 0

			mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/issues", ghclient.TEPsOwner, ghclient.TEPsRepo),
				func(w http.ResponseWriter, r *http.Request) {
					if r.Method == "GET" {
						respBody, err := json.Marshal(tc.existingIssues)
						if err != nil {
							t.Fatal("marshalling GitHub issue list")
						}
						_, _ = fmt.Fprint(w, string(respBody))
					} else if r.Method == "POST" {
						v := new(github.IssueRequest)
						require.NoError(t, json.NewDecoder(r.Body).Decode(v))

						matchedIR := false

						for _, created := range tc.createdIssues {
							ir := created.toIssueRequest(t)

							if cmp.Equal(ir, v) {
								matchedIR = true
							}
						}

						if !matchedIR {
							unknownReq, _ := json.MarshalIndent(v, "", "  ")
							t.Fatalf("received unexpected IssueRequest:\n%s", string(unknownReq))
						}

						createdIssueCalls++
						_, _ = fmt.Fprint(w, `{"number":1}`)
					}
				})

			closedIssueCalls := 0
			modifiedIssueCalls := 0

			for _, modified := range tc.modifiedIssues {
				mux.HandleFunc(fmt.Sprintf("/repos/tektoncd/community/issues/%d", modified.IssueNumber),
					func(w http.ResponseWriter, r *http.Request) {
						v := new(github.IssueRequest)
						require.NoError(t, json.NewDecoder(r.Body).Decode(v))

						require.Equal(t, "PATCH", r.Method)

						// If this is a close, check against expectedClosedIssues
						if v.GetState() == "closed" {
							_, ok := expectedClosedIssues[modified.IssueNumber]
							assert.Truef(t, ok, "did not expect to receive close for issue %d", modified.IssueNumber)
							expectedClosedIssues[modified.IssueNumber] = true
							closedIssueCalls++
						} else {
							ir := modified.toIssueRequest(t)

							if d := cmp.Diff(ir, v); d != "" {
								t.Errorf("difference in PATCH body: %s", d)
							}

							modifiedIssueCalls++
						}

						_, _ = fmt.Fprint(w, `{"number":1}`)
					})
			}

			n := performers.NewIssueManager(tgc)

			opts := &performers.PerformerOptions{
				RunName:      "test-reconcile-run",
				RunNamespace: "foo",
				Action:       prAction,
				PRNumber:     prNumber,
				Title:        "some-title",
				Body:         "some-body",
				Repo:         prRepo,
				IsMerged:     tc.prIsMerged,
				GitRevision:  contentsRef,
			}

			err := n.Perform(ctx, opts)
			if tc.expectedErr != nil {
				assert.Equal(t, tc.expectedErr, err)
			} else {
				if tc.doesNothing {
					require.Nil(t, err)
				} else {
					require.NotNil(t, err)
					recEvt, ok := err.(*kreconciler.ReconcilerEvent)
					if !ok {
						t.Fatalf("did not expect non-ReconcilerEvent error %s", recEvt)
					} else {
						if recEvt.EventType != tc.expectedEventType {
							t.Errorf("Expected event type to be %s but was %s with message %s", tc.expectedEventType, recEvt.EventType, recEvt.Error())
						}
						if recEvt.Reason != tc.expectedReason {
							t.Errorf("Expected reason to be %q but was %q with message %s", tc.expectedReason, recEvt.Reason, recEvt.Error())
						}
					}
					assert.Equal(t, len(tc.createdIssues), createdIssueCalls, "wrong number of issue creation calls")
					assert.Equal(t, len(tc.modifiedIssues), modifiedIssueCalls, "wrong number of issue modification calls")
					assert.Equal(t, len(tc.closedIssues), closedIssueCalls, "wrong number of issue close calls")
				}
			}
		})
	}
}
