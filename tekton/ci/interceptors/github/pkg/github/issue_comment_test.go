package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v34/github"
	"github.com/tektoncd/plumbing/tekton/ci/interceptors/github/pkg/github/bindings"
	pb "github.com/tektoncd/plumbing/tekton/ci/interceptors/github/pkg/proto/v1alpha1/config_go_proto"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

func TestExecute_IssueComment(t *testing.T) {
	ctx := context.Background()
	h := &IssueComment{}

	// Setup fake GitHub client.
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	client := github.NewClient(srv.Client())
	client.BaseURL = mustParseURL(srv.URL + "/")
	mux.HandleFunc("/repos/tektoncd/results/contents/OWNERS", func(rw http.ResponseWriter, r *http.Request) {
		json.NewEncoder(rw).Encode(map[string]string{
			"type":    "file",
			"content": "Codercat",
		})
	})
	pr := &github.PullRequest{
		Head: &github.PullRequestBranch{
			SHA: github.String("deadbeef"),
			Repo: &github.Repository{
				CloneURL: github.String("https://example.com/repo"),
			},
		},
		Base: &github.PullRequestBranch{
			SHA: github.String("deadbeef1"),
			Repo: &github.Repository{
				Owner: &github.User{
					Login: github.String("Codertocat"),
				},
				Name: github.String("Hello-World"),
			},
		},
	}
	mux.HandleFunc("/repos/tektoncd/results/pulls/1", func(rw http.ResponseWriter, r *http.Request) {
		json.NewEncoder(rw).Encode(pr)
	})

	f, err := ioutil.ReadFile("testdata/issue_comment.json")
	if err != nil {
		log.Fatal(err)
	}
	req := &v1alpha1.InterceptorRequest{
		Body: string(f),
		Header: map[string][]string{
			"X-Github-Event": {"issue_comment"},
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
				PullRequest: &pb.PullRequestConfig{
					Comment: &pb.PullRequestConfig_CommentConfig{},
				},
			},
			ok: true,
		},
		{
			name: "all comments",
			cfg: &pb.Config{
				PullRequest: &pb.PullRequestConfig{
					Comment: &pb.PullRequestConfig_CommentConfig{
						Match: ".*",
					},
				},
			},
			ok: true,
		},
		{
			name: "non-existant OWNERS",
			cfg: &pb.Config{
				PullRequest: &pb.PullRequestConfig{
					Comment: &pb.PullRequestConfig_CommentConfig{
						Approvers: &pb.File{
							Path: "doesnotexist",
						},
					},
				},
			},
			ok: false,
		},
		{
			name: "non-matching comment",
			cfg: &pb.Config{
				PullRequest: &pb.PullRequestConfig{
					Comment: &pb.PullRequestConfig_CommentConfig{
						Match: "/not-ok-to-test",
					},
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
						URL:      "https://example.com/repo",
						Revision: "deadbeef",
					},
					"github": bindings.GitHub{
						Owner: "Codertocat",
						Repo:  "Hello-World",
					},
					"pull_request": pr,
				},
			}
			if diff := cmp.Diff(want, resp); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(fmt.Errorf("error parsing URL %s: %v", s, err))
	}
	return u
}
