package github

import (
	"context"
	"encoding/json"

	"github.com/gobwas/glob"
	"github.com/google/go-github/v34/github"
	"github.com/tektoncd/plumbing/tekton/ci/interceptors/github/pkg/github/bindings"
	pb "github.com/tektoncd/plumbing/tekton/ci/interceptors/github/pkg/proto/v1alpha1/config_go_proto"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"google.golang.org/grpc/codes"
)

var (
	_ = Interceptor(&PullRequest{})

	actions = map[string]bool{
		"opened":       true,
		"synchronized": true,
	}
)

type PullRequest struct{}

func (c *PullRequest) Execute(ctx context.Context, client *github.Client, cfg *pb.Config, req *v1alpha1.InterceptorRequest) (*v1alpha1.InterceptorResponse, error) {
	event := new(github.PullRequestEvent)
	if err := json.Unmarshal([]byte(req.Body), event); err != nil {
		return nil, Errorf(codes.InvalidArgument, "%s", err.Error())
	}

	if cfg.GetPullRequest() == nil {
		return nil, Error(codes.FailedPrecondition, "trigger not configured for pull_request")
	}

	if _, ok := actions[event.GetAction()]; !ok {
		return nil, Error(codes.Unimplemented, "unsupported action")
	}

	prCfg := cfg.GetPullRequest()
	if prCfg.GetComment() != nil {
		return nil, Error(codes.FailedPrecondition, "waiting for authorized approval")
	}

	patterns := prCfg.GetBranch()
	if patterns == nil {
		// Default to all branches
		patterns = []string{"**"}
	}
	for _, pattern := range patterns {
		g, err := glob.Compile(pattern)
		if err != nil {
			continue
		}
		pr := event.GetPullRequest()
		if g.Match(pr.GetBase().GetRef()) {
			return &v1alpha1.InterceptorResponse{
				Continue: true,
				Extensions: map[string]interface{}{
					"git": bindings.Git{
						URL:      pr.GetHead().GetRepo().GetCloneURL(),
						Revision: pr.GetHead().GetSHA(),
					},
					"github": bindings.GitHub{
						Owner:        pr.GetBase().GetRepo().GetOwner().GetLogin(),
						Repo:         pr.GetBase().GetRepo().GetName(),
						Installation: event.GetInstallation().GetID(),
					},
				},
			}, nil
		}
	}
	return &v1alpha1.InterceptorResponse{
		Continue: false,
		Status: v1alpha1.Status{
			Code:    codes.FailedPrecondition,
			Message: "did not find matching branch pattern",
		},
	}, nil
}
