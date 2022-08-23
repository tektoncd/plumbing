package reconciler

import (
	"context"
	"fmt"
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
	owner := "some-org"
	botUser := "k8s-ci-robot"

	testCases := []struct {
		name             string
		existingComments []*scm.Comment
		info             *ReportInfo
		expectedComments []*scm.Comment
	}{
		{
			name: "first comment",
			info: &ReportInfo{
				Repo:       "some-repo",
				PRNumber:   5,
				SHA:        "abcd1234",
				JobName:    "some-job",
				IsSuccess:  false,
				LogURL:     "http://some/where",
				IsOptional: false,
			},
			expectedComments: []*scm.Comment{{
				Body: `The following Tekton test **failed**:

Test name | Commit | Details | Required | Rerun command
--- | --- | --- | --- | ---
some-job | abcd1234 | [link](http://some/where) | true | ` + "`/test some-job`" + `

<!-- Tekton test report -->`,
				Author: scm.User{Login: botUser},
			}},
		}, {
			name: "replace comment retaining existing failure",
			info: &ReportInfo{
				Repo:       "some-repo",
				PRNumber:   5,
				SHA:        "abcd1234",
				JobName:    "some-job",
				IsSuccess:  false,
				LogURL:     "http://some/where",
				IsOptional: false,
			},
			existingComments: []*scm.Comment{{
				ID: 1,
				Body: `The following Tekton tests **failed**:

Test name | Commit | Details | Required | Rerun command
--- | --- | --- | --- | ---
some-job | 12345678 | [link](http://some/where) | true | ` + "`/test some-job`" + `
some-other-job | 12345678 | [link](http://some/where/else) | true | ` + "`/test some-other-job`" + `

<!-- Tekton test report -->`,
				Author: scm.User{Login: botUser},
			}},
			expectedComments: []*scm.Comment{{
				Body: `The following Tekton tests **failed**:

Test name | Commit | Details | Required | Rerun command
--- | --- | --- | --- | ---
some-other-job | 12345678 | [link](http://some/where/else) | true | ` + "`/test some-other-job`" + `
some-job | abcd1234 | [link](http://some/where) | true | ` + "`/test some-job`" + `

<!-- Tekton test report -->`,
				Author: scm.User{Login: botUser},
			}},
		}, {
			name: "replace comment removing obsolete",
			info: &ReportInfo{
				Repo:       "some-repo",
				PRNumber:   5,
				SHA:        "abcd1234",
				JobName:    "some-job",
				IsSuccess:  true,
				LogURL:     "http://some/where",
				IsOptional: false,
			},
			existingComments: []*scm.Comment{{
				ID: 1,
				Body: `The following Tekton tests **failed**:

Test name | Commit | Details | Required | Rerun command
--- | --- | --- | --- | ---
some-job | 12345678 | [link](http://some/where) | true | ` + "`/test some-job`" + `
some-other-job | 12345678 | [link](http://some/where/else) | true | ` + "`/test some-other-job`" + `

<!-- Tekton test report -->`,
				Author: scm.User{Login: botUser},
			}},
			expectedComments: []*scm.Comment{{
				Body: `The following Tekton test **failed**:

Test name | Commit | Details | Required | Rerun command
--- | --- | --- | --- | ---
some-other-job | 12345678 | [link](http://some/where/else) | true | ` + "`/test some-other-job`" + `

<!-- Tekton test report -->`,
				Author: scm.User{Login: botUser},
			}},
		}, {
			name: "delete comment",
			info: &ReportInfo{
				Repo:       "some-repo",
				PRNumber:   5,
				SHA:        "abcd1234",
				JobName:    "some-job",
				IsSuccess:  true,
				LogURL:     "http://some/where",
				IsOptional: false,
			},
			existingComments: []*scm.Comment{{
				ID: 1,
				Body: `The following Tekton test **failed**:

Test name | Commit | Details | Required | Rerun command
--- | --- | --- | --- | ---
some-job | abcd1234 | [link](http://some/where) | true | ` + "`/test some-job`" + `

<!-- Tekton test report -->`,
				Author: scm.User{Login: botUser},
			}},
			expectedComments: []*scm.Comment{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakeScmClient, fc := fake.NewDefault()

			fc.PullRequestComments[5] = tc.existingComments

			r := &Reconciler{
				SCMClient: fakeScmClient,
				Owner:     owner,
				BotUser:   botUser,
			}

			testRun := reportInfoToRun(tc.info)

			if err := r.ReconcileKind(context.Background(), testRun); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if d := cmp.Diff(tc.expectedComments, fc.PullRequestComments[5]); d != "" {
				t.Errorf("comments differed from expected: %s", diff.PrintWantGot(d))
			}
		})
	}
}

func reportInfoToRun(info *ReportInfo) *v1alpha1.Run {
	return &v1alpha1.Run{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-run",
			Namespace: "foo",
		},
		Spec: v1alpha1.RunSpec{
			Ref: &v1beta1.TaskRef{
				Kind:       "PRCommenter",
				APIVersion: "custom.tekton.dev/v0",
			},
			Params: []v1beta1.Param{{
				Name:  repoKey,
				Value: *v1beta1.NewStructuredValues(info.Repo),
			}, {
				Name:  prNumberKey,
				Value: *v1beta1.NewStructuredValues(fmt.Sprintf("%d", info.PRNumber)),
			}, {
				Name:  shaKey,
				Value: *v1beta1.NewStructuredValues(info.SHA),
			}, {
				Name:  jobNameKey,
				Value: *v1beta1.NewStructuredValues(info.JobName),
			}, {
				Name:  successKey,
				Value: *v1beta1.NewStructuredValues(fmt.Sprintf("%t", info.IsSuccess)),
			}, {
				Name:  optionalKey,
				Value: *v1beta1.NewStructuredValues(fmt.Sprintf("%t", info.IsOptional)),
			}, {
				Name:  logURLKey,
				Value: *v1beta1.NewStructuredValues(info.LogURL),
			}},
		},
	}

}
