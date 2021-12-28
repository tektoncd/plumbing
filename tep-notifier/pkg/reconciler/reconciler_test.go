package reconciler_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v41/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/plumbing/tep-notifier/pkg/reconciler"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

const (
	defaultTEPReadmeContent = `there are three teps in here
on later lines
|[TEP-1234](1234-something-or-other.md) | Some TEP Title | proposed | 2021-12-20 |
|[TEP-5678](5678-second-one.md) | Another TEP Title | proposed | 2021-12-20 |
|[TEP-4321](4321-third-one.md) | Yet Another TEP Title | implementing | 2021-12-20 |
tada, three valid TEPs
`
)

var (
	readmeURL = fmt.Sprintf("/repos/%s/%s/contents/%s/%s", reconciler.TEPsOwner, reconciler.TEPsRepo,
		reconciler.TEPsDirectory, reconciler.TEPsReadmeFile)
	defaultRunParams = map[string]string{
		reconciler.ActionParamName:     "opened",
		reconciler.PRNumberParamName:   "1",
		reconciler.PRTitleParamName:    "Some PR",
		reconciler.PRBodyParamName:     "A PR body, without any TEPs in it",
		reconciler.PackageParamName:    "tektoncd/pipeline",
		reconciler.PRIsMergedParamName: "false",
	}
)

