package reconciler

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/diff"
)

func TestStatusInfoFromRun(t *testing.T) {
	testCases := []struct {
		name string
		run  *v1beta1.CustomRun
		info *StatusInfo
		err  string
	}{
		{
			name: "valid run",
			run: &v1beta1.CustomRun{
				Spec: v1beta1.CustomRunSpec{
					Params: []v1beta1.Param{
						{
							Name:  repoKey,
							Value: *v1beta1.NewStructuredValues("some-org/some-repo"),
						}, {
							Name:  shaKey,
							Value: *v1beta1.NewStructuredValues("abcd1234"),
						}, {
							Name:  jobNameKey,
							Value: *v1beta1.NewStructuredValues("some-job"),
						}, {
							Name:  stateKey,
							Value: *v1beta1.NewStructuredValues("success"),
						}, {
							Name:  descriptionKey,
							Value: *v1beta1.NewStructuredValues("some description"),
						}, {
							Name:  targetURLKey,
							Value: *v1beta1.NewStructuredValues("http://some/where"),
						},
					},
				},
			},
			info: &StatusInfo{
				Repo:        "some-org/some-repo",
				SHA:         "abcd1234",
				JobName:     "some-job",
				State:       "success",
				Description: "some description",
				TargetURL:   "http://some/where",
			},
		}, {
			name: "missing repo",
			run: &v1beta1.CustomRun{
				Spec: v1beta1.CustomRunSpec{
					Params: []v1beta1.Param{
						{
							Name:  shaKey,
							Value: *v1beta1.NewStructuredValues("abcd1234"),
						}, {
							Name:  jobNameKey,
							Value: *v1beta1.NewStructuredValues("some-job"),
						}, {
							Name:  stateKey,
							Value: *v1beta1.NewStructuredValues("success"),
						}, {
							Name:  descriptionKey,
							Value: *v1beta1.NewStructuredValues("some description"),
						}, {
							Name:  targetURLKey,
							Value: *v1beta1.NewStructuredValues("http://some/where"),
						},
					},
				},
			},
			err: "missing field(s): repo",
		}, {
			name: "missing SHA",
			run: &v1beta1.CustomRun{
				Spec: v1beta1.CustomRunSpec{
					Params: []v1beta1.Param{
						{
							Name:  repoKey,
							Value: *v1beta1.NewStructuredValues("some-org/some-repo"),
						}, {
							Name:  jobNameKey,
							Value: *v1beta1.NewStructuredValues("some-job"),
						}, {
							Name:  stateKey,
							Value: *v1beta1.NewStructuredValues("success"),
						}, {
							Name:  descriptionKey,
							Value: *v1beta1.NewStructuredValues("some description"),
						}, {
							Name:  targetURLKey,
							Value: *v1beta1.NewStructuredValues("http://some/where"),
						},
					},
				},
			},
			err: "missing field(s): sha",
		}, {
			name: "missing job name",
			run: &v1beta1.CustomRun{
				Spec: v1beta1.CustomRunSpec{
					Params: []v1beta1.Param{
						{
							Name:  repoKey,
							Value: *v1beta1.NewStructuredValues("some-org/some-repo"),
						}, {
							Name:  shaKey,
							Value: *v1beta1.NewStructuredValues("abcd1234"),
						}, {
							Name:  stateKey,
							Value: *v1beta1.NewStructuredValues("success"),
						}, {
							Name:  descriptionKey,
							Value: *v1beta1.NewStructuredValues("some description"),
						}, {
							Name:  targetURLKey,
							Value: *v1beta1.NewStructuredValues("http://some/where"),
						},
					},
				},
			},
			err: "missing field(s): jobName",
		}, {
			name: "missing state",
			run: &v1beta1.CustomRun{
				Spec: v1beta1.CustomRunSpec{
					Params: []v1beta1.Param{
						{
							Name:  repoKey,
							Value: *v1beta1.NewStructuredValues("some-org/some-repo"),
						}, {
							Name:  shaKey,
							Value: *v1beta1.NewStructuredValues("abcd1234"),
						}, {
							Name:  jobNameKey,
							Value: *v1beta1.NewStructuredValues("some-job"),
						}, {
							Name:  descriptionKey,
							Value: *v1beta1.NewStructuredValues("some description"),
						}, {
							Name:  targetURLKey,
							Value: *v1beta1.NewStructuredValues("http://some/where"),
						},
					},
				},
			},
			err: "missing field(s): state",
		}, {
			name: "non-string value",
			run: &v1beta1.CustomRun{
				Spec: v1beta1.CustomRunSpec{
					Params: []v1beta1.Param{{
						Name:  repoKey,
						Value: *v1beta1.NewStructuredValues("bob", "steve"),
					}, {
						Name:  shaKey,
						Value: *v1beta1.NewStructuredValues("abcd1234"),
					}, {
						Name:  jobNameKey,
						Value: *v1beta1.NewStructuredValues("some-job"),
					}, {
						Name:  stateKey,
						Value: *v1beta1.NewStructuredValues("success"),
					}, {
						Name:  descriptionKey,
						Value: *v1beta1.NewStructuredValues("some description"),
					}, {
						Name:  targetURLKey,
						Value: *v1beta1.NewStructuredValues("http://some/where"),
					}},
				},
			},
			err: "invalid value: should be a string, is array: repo",
		}, {
			name: "invalid state value",
			run: &v1beta1.CustomRun{
				Spec: v1beta1.CustomRunSpec{
					Params: []v1beta1.Param{{
						Name:  repoKey,
						Value: *v1beta1.NewStructuredValues("some-org/some-repo"),
					}, {
						Name:  shaKey,
						Value: *v1beta1.NewStructuredValues("abcd1234"),
					}, {
						Name:  jobNameKey,
						Value: *v1beta1.NewStructuredValues("some-job"),
					}, {
						Name:  stateKey,
						Value: *v1beta1.NewStructuredValues("on fire"),
					}, {
						Name:  descriptionKey,
						Value: *v1beta1.NewStructuredValues("some description"),
					}, {
						Name:  targetURLKey,
						Value: *v1beta1.NewStructuredValues("http://some/where"),
					}},
				},
			},
			err: "invalid value: must be one of 'error', 'pending', 'failure', or 'success', but was on fire: state",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := StatusInfoFromRun(tc.run)
			if err != nil {
				if tc.err == "" {
					t.Fatalf("expected no error, but got '%s'", err.Error())
				} else if tc.err != err.Error() {
					t.Fatalf("expected error '%s', but got '%s'", tc.err, err.Error())
				}
			} else {
				if tc.err != "" {
					t.Fatalf("expected error '%s', but got no error", tc.err)
				}

				if d := cmp.Diff(tc.info, info); d != "" {
					t.Errorf("result differs: %s", diff.PrintWantGot(d))
				}
			}
		})
	}
}
