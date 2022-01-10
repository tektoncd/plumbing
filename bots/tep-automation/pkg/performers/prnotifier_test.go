package performers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/v41/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/ghclient"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/performers"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/tep"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/testutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kreconciler "knative.dev/pkg/reconciler"
)

func TestPROpenedComment(t *testing.T) {
	teps := []tep.TEPInfo{
		{
			ID:           "1234",
			Title:        "Some TEP Title",
			Status:       tep.ProposedStatus,
			Filename:     "1234-something-or-other.md",
			LastModified: time.Date(2021, time.December, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:           "5678",
			Title:        "Some Other TEP Title",
			Status:       tep.ImplementableStatus,
			Filename:     "5678-insert-filename-here.md",
			LastModified: time.Date(2021, time.December, 21, 0, 0, 0, 0, time.UTC),
		},
	}

	expectedComment := performers.ToImplementingCommentHeader +
		" * [TEP-1234 (Some TEP Title)](https://github.com/tektoncd/community/blob/main/teps/1234-something-or-other.md), current status: `proposed`\n" +
		" * [TEP-5678 (Some Other TEP Title)](https://github.com/tektoncd/community/blob/main/teps/5678-insert-filename-here.md), current status: `implementable`\n" +
		"\n" +
		"<!-- TEP update: TEP-1234 status: proposed -->\n" +
		"<!-- TEP update: TEP-5678 status: implementable -->\n"

	assert.Equal(t, expectedComment, performers.PROpenedComment(teps))
}

func TestPRClosedComment(t *testing.T) {
	teps := []tep.TEPInfo{
		{
			ID:           "1234",
			Title:        "Some TEP Title",
			Status:       tep.ImplementingStatus,
			Filename:     "1234-something-or-other.md",
			LastModified: time.Date(2021, time.December, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:           "5678",
			Title:        "Some Other TEP Title",
			Status:       tep.ImplementingStatus,
			Filename:     "5678-insert-filename-here.md",
			LastModified: time.Date(2021, time.December, 21, 0, 0, 0, 0, time.UTC),
		},
	}

	expectedComment := performers.ToImplementedCommentHeader +
		" * [TEP-1234 (Some TEP Title)](https://github.com/tektoncd/community/blob/main/teps/1234-something-or-other.md), current status: `implementing`\n" +
		" * [TEP-5678 (Some Other TEP Title)](https://github.com/tektoncd/community/blob/main/teps/5678-insert-filename-here.md), current status: `implementing`\n" +
		"\n" +
		"<!-- TEP update: TEP-1234 status: implementing -->\n" +
		"<!-- TEP update: TEP-5678 status: implementing -->\n"

	assert.Equal(t, expectedComment, performers.PRMergedComment(teps))
}

func TestPRNotifier_Perform(t *testing.T) {
	testCases := []struct {
		name              string
		paramOverrides    map[string]string
		additionalParams  map[string]string
		requests          map[string]func(w http.ResponseWriter, r *http.Request)
		doesNothing       bool
		expectedEventType string
		expectedReason    string
		expectedErr       error
	}{
		{
			name:        "no TEPs",
			doesNothing: true,
		},
		{
			name: "wrong action",
			paramOverrides: map[string]string{
				tep.ActionParamName: "assigned",
			},
			doesNothing: true,
		},
		{
			name: "closed but unmerged",
			paramOverrides: map[string]string{
				tep.ActionParamName: "closed",
			},
			doesNothing: true,
		},
		{
			name: "fetching README 404",
			paramOverrides: map[string]string{
				tep.PRTitleParamName: "PR referencing TEP-1234",
			},
			expectedEventType: corev1.EventTypeWarning,
			expectedReason:    "LoadingPRTEPs",
		},
		{
			name: "fetching PR comments 404",
			paramOverrides: map[string]string{
				tep.PRTitleParamName: "PR referencing TEP-1234",
			},
			requests: map[string]func(w http.ResponseWriter, r *http.Request){
				testutil.ReadmeURL: testutil.DefaultREADMEHandlerFunc(),
			},
			expectedEventType: corev1.EventTypeWarning,
			expectedReason:    "CheckingPRComments",
		},
		{
			name: "adding comment for opened PR",
			paramOverrides: map[string]string{
				tep.PRTitleParamName: "PR referencing TEP-1234",
			},
			requests: map[string]func(w http.ResponseWriter, r *http.Request){
				testutil.ReadmeURL:                           testutil.DefaultREADMEHandlerFunc(),
				"/repos/tektoncd/pipeline/issues/1/comments": testutil.NoCommentsOnPRHandlerFunc(t),
			},
			expectedEventType: corev1.EventTypeNormal,
			expectedReason:    "CommentAdded",
		},
		{
			name: "editing comment for opened PR",
			paramOverrides: map[string]string{
				tep.PRTitleParamName: "PR referencing TEP-1234",
				tep.PRBodyParamName:  "With a body referencing TEP-5678",
			},
			requests: map[string]func(w http.ResponseWriter, r *http.Request){
				testutil.ReadmeURL: testutil.DefaultREADMEHandlerFunc(),
				"/repos/tektoncd/pipeline/issues/1/comments": func(w http.ResponseWriter, r *http.Request) {
					require.Equal(t, "GET", r.Method)

					commentID := int64(1)
					commentUser := ghclient.BotUser
					commentBody := fmt.Sprintf("%s* [TEP-1234] (Some TEP Title)][https://github.com/tektoncd/community/blob/main/teps/1234-something-or-other.md),"+
						"current status: `proposed`\n\n<!-- TEP update: TEP-1234 status: proposed -->\n", performers.ToImplementingCommentHeader)
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
			expectedEventType: corev1.EventTypeNormal,
			expectedReason:    "CommentUpdated",
		},
		{
			name: "wrong state for opened PR",
			paramOverrides: map[string]string{
				tep.PRTitleParamName: "PR referencing TEP-4321",
			},
			requests: map[string]func(w http.ResponseWriter, r *http.Request){
				testutil.ReadmeURL:                           testutil.DefaultREADMEHandlerFunc(),
				"/repos/tektoncd/pipeline/issues/1/comments": testutil.NoCommentsOnPRHandlerFunc(t),
			},
			doesNothing: true,
		},
		{
			name: "wrong state for closed PR",
			paramOverrides: map[string]string{
				tep.ActionParamName:     "closed",
				tep.PRTitleParamName:    "PR referencing TEP-1234",
				tep.PRIsMergedParamName: "true",
			},
			requests: map[string]func(w http.ResponseWriter, r *http.Request){
				testutil.ReadmeURL:                           testutil.DefaultREADMEHandlerFunc(),
				"/repos/tektoncd/pipeline/issues/1/comments": testutil.NoCommentsOnPRHandlerFunc(t),
			},
			doesNothing: true,
		},
		{
			name: "adding comment for closed PR",
			paramOverrides: map[string]string{
				tep.ActionParamName:     "closed",
				tep.PRTitleParamName:    "PR referencing TEP-4321",
				tep.PRIsMergedParamName: "true",
			},
			requests: map[string]func(w http.ResponseWriter, r *http.Request){
				testutil.ReadmeURL: testutil.DefaultREADMEHandlerFunc(),
				"/repos/tektoncd/pipeline/issues/1/comments": func(w http.ResponseWriter, r *http.Request) {
					switch r.Method {
					case "GET":
						commentID := int64(1)
						commentUser := ghclient.BotUser
						commentBody := fmt.Sprintf("%s* [TEP-4321] (Some TEP Title)][https://github.com/tektoncd/community/blob/main/teps/4321-something-or-other.md),"+
							"current status: `implementable`\n\n<!-- TEP update: TEP-4321 status: implementable -->\n", performers.ToImplementingCommentHeader)
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
			expectedEventType: corev1.EventTypeNormal,
			expectedReason:    "CommentAdded",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			ghClient, mux, closeFunc := testutil.SetupFakeGitHub()
			defer closeFunc()

			tgc := ghclient.NewTEPGHClient(ghClient)

			for k, v := range tc.requests {
				mux.HandleFunc(k, v)
			}

			n := performers.NewPRNotifier(tgc)

			run := &v1alpha1.Run{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-reconcile-run",
					Namespace: "foo",
				},
				Spec: v1alpha1.RunSpec{
					Params: testutil.ConstructRunParams(tc.paramOverrides, tc.additionalParams),
				},
			}

			opts, err := performers.ToPerformerOptions(run)
			require.NoError(t, err)

			err = n.Perform(ctx, opts)
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
							t.Errorf("Expected event type to be %s but was %s", tc.expectedEventType, recEvt.EventType)
						}
						if recEvt.Reason != tc.expectedReason {
							t.Errorf("Expected reason to be %q but was %q", tc.expectedReason, recEvt.Reason)
						}
					}
				}
			}
		})
	}
}
