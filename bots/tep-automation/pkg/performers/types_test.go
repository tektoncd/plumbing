package performers_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/performers"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/tep"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/testutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kreconciler "knative.dev/pkg/reconciler"
)

func TestToPerformerOptions(t *testing.T) {
	testCases := []struct {
		name              string
		paramOverrides    map[string]string
		additionalParams  map[string]string
		expectedEventType string
		expectedReason    string
	}{
		{
			name: "all params valid",
		},
		{
			name: "missing action",
			paramOverrides: map[string]string{
				tep.ActionParamName: "",
			},
			expectedEventType: corev1.EventTypeWarning,
			expectedReason:    "MissingPullRequestAction",
		},
		{
			name: "missing PR number",
			paramOverrides: map[string]string{
				tep.PRNumberParamName: "",
			},
			expectedEventType: corev1.EventTypeWarning,
			expectedReason:    "MissingPullRequestNumber",
		},
		{
			name: "missing PR title",
			paramOverrides: map[string]string{
				tep.PRTitleParamName: "",
			},
			expectedEventType: corev1.EventTypeWarning,
			expectedReason:    "MissingPullRequestTitle",
		},
		{
			name: "missing PR body",
			paramOverrides: map[string]string{
				tep.PRBodyParamName: "",
			},
			expectedEventType: corev1.EventTypeWarning,
			expectedReason:    "MissingPullRequestBody",
		},
		{
			name: "missing package",
			paramOverrides: map[string]string{
				tep.PackageParamName: "",
			},
			expectedEventType: corev1.EventTypeWarning,
			expectedReason:    "MissingPackage",
		},
		{
			name: "missing PR isMerged",
			paramOverrides: map[string]string{
				tep.PRIsMergedParamName: "",
			},
			expectedEventType: corev1.EventTypeWarning,
			expectedReason:    "MissingPullRequestIsMerged",
		},
		{
			name: "invalid PR number",
			paramOverrides: map[string]string{
				tep.PRNumberParamName: "banana",
			},
			expectedEventType: corev1.EventTypeWarning,
			expectedReason:    "InvalidPullRequestNumber",
		},
		{
			name: "invalid package",
			paramOverrides: map[string]string{
				tep.PackageParamName: "not-owner-slash-repo",
			},
			expectedEventType: corev1.EventTypeWarning,
			expectedReason:    "InvalidPackage",
		},
		{
			name: "invalid additional param",
			additionalParams: map[string]string{
				"something": "or other",
			},
			expectedEventType: corev1.EventTypeWarning,
			expectedReason:    "UnexpectedParams",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			run := &v1alpha1.Run{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-reconcile-run",
					Namespace: "foo",
				},
				Spec: v1alpha1.RunSpec{
					Params: testutil.ConstructRunParams(tc.paramOverrides, tc.additionalParams),
				},
			}

			_, err := performers.ToPerformerOptions(run)
			if tc.expectedReason != "" || tc.expectedEventType != "" {
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
			} else {
				require.NoError(t, err)
			}
		})
	}
}
