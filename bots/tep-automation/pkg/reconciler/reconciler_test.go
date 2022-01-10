package reconciler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/tep"

	"github.com/google/go-github/v41/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/ghclient"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/performers"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/reconciler"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/testutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestReconcileKind(t *testing.T) {
	defaultKindRef := &v1beta1.TaskRef{
		APIVersion: v1beta1.SchemeGroupVersion.String(),
		Kind:       reconciler.Kind,
	}

	defaultContentRef := "some-ref"

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
			name: "missing action",
			paramOverrides: map[string]string{
				tep.ActionParamName: "",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "MissingPullRequestAction",
		},
		{
			name: "missing PR number",
			paramOverrides: map[string]string{
				tep.PRNumberParamName: "",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "MissingPullRequestNumber",
		},
		{
			name: "missing PR title",
			paramOverrides: map[string]string{
				tep.PRTitleParamName: "",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "MissingPullRequestTitle",
		},
		{
			name: "missing PR body",
			paramOverrides: map[string]string{
				tep.PRBodyParamName: "",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "MissingPullRequestBody",
		},
		{
			name: "missing package",
			paramOverrides: map[string]string{
				tep.PackageParamName: "",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "MissingPackage",
		},
		{
			name: "missing PR isMerged",
			paramOverrides: map[string]string{
				tep.PRIsMergedParamName: "",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "MissingPullRequestIsMerged",
		},
		{
			name: "invalid PR number",
			paramOverrides: map[string]string{
				tep.PRNumberParamName: "banana",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "InvalidPullRequestNumber",
		},
		{
			name: "invalid package",
			paramOverrides: map[string]string{
				tep.PackageParamName: "not-owner-slash-repo",
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
				tep.PRTitleParamName: "PR referencing TEP-1234",
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "LoadingPRTEPs",
		},
		{
			name: "fetching PR comments 404",
			paramOverrides: map[string]string{
				tep.PRTitleParamName: "PR referencing TEP-1234",
			},
			requests: map[string]func(w http.ResponseWriter, r *http.Request){
				testutil.ReadmeURL: testutil.DefaultREADMEHandlerFunc(),
			},
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "CheckingPRComments",
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
			expectedStatus: corev1.ConditionTrue,
			expectedReason: "AllSucceeded",
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
			expectedStatus: corev1.ConditionTrue,
			expectedReason: "AllSucceeded",
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
			expectedStatus: corev1.ConditionTrue,
			expectedReason: "AllSucceeded",
		},
		{
			name: "create tracking issue",
			paramOverrides: map[string]string{
				tep.GitRevisionParamName: defaultContentRef,
				tep.PackageParamName:     "tektoncd/community",
			},
			requests: map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/repos/%s/%s/pulls/%d/files", ghclient.TEPsOwner, ghclient.TEPsRepo, 1): func(w http.ResponseWriter, r *http.Request) {
					respBody, err := json.Marshal([]*github.CommitFile{{
						SHA:      github.String("some-sha"),
						Filename: github.String("teps/1234-some-proposal.md"),
						Status:   github.String("added"),
					}})
					if err != nil {
						t.Fatal("marshalling GitHub file list")
					}
					_, _ = fmt.Fprint(w, string(respBody))
				},
				fmt.Sprintf("/repos/%s/%s/contents/teps/1234-some-proposal.md", ghclient.TEPsOwner, ghclient.TEPsRepo): func(w http.ResponseWriter, r *http.Request) {
					if !strings.HasSuffix(r.RequestURI, fmt.Sprintf("?ref=%s", defaultContentRef)) {
						t.Errorf("expected request for ref %s, but URI was %s", defaultContentRef, r.RequestURI)
					}

					fileContent, err := ioutil.ReadFile(filepath.Join("..", "ghclient", "testdata", "1234-some-proposal.md"))
					if err != nil {
						t.Fatal("reading ../ghclient/testdata/1234-some-proposal.md")
					}

					_, _ = fmt.Fprint(w, testutil.GHContentJSON(string(fileContent)))
				},
				fmt.Sprintf("/repos/%s/%s/issues", ghclient.TEPsOwner, ghclient.TEPsRepo): func(w http.ResponseWriter, r *http.Request) {
					if r.Method == "GET" {
						respBody, err := json.Marshal([]*github.Issue{})
						if err != nil {
							t.Fatal("marshalling GitHub issue list")
						}
						_, _ = fmt.Fprint(w, string(respBody))
					} else if r.Method == "POST" {
						_, _ = fmt.Fprint(w, `{"number":2}`)
					}
				},
			},
			expectedStatus: corev1.ConditionTrue,
			expectedReason: "AllSucceeded",
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

			r := &reconciler.Reconciler{
				GHClient: tgc,
				Performers: []performers.Performer{
					performers.NewPRNotifier(tgc),
					performers.NewIssueCreator(tgc),
				},
			}

			run := &v1alpha1.Run{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-reconcile-run",
					Namespace: "foo",
				},
				Spec: v1alpha1.RunSpec{
					Params: testutil.ConstructRunParams(tc.paramOverrides, tc.additionalParams),
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
