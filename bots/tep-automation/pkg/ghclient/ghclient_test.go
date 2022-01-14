package ghclient_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v41/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/ghclient"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/tep"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/testutil"
)

func TestGetTEPsFromReadme(t *testing.T) {
	testCases := []struct {
		name         string
		respContent  string
		expectedTEPs map[string]tep.TEPInfo
	}{
		{
			name:         "none",
			respContent:  testutil.GHContentJSON("nothing"),
			expectedTEPs: make(map[string]tep.TEPInfo),
		},
		{
			name: "one TEP",
			respContent: `there's one tep in here
on a later line
|[TEP-1234](1234-something-or-other.md) | Some TEP Title | proposed | 2021-12-20 |
|[TEP-5678](5678-not-valid-line.md) | | proposed | 2021-12-20 |
tada, a single valid TEP and a bogus line
`,
			expectedTEPs: map[string]tep.TEPInfo{
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
			name:        "three TEPs",
			respContent: testutil.DefaultTEPReadmeContent,
			expectedTEPs: map[string]tep.TEPInfo{
				"1234": {
					ID:           "1234",
					Title:        "Some TEP Title",
					Status:       tep.ProposedStatus,
					Filename:     "1234-something-or-other.md",
					LastModified: time.Date(2021, time.December, 20, 0, 0, 0, 0, time.UTC),
				},
				"5678": {
					ID:           "5678",
					Title:        "Another TEP Title",
					Status:       tep.ProposedStatus,
					Filename:     "5678-second-one.md",
					LastModified: time.Date(2021, time.December, 20, 0, 0, 0, 0, time.UTC),
				},
				"4321": {
					ID:           "4321",
					Title:        "Yet Another TEP Title",
					Status:       tep.ImplementingStatus,
					Filename:     "4321-third-one.md",
					LastModified: time.Date(2021, time.December, 20, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mux, closeFunc := testutil.SetupFakeGitHub()
			defer closeFunc()

			mux.HandleFunc(testutil.ReadmeURL,
				func(w http.ResponseWriter, r *http.Request) {
					if !strings.HasSuffix(r.RequestURI, fmt.Sprintf("?ref=%s", ghclient.TEPsBranch)) {
						t.Errorf("expected request for branch %s, but URI was %s", ghclient.TEPsBranch, r.RequestURI)
					}
					_, _ = fmt.Fprint(w, testutil.GHContentJSON(tc.respContent))
				})

			tgc := ghclient.NewTEPGHClient(client)

			ctx := context.Background()

			teps, err := tgc.GetTEPsFromReadme(ctx)
			require.NoError(t, err)

			if d := cmp.Diff(tc.expectedTEPs, teps); d != "" {
				t.Errorf("Wrong TEPs from README.md: (-want, +got): %s", d)
			}
		})
	}
}

type testComment struct {
	id   int64
	user string
	body string
}

func TestTEPCommentsOnPR(t *testing.T) {
	testCases := []struct {
		name     string
		comments []testComment
		expected []tep.CommentInfo
	}{
		{
			name: "none with bot user comment",
			comments: []testComment{{
				id:   1,
				user: ghclient.BotUser,
				body: "There are no TEPs here",
			}},
		},
		{
			name: "one with bot user comment",
			comments: []testComment{{
				id:   1,
				user: ghclient.BotUser,
				body: `this comment has some text
and it also has a TEP

<!-- TEP Notifier Action: implementing -->

<!-- TEP update: TEP-1234 status: proposed -->
`,
			}},
			expected: []tep.CommentInfo{{
				CommentID:     1,
				TEPs:          []string{"1234"},
				ToImplemented: false,
			}},
		},
		{
			name: "both implementing and implemented comments",
			comments: []testComment{
				{
					id:   1,
					user: ghclient.BotUser,
					body: `this comment has some text
and it also has a TEP

<!-- TEP Notifier Action: implementing -->

<!-- TEP update: TEP-1234 status: proposed -->
`,
				},
				{
					id:   2,
					user: ghclient.BotUser,
					body: `close this TEP

<!-- TEP Notifier Action: implemented -->

<!-- TEP update: TEP-1234 status: implementing -->
`,
				},
			},
			expected: []tep.CommentInfo{
				{
					CommentID:     1,
					TEPs:          []string{"1234"},
					ToImplemented: false,
				},
				{
					CommentID:     2,
					TEPs:          []string{"1234"},
					ToImplemented: true,
				},
			},
		},
		{
			name: "one with other user comment",
			comments: []testComment{{
				id:   1,
				user: "abayer",
				body: `this comment has some text
and it also has a TEP

<!-- TEP update: TEP-1234 status: implementing -->
`,
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mux, closeFunc := testutil.SetupFakeGitHub()
			defer closeFunc()

			mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/issues/1/comments", ghclient.TEPsOwner, ghclient.TEPsRepo),
				func(w http.ResponseWriter, r *http.Request) {
					var ghComments []*github.IssueComment

					for _, c := range tc.comments {
						cCopy := c
						ghComments = append(ghComments, &github.IssueComment{
							ID: &cCopy.id,
							User: &github.User{
								Login: &cCopy.user,
							},
							Body: &cCopy.body,
						})
					}

					respBody, err := json.Marshal(ghComments)
					if err != nil {
						t.Fatal("marshalling GitHub comments")
					}
					_, _ = fmt.Fprint(w, string(respBody))
				})

			tgc := ghclient.NewTEPGHClient(client)

			ctx := context.Background()

			tepComments, err := tgc.TEPCommentsOnPR(ctx, ghclient.TEPsRepo, 1)
			require.NoError(t, err)

			assert.ElementsMatch(t, tc.expected, tepComments)
		})
	}
}

func TestAddComment(t *testing.T) {
	client, mux, closeFunc := testutil.SetupFakeGitHub()
	defer closeFunc()

	input := &github.IssueComment{
		Body: github.String("some body"),
	}

	mux.HandleFunc("/repos/tektoncd/pipeline/issues/1/comments",
		func(w http.ResponseWriter, r *http.Request) {
			v := new(github.IssueComment)
			require.NoError(t, json.NewDecoder(r.Body).Decode(v))

			require.Equal(t, "POST", r.Method)

			if d := cmp.Diff(input, v); d != "" {
				t.Errorf("difference in POST body: %s", d)
			}
			_, _ = fmt.Fprint(w, `{"id":1}`)
		})

	tgc := ghclient.NewTEPGHClient(client)

	ctx := context.Background()

	require.NoError(t, tgc.AddComment(ctx, "pipeline", 1, "some body"))
}

func TestEditComment(t *testing.T) {
	client, mux, closeFunc := testutil.SetupFakeGitHub()
	defer closeFunc()

	input := &github.IssueComment{
		Body: github.String("some new body"),
	}

	mux.HandleFunc("/repos/tektoncd/pipeline/issues/comments/1",
		func(w http.ResponseWriter, r *http.Request) {
			v := new(github.IssueComment)
			require.NoError(t, json.NewDecoder(r.Body).Decode(v))

			require.Equal(t, "PATCH", r.Method)

			if d := cmp.Diff(input, v); d != "" {
				t.Errorf("difference in PATCH body: %s", d)
			}
			_, _ = fmt.Fprint(w, `{"id":1}`)
		})

	tgc := ghclient.NewTEPGHClient(client)

	ctx := context.Background()

	require.NoError(t, tgc.EditComment(ctx, "pipeline", 1, "some new body"))
}

func TestCreateTrackingIssue(t *testing.T) {
	client, mux, closeFunc := testutil.SetupFakeGitHub()
	defer closeFunc()

	expectedTitle := "TEP-12345 Tracking Issue"
	expectedBody := "some body"
	expectedLabels := []string{
		ghclient.TrackingIssueLabel,
		tep.NewStatus.TrackingLabel(),
	}
	expectedAssignees := []string{
		"abayer",
		"vdemeester",
	}

	input := &github.IssueRequest{
		Title:     &expectedTitle,
		Body:      &expectedBody,
		Labels:    &expectedLabels,
		Assignees: &expectedAssignees,
	}

	mux.HandleFunc("/repos/tektoncd/community/issues",
		func(w http.ResponseWriter, r *http.Request) {
			v := new(github.IssueRequest)
			require.NoError(t, json.NewDecoder(r.Body).Decode(v))

			require.Equal(t, "POST", r.Method)

			if d := cmp.Diff(input, v); d != "" {
				t.Errorf("difference in POST body: %s", d)
			}
			_, _ = fmt.Fprint(w, `{"id":1}`)
		})

	tgc := ghclient.NewTEPGHClient(client)

	ctx := context.Background()

	require.NoError(t, tgc.CreateTrackingIssue(ctx, "12345", expectedBody, []string{"abayer", "vdemeester"}, tep.NewStatus))
}

func TestUpdateTrackingIssue(t *testing.T) {
	client, mux, closeFunc := testutil.SetupFakeGitHub()
	defer closeFunc()

	expectedTitle := "TEP-12345 Tracking Issue"
	expectedBody := "some body"
	expectedLabels := []string{
		ghclient.TrackingIssueLabel,
		tep.NewStatus.TrackingLabel(),
	}
	expectedAssignees := []string{
		"abayer",
		"vdemeester",
	}

	input := &github.IssueRequest{
		Title:     &expectedTitle,
		Body:      &expectedBody,
		Labels:    &expectedLabels,
		Assignees: &expectedAssignees,
	}

	issueNumber := 5

	mux.HandleFunc(fmt.Sprintf("/repos/tektoncd/community/issues/%d", issueNumber),
		func(w http.ResponseWriter, r *http.Request) {
			v := new(github.IssueRequest)
			require.NoError(t, json.NewDecoder(r.Body).Decode(v))

			require.Equal(t, "PATCH", r.Method)

			if d := cmp.Diff(input, v); d != "" {
				t.Errorf("difference in PATCH body: %s", d)
			}
			_, _ = fmt.Fprint(w, `{"number":5}`)
		})

	tgc := ghclient.NewTEPGHClient(client)

	ctx := context.Background()

	require.NoError(t, tgc.UpdateTrackingIssue(ctx, issueNumber, "12345", expectedBody, []string{"abayer", "vdemeester"}, tep.NewStatus))
}

func TestCloseTrackingIssue(t *testing.T) {
	client, mux, closeFunc := testutil.SetupFakeGitHub()
	defer closeFunc()

	commentInput := &github.IssueComment{
		Body: github.String("some body"),
	}

	issueInput := &github.IssueRequest{
		State: github.String("closed"),
	}

	issueNumber := 5

	mux.HandleFunc("/repos/tektoncd/community/issues/5/comments",
		func(w http.ResponseWriter, r *http.Request) {
			v := new(github.IssueComment)
			require.NoError(t, json.NewDecoder(r.Body).Decode(v))

			require.Equal(t, "POST", r.Method)

			if d := cmp.Diff(commentInput, v); d != "" {
				t.Errorf("difference in POST body: %s", d)
			}
			_, _ = fmt.Fprint(w, `{"id":1}`)
		})

	mux.HandleFunc(fmt.Sprintf("/repos/tektoncd/community/issues/%d", issueNumber),
		func(w http.ResponseWriter, r *http.Request) {
			v := new(github.IssueRequest)
			require.NoError(t, json.NewDecoder(r.Body).Decode(v))

			require.Equal(t, "PATCH", r.Method)

			if d := cmp.Diff(issueInput, v); d != "" {
				t.Errorf("difference in PATCH body: %s", d)
			}
			_, _ = fmt.Fprint(w, `{"number":5}`)
		})

	tgc := ghclient.NewTEPGHClient(client)

	ctx := context.Background()

	require.NoError(t, tgc.CloseTrackingIssue(ctx, issueNumber, "some body"))
}

func TestExtractTEPInfoFromTEPPR(t *testing.T) {
	prNumber := 1
	contentsRef := "some-ref"

	testCases := []struct {
		name              string
		listFilesResponse []*github.CommitFile
		teps              []tep.TEPInfo
		expectedErr       error
	}{
		{
			name:              "no files",
			listFilesResponse: []*github.CommitFile{},
		},
		{
			name: "new TEP file",
			listFilesResponse: []*github.CommitFile{{
				SHA:      github.String("some-sha"),
				Filename: github.String("teps/1234-some-proposal.md"),
				Status:   github.String("added"),
			}},
			teps: []tep.TEPInfo{{
				ID:           "1234",
				Title:        "Some New Feature",
				Status:       tep.ProposedStatus,
				Filename:     "1234-some-proposal.md",
				LastModified: time.Date(2022, time.January, 6, 0, 0, 0, 0, time.UTC),
				Authors: []string{
					"abayer",
					"vdemeester",
				},
			}},
		},
		{
			name: "modified TEP file",
			listFilesResponse: []*github.CommitFile{{
				SHA:      github.String("some-sha"),
				Filename: github.String("teps/1234-some-proposal.md"),
				Status:   github.String("modified"),
			}},
			teps: []tep.TEPInfo{{
				ID:           "1234",
				Title:        "Some New Feature",
				Status:       tep.ProposedStatus,
				Filename:     "1234-some-proposal.md",
				LastModified: time.Date(2022, time.January, 6, 0, 0, 0, 0, time.UTC),
				Authors: []string{
					"abayer",
					"vdemeester",
				},
			}},
		},
		{
			name: "multiple TEP files",
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
			teps: []tep.TEPInfo{{
				ID:           "5678",
				Title:        "Just Show Cats",
				Status:       tep.ImplementingStatus,
				Filename:     "5678-just-cats.md",
				LastModified: time.Date(2021, time.January, 6, 0, 0, 0, 0, time.UTC),
				Authors: []string{
					"abayer",
					"bobcatfish",
				},
			}, {
				ID:           "1234",
				Title:        "Some New Feature",
				Status:       tep.ProposedStatus,
				Filename:     "1234-some-proposal.md",
				LastModified: time.Date(2022, time.January, 6, 0, 0, 0, 0, time.UTC),
				Authors: []string{
					"abayer",
					"vdemeester",
				},
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mux, closeFunc := testutil.SetupFakeGitHub()
			defer closeFunc()

			mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/pulls/%d/files", ghclient.TEPsOwner, ghclient.TEPsRepo, prNumber),
				func(w http.ResponseWriter, r *http.Request) {
					respBody, err := json.Marshal(tc.listFilesResponse)
					if err != nil {
						t.Fatal("marshalling GitHub file list")
					}
					_, _ = fmt.Fprint(w, string(respBody))
				})

			for _, f := range tc.listFilesResponse {
				fn := strings.TrimPrefix(*f.Filename, "teps/")
				mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/contents/teps/%s", ghclient.TEPsOwner, ghclient.TEPsRepo, fn),
					func(w http.ResponseWriter, r *http.Request) {
						if !strings.HasSuffix(r.RequestURI, fmt.Sprintf("?ref=%s", contentsRef)) {
							t.Errorf("expected request for ref %s, but URI was %s", contentsRef, r.RequestURI)
						}

						fileContent, err := ioutil.ReadFile(filepath.Join("testdata", fn))
						if err != nil {
							t.Fatalf("reading testdata/%s", fn)
						}

						_, _ = fmt.Fprint(w, testutil.GHContentJSON(string(fileContent)))
					})
			}
			tgc := ghclient.NewTEPGHClient(client)

			ctx := context.Background()

			foundTEPs, err := tgc.ExtractTEPInfoFromTEPPR(ctx, prNumber, contentsRef)

			if tc.expectedErr != nil {
				require.EqualError(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)

				assert.ElementsMatch(t, tc.teps, foundTEPs, "wrong TEPs")
			}
		})
	}
}

type testIssue struct {
	number    int
	title     string
	body      string
	state     string
	labels    []string
	assignees []string
}

func TestGetTrackingIssues(t *testing.T) {
	testCases := []struct {
		name           string
		allIssues      []testIssue
		listOpts       *ghclient.GetTrackingIssuesOptions
		expectedIssues map[string]*tep.TrackingIssue
		expectedErr    error
	}{
		{
			name:           "no issues",
			expectedIssues: map[string]*tep.TrackingIssue{},
		},
		{
			name: "no tracking issues",
			allIssues: []testIssue{{
				number:    5,
				title:     "some other issue",
				body:      "no TEP stuff in here",
				state:     "open",
				labels:    []string{"some-other-label"},
				assignees: []string{"bob"},
			}},
			expectedIssues: map[string]*tep.TrackingIssue{},
		},
		{
			name: "one tracking issue without PRs",
			allIssues: []testIssue{{
				number: 7,
				title:  "TEP-1234 Tracking Issue",
				body:   "Nothing in here",
				state:  "open",
				labels: []string{
					ghclient.TrackingIssueLabel,
					tep.ProposedStatus.TrackingLabel(),
				},
				assignees: []string{"bob", "steve"},
			}},
			expectedIssues: map[string]*tep.TrackingIssue{
				"1234": {
					IssueNumber: 7,
					IssueState:  "open",
					TEPStatus:   tep.ProposedStatus,
					TEPID:       "1234",
					Assignees:   []string{"bob", "steve"},
				},
			},
		},
		{
			name: "one tracking issue with PRs",
			allIssues: []testIssue{{
				number: 7,
				title:  "TEP-1234 Tracking Issue",
				body: `Something in here this time
<!-- TEP PR: 55 -->
<!-- TEP PR: 66 -->
<!-- Implementation PR: repo: pipeline number: 77 -->
<!-- Implementation PR: repo: triggers number: 88 -->`,
				state: "open",
				labels: []string{
					ghclient.TrackingIssueLabel,
					tep.ProposedStatus.TrackingLabel(),
				},
				assignees: []string{"bob", "steve"},
			}},
			expectedIssues: map[string]*tep.TrackingIssue{
				"1234": {
					IssueNumber: 7,
					IssueState:  "open",
					TEPStatus:   tep.ProposedStatus,
					TEPID:       "1234",
					TEPPRs:      []int{55, 66},
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
					Assignees: []string{"bob", "steve"},
				},
			},
		},
		{
			name: "filter by issue state",
			allIssues: []testIssue{
				{
					number: 6,
					title:  "TEP-0001 Tracking Issue",
					body:   "Nothing in here",
					state:  "closed",
					labels: []string{
						ghclient.TrackingIssueLabel,
						tep.ImplementedStatus.TrackingLabel(),
					},
					assignees: []string{"foo", "bar"},
				},
				{
					number: 7,
					title:  "TEP-1234 Tracking Issue",
					body:   "Nothing in here",
					state:  "open",
					labels: []string{
						ghclient.TrackingIssueLabel,
						tep.ProposedStatus.TrackingLabel(),
					},
					assignees: []string{"bob", "steve"},
				},
				{
					number: 9,
					title:  "TEP-0002 Tracking Issue",
					body:   "Nothing in here",
					state:  "closed",
					labels: []string{
						ghclient.TrackingIssueLabel,
						tep.WithdrawnStatus.TrackingLabel(),
					},
					assignees: []string{"a", "b"},
				},
			},
			listOpts: &ghclient.GetTrackingIssuesOptions{
				IssueState: "open",
			},
			expectedIssues: map[string]*tep.TrackingIssue{
				"1234": {
					IssueNumber: 7,
					IssueState:  "open",
					TEPStatus:   tep.ProposedStatus,
					TEPID:       "1234",
					Assignees:   []string{"bob", "steve"},
				},
			},
		},
		{
			name: "filter by TEP status",
			allIssues: []testIssue{
				{
					number: 6,
					title:  "TEP-0001 Tracking Issue",
					body:   "Nothing in here",
					state:  "closed",
					labels: []string{
						ghclient.TrackingIssueLabel,
						tep.ImplementedStatus.TrackingLabel(),
					},
					assignees: []string{"foo", "bar"},
				},
				{
					number: 7,
					title:  "TEP-1234 Tracking Issue",
					body:   "Nothing in here",
					state:  "open",
					labels: []string{
						ghclient.TrackingIssueLabel,
						tep.ProposedStatus.TrackingLabel(),
					},
					assignees: []string{"bob", "steve"},
				},
				{
					number: 9,
					title:  "TEP-0002 Tracking Issue",
					body:   "Nothing in here",
					state:  "closed",
					labels: []string{
						ghclient.TrackingIssueLabel,
						tep.WithdrawnStatus.TrackingLabel(),
					},
					assignees: []string{"a", "b"},
				},
			},
			listOpts: &ghclient.GetTrackingIssuesOptions{
				TEPStatus: tep.ProposedStatus,
			},
			expectedIssues: map[string]*tep.TrackingIssue{
				"1234": {
					IssueNumber: 7,
					IssueState:  "open",
					TEPStatus:   tep.ProposedStatus,
					TEPID:       "1234",
					Assignees:   []string{"bob", "steve"},
				},
			},
		},
		{
			name: "filter by TEP ID",
			allIssues: []testIssue{
				{
					number: 6,
					title:  "TEP-0001 Tracking Issue",
					body:   "Nothing in here",
					state:  "closed",
					labels: []string{
						ghclient.TrackingIssueLabel,
						tep.ImplementedStatus.TrackingLabel(),
					},
					assignees: []string{"foo", "bar"},
				},
				{
					number: 7,
					title:  "TEP-1234 Tracking Issue",
					body:   "Nothing in here",
					state:  "open",
					labels: []string{
						ghclient.TrackingIssueLabel,
						tep.ProposedStatus.TrackingLabel(),
					},
					assignees: []string{"bob", "steve"},
				},
				{
					number: 9,
					title:  "TEP-0002 Tracking Issue",
					body:   "Nothing in here",
					state:  "closed",
					labels: []string{
						ghclient.TrackingIssueLabel,
						tep.WithdrawnStatus.TrackingLabel(),
					},
					assignees: []string{"a", "b"},
				},
			},
			listOpts: &ghclient.GetTrackingIssuesOptions{
				TEPID: "1234",
			},
			expectedIssues: map[string]*tep.TrackingIssue{
				"1234": {
					IssueNumber: 7,
					IssueState:  "open",
					TEPStatus:   tep.ProposedStatus,
					TEPID:       "1234",
					Assignees:   []string{"bob", "steve"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mux, closeFunc := testutil.SetupFakeGitHub()
			defer closeFunc()

			mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/issues", ghclient.TEPsOwner, ghclient.TEPsRepo),
				func(w http.ResponseWriter, r *http.Request) {
					var issues []*github.Issue

					if err := r.ParseForm(); err != nil {
						t.Fatalf("couldn't parse form: %s", err)
					}

					filterState := ""
					filterTEPStatus := ""

					if tc.listOpts != nil {
						if tc.listOpts.IssueState != "" {
							filterState = tc.listOpts.IssueState
							assert.Equal(t, tc.listOpts.IssueState, r.Form.Get("state"))
						}
						if tc.listOpts.TEPStatus != "" {
							filterTEPStatus = tc.listOpts.TEPStatus.TrackingLabel()
							assert.Contains(t, r.Form.Get("labels"), tc.listOpts.TEPStatus.TrackingLabel())
						}
					}
					assert.Contains(t, r.Form.Get("labels"), ghclient.TrackingIssueLabel)

					for _, i := range tc.allIssues {
						if filterState != "" && i.state != filterState {
							continue
						}
						if filterTEPStatus != "" {
							foundLabel := false
							for _, l := range i.labels {
								if l == filterTEPStatus {
									foundLabel = true
									break
								}
							}
							if !foundLabel {
								continue
							}
						}

						hasTrackingLabel := false
						for _, l := range i.labels {
							if l == ghclient.TrackingIssueLabel {
								hasTrackingLabel = true
								break
							}
						}
						if !hasTrackingLabel {
							continue
						}

						ghIssue := github.Issue{
							Number: github.Int(i.number),
							State:  github.String(i.state),
							Title:  github.String(i.title),
							Body:   github.String(i.body),
							Labels: []*github.Label{{
								Name: github.String(ghclient.TrackingIssueLabel),
							}},
							Assignees: []*github.User{},
						}

						for _, l := range i.labels {
							ghIssue.Labels = append(ghIssue.Labels, &github.Label{Name: github.String(l)})
						}

						for _, a := range i.assignees {
							ghIssue.Assignees = append(ghIssue.Assignees, &github.User{Login: github.String(a)})
						}

						issues = append(issues, &ghIssue)
					}

					respBody, err := json.Marshal(issues)
					if err != nil {
						t.Fatal("marshalling GitHub issues")
					}
					_, _ = fmt.Fprint(w, string(respBody))
				})

			tgc := ghclient.NewTEPGHClient(client)

			ctx := context.Background()

			trackingIssues, err := tgc.GetTrackingIssues(ctx, tc.listOpts)

			if tc.expectedErr != nil {
				require.EqualError(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)

				assert.Equal(t, tc.expectedIssues, trackingIssues, "wrong tracking issues")
			}
		})
	}
}
