package reconciler

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/diff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcile(t *testing.T) {
	sha := "abcd1234"
	botUser := "k8s-ci-robot"

	testCases := []struct {
		name             string
		existingStatuses []*scm.Status
		info             *StatusInfo
		expectedStatuses []*scm.Status
	}{
		{
			name: "first status update",
			info: &StatusInfo{
				Repo:        "some-org/some-repo",
				SHA:         sha,
				JobName:     "some-job",
				State:       "success",
				Description: "some description",
				TargetURL:   "http://some/where",
			},
			expectedStatuses: []*scm.Status{{
				State:  scm.ToState("success"),
				Label:  "some-job",
				Desc:   "some description",
				Target: "http://some/where",
			}},
		}, {
			name: "replace job",
			info: &StatusInfo{
				Repo:      "some-org/some-repo",
				SHA:       sha,
				JobName:   "some-job",
				State:     "success",
				TargetURL: "http://some/where",
			},
			existingStatuses: []*scm.Status{{
				State:  scm.ToState("failure"),
				Label:  "some-job",
				Target: "http://some/where",
			}},
			expectedStatuses: []*scm.Status{{
				State:  scm.ToState("success"),
				Label:  "some-job",
				Target: "http://some/where",
			}},
		}, {
			name: "replace job retaining existing other job",
			info: &StatusInfo{
				Repo:      "some-org/some-repo",
				SHA:       sha,
				JobName:   "some-job",
				State:     "success",
				TargetURL: "http://some/where",
			},
			existingStatuses: []*scm.Status{{
				State:  scm.ToState("success"),
				Label:  "other-job",
				Target: "http://some/where/else",
			}},
			expectedStatuses: []*scm.Status{{
				State:  scm.ToState("success"),
				Label:  "other-job",
				Target: "http://some/where/else",
			}, {
				State:  scm.ToState("success"),
				Label:  "some-job",
				Target: "http://some/where",
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakeScmClient, fc := fake.NewDefault()

			fc.Statuses[sha] = tc.existingStatuses

			r := &Reconciler{
				SCMClient: fakeScmClient,
				BotUser:   botUser,
			}

			testRun := statusInfoToRun(tc.info)

			if err := r.ReconcileKind(context.Background(), testRun); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if d := cmp.Diff(tc.expectedStatuses, fc.Statuses[sha]); d != "" {
				t.Errorf("comments differed from expected: %s", diff.PrintWantGot(d))
			}
		})
	}
}

func statusInfoToRun(info *StatusInfo) *v1alpha1.Run {
	return &v1alpha1.Run{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-run",
			Namespace: "foo",
		},
		Spec: v1alpha1.RunSpec{
			Ref: &v1beta1.TaskRef{
				Kind:       "PRStatusUpdater",
				APIVersion: "custom.tekton.dev/v0",
			},
			Params: []v1beta1.Param{{
				Name:  repoKey,
				Value: *v1beta1.NewStructuredValues(info.Repo),
			}, {
				Name:  shaKey,
				Value: *v1beta1.NewStructuredValues(info.SHA),
			}, {
				Name:  jobNameKey,
				Value: *v1beta1.NewStructuredValues(info.JobName),
			}, {
				Name:  stateKey,
				Value: *v1beta1.NewStructuredValues(info.State),
			}, {
				Name:  descriptionKey,
				Value: *v1beta1.NewStructuredValues(info.Description),
			}, {
				Name:  targetURLKey,
				Value: *v1beta1.NewStructuredValues(info.TargetURL),
			}},
		},
	}

}