func TestExtractTEPsFromReadme(t *testing.T) {
	testCases := []struct {
		name     string
		body     string
		expected map[string]reconciler.TEPInfo
		errStr   string
	}{
		{
			name:     "no TEPs",
			body:     "there are no teps here",
			expected: make(map[string]reconciler.TEPInfo),
		},
		{
			name: "single TEP",
			body: `there's one tep in here
on a later line
|[TEP-1234](1234-something-or-other.md) | Some TEP Title | proposed | 2021-12-20 |
|[TEP-5678](5678-not-valid-line.md) | | proposed | 2021-12-20 |
tada, a single valid TEP and a bogus line
`,
			expected: map[string]reconciler.TEPInfo{
				"1234": {
					ID:           "1234",
					Title:        "Some TEP Title",
					Status:       reconciler.ProposedStatus,
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
			expected: map[string]reconciler.TEPInfo{
				"1234": {
					ID:           "1234",
					Title:        "Some TEP Title",
					Status:       reconciler.ProposedStatus,
					Filename:     "1234-something-or-other.md",
					LastModified: time.Date(2021, time.December, 20, 0, 0, 0, 0, time.UTC),
				},
				"5678": {
					ID:           "5678",
					Title:        "A Second TEP Title",
					Status:       reconciler.ImplementedStatus,
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
			teps, err := reconciler.ExtractTEPsFromReadme(tc.body)
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
			found := reconciler.GetTEPIDsFromPR(tc.title, tc.body)
			assert.ElementsMatch(t, tc.ids, found)
		})
	}
}

func TestGetTEPsWithStatus(t *testing.T) {
	testCases := []struct {
		name        string
		status      reconciler.TEPStatus
		input       map[string]reconciler.TEPInfo
		expectedIDs []string
	}{
		{
			name:   "none",
			status: reconciler.ProposedStatus,
			input: map[string]reconciler.TEPInfo{
				"1234": {
					ID:           "1234",
					Title:        "Some TEP",
					Status:       reconciler.ImplementableStatus,
					Filename:     "1234.md",
					LastModified: time.Time{},
				},
			},
		},
		{
			name:   "one match",
			status: reconciler.ProposedStatus,
			input: map[string]reconciler.TEPInfo{
				"1234": {
					ID:           "1234",
					Title:        "Some TEP",
					Status:       reconciler.ImplementableStatus,
					Filename:     "1234.md",
					LastModified: time.Time{},
				},
				"4321": {
					ID:           "4321",
					Title:        "Some Other TEP",
					Status:       reconciler.ProposedStatus,
					Filename:     "4321.md",
					LastModified: time.Time{},
				},
			},
			expectedIDs: []string{"4321"},
		},
		{
			name:   "two match",
			status: reconciler.ProposedStatus,
			input: map[string]reconciler.TEPInfo{
				"1234": {
					ID:           "1234",
					Title:        "Some TEP",
					Status:       reconciler.ImplementableStatus,
					Filename:     "1234.md",
					LastModified: time.Time{},
				},
				"4321": {
					ID:           "4321",
					Title:        "Some Other TEP",
					Status:       reconciler.ProposedStatus,
					Filename:     "4321.md",
					LastModified: time.Time{},
				},
				"5678": {
					ID:           "5678",
					Title:        "A Third TEP",
					Status:       reconciler.ImplementedStatus,
					Filename:     "5678.md",
					LastModified: time.Time{},
				},
				"8765": {
					ID:           "8765",
					Title:        "A Fourth TEP",
					Status:       reconciler.ProposedStatus,
					Filename:     "8765.md",
					LastModified: time.Time{},
				},
			},
			expectedIDs: []string{"4321", "8765"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			found := reconciler.GetTEPsWithStatus(tc.input, tc.status)
			var foundIDs []string
			for k := range found {
				foundIDs = append(foundIDs, k)
			}
			assert.ElementsMatch(t, tc.expectedIDs, foundIDs)
		})
	}
}

func TestGetTEPsFromReadme(t *testing.T) {
	testCases := []struct {
		name         string
		respContent  string
		expectedTEPs map[string]reconciler.TEPInfo
	}{
		{
			name:         "none",
			respContent:  ghContentJSON("nothing"),
			expectedTEPs: make(map[string]reconciler.TEPInfo),
		},
		{
			name: "one TEP",
			respContent: `there's one tep in here
on a later line
|[TEP-1234](1234-something-or-other.md) | Some TEP Title | proposed | 2021-12-20 |
|[TEP-5678](5678-not-valid-line.md) | | proposed | 2021-12-20 |
tada, a single valid TEP and a bogus line
`,
			expectedTEPs: map[string]reconciler.TEPInfo{
				"1234": {
					ID:           "1234",
					Title:        "Some TEP Title",
					Status:       reconciler.ProposedStatus,
					Filename:     "1234-something-or-other.md",
					LastModified: time.Date(2021, time.December, 20, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name:        "three TEPs",
			respContent: defaultTEPReadmeContent,
			expectedTEPs: map[string]reconciler.TEPInfo{
				"1234": {
					ID:           "1234",
					Title:        "Some TEP Title",
					Status:       reconciler.ProposedStatus,
					Filename:     "1234-something-or-other.md",
					LastModified: time.Date(2021, time.December, 20, 0, 0, 0, 0, time.UTC),
				},
				"5678": {
					ID:           "5678",
					Title:        "Another TEP Title",
					Status:       reconciler.ProposedStatus,
					Filename:     "5678-second-one.md",
					LastModified: time.Date(2021, time.December, 20, 0, 0, 0, 0, time.UTC),
				},
				"4321": {
					ID:           "4321",
					Title:        "Yet Another TEP Title",
					Status:       reconciler.ImplementingStatus,
					Filename:     "4321-third-one.md",
					LastModified: time.Date(2021, time.December, 20, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, mux, closeFunc := setupFakeGitHub()
			defer closeFunc()

			mux.HandleFunc(readmeURL,
				func(w http.ResponseWriter, r *http.Request) {
					if !strings.HasSuffix(r.RequestURI, fmt.Sprintf("?ref=%s", reconciler.TEPsBranch)) {
						t.Errorf("expected request for branch %s, but URI was %s", reconciler.TEPsBranch, r.RequestURI)
					}
					_, _ = fmt.Fprint(w, ghContentJSON(tc.respContent))
				})

			r := &reconciler.Reconciler{GHClient: client}

			ctx := context.Background()

			teps, err := r.GetTEPsFromReadme(ctx)
			require.NoError(t, err)

			if d := cmp.Diff(tc.expectedTEPs, teps); d != "" {
				t.Errorf("Wrong TEPs from README.md: (-want, +got): %s", d)
			}
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
				"1234": string(reconciler.ProposedStatus),
				"5678": string(reconciler.ImplementableStatus),
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
				"1234": string(reconciler.ImplementableStatus),
			},
			toImplemented: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			teps, toImplemented := reconciler.GetTEPCommentDetails(tc.comment)
			if d := cmp.Diff(tc.teps, teps); d != "" {
				t.Errorf("Wrong TEPs from comment: (-want, +got): %s", d)
			}
			assert.Equalf(t, tc.toImplemented, toImplemented, "expected toImplemented to be %t, but is %t", tc.toImplemented, toImplemented)
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
		expected []reconciler.TEPCommentInfo
	}{
		{
			name: "none with bot user comment",
			comments: []testComment{{
				id:   1,
				user: reconciler.BotUser,
				body: "There are no TEPs here",
			}},
		},
		{
			name: "one with bot user comment",
			comments: []testComment{{
				id:   1,
				user: reconciler.BotUser,
				body: `this comment has some text
and it also has a TEP

<!-- TEP Notifier Action: implementing -->

<!-- TEP update: TEP-1234 status: proposed -->
`,
			}},
			expected: []reconciler.TEPCommentInfo{{
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
					user: reconciler.BotUser,
					body: `this comment has some text
and it also has a TEP

<!-- TEP Notifier Action: implementing -->

<!-- TEP update: TEP-1234 status: proposed -->
`,
				},
				{
					id:   2,
					user: reconciler.BotUser,
					body: `close this TEP

<!-- TEP Notifier Action: implemented -->

<!-- TEP update: TEP-1234 status: implementing -->
`,
				},
			},
			expected: []reconciler.TEPCommentInfo{
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
			client, mux, closeFunc := setupFakeGitHub()
			defer closeFunc()

			mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/issues/1/comments", reconciler.TEPsOwner, reconciler.TEPsRepo),
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

			r := &reconciler.Reconciler{GHClient: client}

			ctx := context.Background()

			tepComments, err := r.TEPCommentsOnPR(ctx, reconciler.TEPsRepo, 1)
			require.NoError(t, err)

			assert.ElementsMatch(t, tc.expected, tepComments)
		})
	}
}

func TestPROpenedComment(t *testing.T) {
	teps := []reconciler.TEPInfo{
		{
			ID:           "1234",
			Title:        "Some TEP Title",
			Status:       reconciler.ProposedStatus,
			Filename:     "1234-something-or-other.md",
			LastModified: time.Date(2021, time.December, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:           "5678",
			Title:        "Some Other TEP Title",
			Status:       reconciler.ImplementableStatus,
			Filename:     "5678-insert-filename-here.md",
			LastModified: time.Date(2021, time.December, 21, 0, 0, 0, 0, time.UTC),
		},
	}

	expectedComment := reconciler.ToImplementingCommentHeader +
		" * [TEP-1234 (Some TEP Title)](https://github.com/tektoncd/community/blob/main/teps/1234-something-or-other.md), current status: `proposed`\n" +
		" * [TEP-5678 (Some Other TEP Title)](https://github.com/tektoncd/community/blob/main/teps/5678-insert-filename-here.md), current status: `implementable`\n" +
		"\n" +
		"<!-- TEP update: TEP-1234 status: proposed -->\n" +
		"<!-- TEP update: TEP-5678 status: implementable -->\n"

	assert.Equal(t, expectedComment, reconciler.PROpenedComment(teps))
}

func TestPRClosedComment(t *testing.T) {
	teps := []reconciler.TEPInfo{
		{
			ID:           "1234",
			Title:        "Some TEP Title",
			Status:       reconciler.ImplementingStatus,
			Filename:     "1234-something-or-other.md",
			LastModified: time.Date(2021, time.December, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:           "5678",
			Title:        "Some Other TEP Title",
			Status:       reconciler.ImplementingStatus,
			Filename:     "5678-insert-filename-here.md",
			LastModified: time.Date(2021, time.December, 21, 0, 0, 0, 0, time.UTC),
		},
	}

	expectedComment := reconciler.ToImplementedCommentHeader +
		" * [TEP-1234 (Some TEP Title)](https://github.com/tektoncd/community/blob/main/teps/1234-something-or-other.md), current status: `implementing`\n" +
		" * [TEP-5678 (Some Other TEP Title)](https://github.com/tektoncd/community/blob/main/teps/5678-insert-filename-here.md), current status: `implementing`\n" +
		"\n" +
		"<!-- TEP update: TEP-1234 status: implementing -->\n" +
		"<!-- TEP update: TEP-5678 status: implementing -->\n"

	assert.Equal(t, expectedComment, reconciler.PRMergedComment(teps))
}

func TestAddComment(t *testing.T) {
	client, mux, closeFunc := setupFakeGitHub()
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

	r := &reconciler.Reconciler{GHClient: client}

	ctx := context.Background()

	require.NoError(t, r.AddComment(ctx, "pipeline", 1, "some body"))
}

func TestEditComment(t *testing.T) {
	client, mux, closeFunc := setupFakeGitHub()
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

	r := &reconciler.Reconciler{GHClient: client}

	ctx := context.Background()

	require.NoError(t, r.EditComment(ctx, "pipeline", 1, "some new body"))
}

func TestReconcileKind(t *testing.T) {
	defaultKindRef := &v1beta1.TaskRef{
		APIVersion: v1beta1.SchemeGroupVersion.String(),
		Kind:       reconciler.Kind,
	}

	testCases := []struct {
		name             string
		paramOverrides   map[string]string
		additionalParams map[string]string
		kindRef          *v1beta1.TaskRef
		requests         map[string]func(w http.ResponseWriter, r *http.Request)
		doesNothing      bool
		expectedStatus   corev1.ConditionStatus
		expectedReason   string
		expectedErr      error
	}{
		{
			name:        "no TEPs",
			doesNothing: true,
		},
		{
			name: "invalid ref",
			kindRef: &v1beta1.TaskRef{
				APIVersion: v1beta1.SchemeGroupVersion.String(),
				Kind:       "SomethingElse",
			},
			doesNothing: true,
		},
		{
			name: "wrong action",
			paramOverrides: map[string]string{
				reconciler.ActionParamName: "assigned",
			},
			doesNothing: true,
		},
		{
			name: "closed but unmerged",
			paramOverrides: map[string]string{
				reconciler.ActionParamName: "closed",
			},
			doesNothing: true,
		},
		{
			name: "missing action",
			paramOverrides: map[string]string{
				reconciler.ActionParamName: "",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "MissingPullRequestAction",
		},
		{
			name: "missing PR number",
			paramOverrides: map[string]string{
				reconciler.PRNumberParamName: "",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "MissingPullRequestNumber",
		},
		{
			name: "missing PR title",
			paramOverrides: map[string]string{
				reconciler.PRTitleParamName: "",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "MissingPullRequestTitle",
		},
		{
			name: "missing PR body",
			paramOverrides: map[string]string{
				reconciler.PRBodyParamName: "",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "MissingPullRequestBody",
		},
		{
			name: "missing package",
			paramOverrides: map[string]string{
				reconciler.PackageParamName: "",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "MissingPackage",
		},
		{
			name: "missing PR isMerged",
			paramOverrides: map[string]string{
				reconciler.PRIsMergedParamName: "",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "MissingPullRequestIsMerged",
		},
		{
			name: "invalid PR number",
			paramOverrides: map[string]string{
				reconciler.PRNumberParamName: "banana",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "InvalidPullRequestNumber",
		},
		{
			name: "invalid package",
			paramOverrides: map[string]string{
				reconciler.PackageParamName: "not-owner-slash-repo",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "InvalidPackage",
		},
		{
			name: "invalid additional param",
			additionalParams: map[string]string{
				"something": "or other",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "UnexpectedParams",
		},
		{
			name: "fetching README 404",
			paramOverrides: map[string]string{
				reconciler.PRTitleParamName: "PR referencing TEP-1234",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "LoadingPRTEPs",
		},
		{
			name: "fetching PR comments 404",
			paramOverrides: map[string]string{
				reconciler.PRTitleParamName: "PR referencing TEP-1234",
			},
			requests: map[string]func(w http.ResponseWriter, r *http.Request){
				readmeURL: defaultREADMEHandlerFunc(),
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "CheckingPRComments",
		},
		{
			name: "adding comment for opened PR",
			paramOverrides: map[string]string{
				reconciler.PRTitleParamName: "PR referencing TEP-1234",
			},
			requests: map[string]func(w http.ResponseWriter, r *http.Request){
				readmeURL: defaultREADMEHandlerFunc(),
				"/repos/tektoncd/pipeline/issues/1/comments": noCommentsOnPRHandlerFunc(t),
			},
			expectedStatus: corev1.ConditionTrue,
			expectedReason: "CommentAdded",
		},
		{
			name: "editing comment for opened PR",
			paramOverrides: map[string]string{
				reconciler.PRTitleParamName: "PR referencing TEP-1234",
				reconciler.PRBodyParamName:  "With a body referencing TEP-5678",
			},
			requests: map[string]func(w http.ResponseWriter, r *http.Request){
				readmeURL: defaultREADMEHandlerFunc(),
				"/repos/tektoncd/pipeline/issues/1/comments": func(w http.ResponseWriter, r *http.Request) {
					require.Equal(t, "GET", r.Method)

					commentID := int64(1)
					commentUser := reconciler.BotUser
					commentBody := fmt.Sprintf("%s* [TEP-1234] (Some TEP Title)][https://github.com/tektoncd/community/blob/main/teps/1234-something-or-other.md),"+
						"current status: `proposed`\n\n<!-- TEP update: TEP-1234 status: proposed -->\n", reconciler.ToImplementingCommentHeader)
					comments := []*github.IssueComment{{
						ID:   &commentID,
						Body: &commentBody,
						User: &github.User{
							Login: &commentUser,
						},
					}}
					respBody, err := json.Marshal(comments)
					if err != nil {
						t.Fatal("marshalling GitHub comments")
					}
					_, _ = fmt.Fprint(w, string(respBody))
				},
				"/repos/tektoncd/pipeline/issues/comments/1": func(w http.ResponseWriter, r *http.Request) {
					require.Equal(t, "PATCH", r.Method)
					body, err := ioutil.ReadAll(r.Body)
					require.NoError(t, err)
					require.Contains(t, string(body), "TEP-5678")
					_, _ = fmt.Fprint(w, `{"id":1}`)
				},
			},
			expectedStatus: corev1.ConditionTrue,
			expectedReason: "CommentUpdated",
		},
		{
			name: "wrong state for opened PR",
			paramOverrides: map[string]string{
				reconciler.PRTitleParamName: "PR referencing TEP-4321",
			},
			requests: map[string]func(w http.ResponseWriter, r *http.Request){
				readmeURL: defaultREADMEHandlerFunc(),
				"/repos/tektoncd/pipeline/issues/1/comments": noCommentsOnPRHandlerFunc(t),
			},
			doesNothing: true,
		},
		{
			name: "wrong state for closed PR",
			paramOverrides: map[string]string{
				reconciler.ActionParamName:     "closed",
				reconciler.PRTitleParamName:    "PR referencing TEP-1234",
				reconciler.PRIsMergedParamName: "true",
			},
			requests: map[string]func(w http.ResponseWriter, r *http.Request){
				readmeURL: defaultREADMEHandlerFunc(),
				"/repos/tektoncd/pipeline/issues/1/comments": noCommentsOnPRHandlerFunc(t),
			},
			doesNothing: true,
		},
		{
			name: "adding comment for closed PR",
			paramOverrides: map[string]string{
				reconciler.ActionParamName:     "closed",
				reconciler.PRTitleParamName:    "PR referencing TEP-4321",
				reconciler.PRIsMergedParamName: "true",
			},
			requests: map[string]func(w http.ResponseWriter, r *http.Request){
				readmeURL: defaultREADMEHandlerFunc(),
				"/repos/tektoncd/pipeline/issues/1/comments": func(w http.ResponseWriter, r *http.Request) {
					switch r.Method {
					case "GET":
						commentID := int64(1)
						commentUser := reconciler.BotUser
						commentBody := fmt.Sprintf("%s* [TEP-4321] (Some TEP Title)][https://github.com/tektoncd/community/blob/main/teps/4321-something-or-other.md),"+
							"current status: `implementable`\n\n<!-- TEP update: TEP-4321 status: implementable -->\n", reconciler.ToImplementingCommentHeader)
						comments := []*github.IssueComment{{
							ID:   &commentID,
							Body: &commentBody,
							User: &github.User{
								Login: &commentUser,
							},
						}}
						respBody, err := json.Marshal(comments)
						if err != nil {
							t.Fatal("marshalling GitHub comments")
						}
						_, _ = fmt.Fprint(w, string(respBody))
						return
					case "POST":
						_, _ = fmt.Fprint(w, `{"id":1}`)
						return
					default:
						t.Errorf("unexpected method %s", r.Method)
					}
				},
			},
			expectedStatus: corev1.ConditionTrue,
			expectedReason: "CommentAdded",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			ghClient, mux, closeFunc := setupFakeGitHub()
			defer closeFunc()

			for k, v := range tc.requests {
				mux.HandleFunc(k, v)
			}

			r := &reconciler.Reconciler{GHClient: ghClient}

			run := &v1alpha1.Run{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-reconcile-run",
					Namespace: "foo",
				},
				Spec: v1alpha1.RunSpec{
					Params: constructRunParams(tc.paramOverrides, tc.additionalParams),
				},
			}
			if tc.kindRef != nil {
				run.Spec.Ref = tc.kindRef
			} else {
				run.Spec.Ref = defaultKindRef
			}

			err := r.ReconcileKind(ctx, run)
			if tc.expectedErr != nil {
				assert.Equal(t, tc.expectedErr, err)
			} else {
				require.NoError(t, err)
				condition := run.Status.GetCondition(apis.ConditionSucceeded)
				if tc.doesNothing {
					assert.Nil(t, condition)
				} else {
					require.NotNil(t, condition, "Condition missing in Run")

					if condition.Status != tc.expectedStatus {
						t.Errorf("Expected Run status to be %v but was %v", tc.expectedStatus, condition)
					}
					if condition.Reason != tc.expectedReason {
						t.Errorf("Expected reason to be %q but was %q", tc.expectedReason, condition.Reason)
					}
				}
			}
		})
	}
}

func defaultREADMEHandlerFunc() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, ghContentJSON(defaultTEPReadmeContent))
	}
}

func noCommentsOnPRHandlerFunc(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			_, _ = fmt.Fprint(w, `[]`)
			return
		case "POST":
			_, _ = fmt.Fprint(w, `{"id":1}`)
			return
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}
}

func constructRunParams(overrides map[string]string, additionalParams map[string]string) []v1beta1.Param {
	var params []v1beta1.Param

	for key, defaultValue := range defaultRunParams {
		if overrideValue, ok := overrides[key]; ok {
			if overrideValue != "" {
				params = append(params, v1beta1.Param{
					Name:  key,
					Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: overrideValue},
				})
			}
		} else {
			params = append(params, v1beta1.Param{
				Name:  key,
				Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: defaultValue},
			})
		}
	}

	for k, v := range additionalParams {
		params = append(params, v1beta1.Param{
			Name:  k,
			Value: v1beta1.ArrayOrString{Type: v1beta1.ParamTypeString, StringVal: v},
		})
	}

	return params
}

func ghContentJSON(content string) string {
	encContent := base64.StdEncoding.EncodeToString([]byte(content))

	return fmt.Sprintf(`{
		  "type": "file",
          "content": "%s",
		  "encoding": "base64",
		  "size": 1234,
		  "name": "SOMEFILE",
		  "path": "SOMEPATH"
		}`, encContent)
}

func setupFakeGitHub() (*github.Client, *http.ServeMux, func()) {
	apiPath := "/api-v3"

	mux := http.NewServeMux()

	handler := http.NewServeMux()
	handler.Handle(apiPath+"/", http.StripPrefix(apiPath, mux))

	server := httptest.NewServer(handler)

	client := github.NewClient(nil)
	ghURL, _ := url.Parse(server.URL + apiPath + "/")
	client.BaseURL = ghURL
	client.UploadURL = ghURL

	return client, mux, server.Close
}
