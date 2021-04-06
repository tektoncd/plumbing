package github

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/plumbing/tekton/ci/interceptors/github/pkg/github/bindings"
	pb "github.com/tektoncd/plumbing/tekton/ci/interceptors/github/pkg/proto/v1alpha1/config_go_proto"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

func TestExecute_PullRequest(t *testing.T) {
	ctx := context.Background()
	h := &PullRequest{}

	f, err := ioutil.ReadFile("testdata/pull_request.json")
	if err != nil {
		log.Fatal(err)
	}
	req := &v1alpha1.InterceptorRequest{
		Body: string(f),
		Header: map[string][]string{
			"X-Github-Event": {"pull_request"},
		},
	}
	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(req); err != nil {
		log.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		cfg  *pb.Config
		ok   bool
	}{
		{
			name: "default",
			cfg: &pb.Config{
				PullRequest: &pb.PullRequestConfig{},
			},
			ok: true,
		},
		{
			name: "all branches",
			cfg: &pb.Config{
				PullRequest: &pb.PullRequestConfig{
					Branch: []string{"*"},
				},
			},
			ok: true,
		},
		{
			name: "non-matching branch",
			cfg: &pb.Config{
				Push: &pb.PushConfig{
					Ref: []string{"main"},
				},
			},
			ok: false,
		},
		{
			name: "comment config set",
			cfg: &pb.Config{
				PullRequest: &pb.PullRequestConfig{
					Branch:  []string{"*"},
					Comment: &pb.PullRequestConfig_CommentConfig{},
				},
			},
			ok: false,
		},
		{
			name: "no config",
			cfg:  &pb.Config{},
			ok:   false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := h.Execute(ctx, nil, tc.cfg, req)
			if !tc.ok {
				if err == nil {
					t.Fatalf("expected failure, got (%+v, %v)", resp, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected success: %t, got (%+v, %v)", tc.ok, resp, err)
			}
			if !tc.ok {
				return
			}

			want := &v1alpha1.InterceptorResponse{
				Continue: true,
				Extensions: map[string]interface{}{
					"git": bindings.Git{
						URL:      "https://github.com/Codertocat/Hello-World.git",
						Revision: "ec26c3e57ca3a959ca5aad62de7213c562f8c821",
					},
					"github": bindings.GitHub{
						Owner: "Codertocat",
						Repo:  "Hello-World",
					},
				},
			}
			if diff := cmp.Diff(want, resp); diff != "" {
				t.Error(diff)
			}
		})
	}
}
