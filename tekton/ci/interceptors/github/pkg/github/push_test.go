package github

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v34/github"
	"github.com/tektoncd/plumbing/tekton/ci/interceptors/github/pkg/github/bindings"
	pb "github.com/tektoncd/plumbing/tekton/ci/interceptors/github/pkg/proto/v1alpha1/config_go_proto"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

func TestExecute_Push(t *testing.T) {

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	client := github.NewClient(srv.Client())
	client.BaseURL = mustParseURL(srv.URL + "/")

	ctx := context.Background()
	h := &Push{}

	f, err := ioutil.ReadFile("testdata/push.json")
	if err != nil {
		log.Fatal(err)
	}
	req := &v1alpha1.InterceptorRequest{
		Body: string(f),
		Header: map[string][]string{
			"X-Github-Event": {"push"},
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
				Push: &pb.PushConfig{},
			},
			ok: true,
		},
		{
			name: "only tags",
			cfg: &pb.Config{
				Push: &pb.PushConfig{
					Ref: []string{"refs/tags/*"},
				},
			},
			ok: true,
		},
		{
			name: "all refs",
			cfg: &pb.Config{
				Push: &pb.PushConfig{
					Ref: []string{"**"},
				},
			},
			ok: true,
		},
		{
			name: "only branches",
			cfg: &pb.Config{
				Push: &pb.PushConfig{
					Ref: []string{"refs/heads/*"},
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
			resp, err := h.Execute(ctx, client, tc.cfg, req)
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
						Revision: "6113728f27ae82c7b1a177c8d03f9e96e0adf246",
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
