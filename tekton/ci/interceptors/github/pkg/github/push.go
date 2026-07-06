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
	_ = Interceptor(&Push{})
)

type Push struct{}

func (c *Push) Execute(ctx context.Context, client *github.Client, cfg *pb.Config, req *v1alpha1.InterceptorRequest) (*v1alpha1.InterceptorResponse, error) {
	event := new(github.PushEvent)
	if err := json.Unmarshal([]byte(req.Body), event); err != nil {
		return nil, Error(codes.InvalidArgument, err.Error())
	}

	if cfg.GetPush() == nil {
		return nil, Error(codes.FailedPrecondition, "trigger not configured for push")
	}

	if event.GetAfter() == "0000000000000000000000000000000000000000" {
		return nil, Error(codes.FailedPrecondition, "ref was deleted - nothing to do")
	}

	patterns := cfg.GetPush().GetRef()
	if patterns == nil {
		// Default to all branches and tags.
		patterns = []string{"refs/heads/*", "refs/tags/*"}
	}
	for _, pattern := range patterns {
		g, err := glob.Compile(pattern)
		if err != nil {
			continue
		}
		if g.Match(event.GetRef()) {
			return &v1alpha1.InterceptorResponse{
				Continue: true,
				Extensions: map[string]interface{}{
					"git": bindings.Git{
						URL:      event.GetRepo().GetCloneURL(),
						Revision: event.GetAfter(),
					},
					"github": bindings.GitHub{
						Owner:        event.GetRepo().GetOwner().GetLogin(),
						Repo:         event.GetRepo().GetName(),
						Installation: event.GetInstallation().GetID(),
					},
				},
			}, nil
		}
	}
	return nil, Error(codes.FailedPrecondition, "did not find matching ref pattern")
}
