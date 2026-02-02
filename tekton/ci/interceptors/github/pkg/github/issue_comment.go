package github

import (
	"context"
	"encoding/json"
	"regexp"

	"github.com/google/go-github/v34/github"
	"github.com/tektoncd/plumbing/tekton/ci/interceptors/github/pkg/github/bindings"
	pb "github.com/tektoncd/plumbing/tekton/ci/interceptors/github/pkg/proto/v1alpha1/config_go_proto"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"google.golang.org/grpc/codes"
	"sigs.k8s.io/yaml"
)

var (
	_ = Interceptor(&IssueComment{})
)

type IssueComment struct{}

func (c *IssueComment) Execute(ctx context.Context, client *github.Client, cfg *pb.Config, req *v1alpha1.InterceptorRequest) (*v1alpha1.InterceptorResponse, error) {
	event := new(github.IssueCommentEvent)
	if err := json.Unmarshal([]byte(req.Body), event); err != nil {
		return nil, Errorf(codes.InvalidArgument, "%s", err.Error())
	}

	if event.GetAction() != "created" {
		return nil, Errorf(codes.Unimplemented, "unsupported action")
	}

	commentCfg := cfg.GetPullRequest().GetComment()
	if commentCfg == nil {
		// No approver config - take no action (should be covered by PR handler)
		return nil, Error(codes.FailedPrecondition, "comment config not enabled")
	}

	// Check if comment matches.
	match := commentCfg.GetMatch()
	if match == "" {
		match = "/ok-to-test"
	}
	re, err := regexp.Compile(match)
	if err != nil {
		return nil, Error(codes.FailedPrecondition, "invalid match keyphrase")
	}
	if !re.MatchString(event.GetComment().GetBody()) {
		return nil, Error(codes.FailedPrecondition, "comment does not match keyphrase")
	}

	// See if the comment came from an approved user. We do this after the
	// comment match to save an API call if we can.
	approverCfg := commentCfg.GetApprovers()
	path := approverCfg.GetPath()
	if path == "" {
		path = "OWNERS"
	}
	org := event.GetRepo().GetOwner().GetLogin()
	repo := event.GetRepo().GetName()
	number := event.GetIssue().GetNumber()
	commentAuthor := event.GetComment().GetUser().GetLogin()

	fc, _, _, err := client.Repositories.GetContents(ctx, org, repo, path, &github.RepositoryContentGetOptions{Ref: approverCfg.GetRevision()})
	if err != nil {
		return nil, err
	}
	owners, err := fc.GetContent()
	if err != nil {
		return nil, err
	}
	ok, err := containsOwner(owners, commentAuthor)
	if err != nil {
		return nil, Errorf(codes.InvalidArgument, "unable to read OWNERS file")
	}
	if !ok {
		return nil, Errorf(codes.PermissionDenied, "user not allowed to approve trigger")
	}

	pr, _, err := client.PullRequests.Get(ctx, org, repo, number)
	if err != nil {
		return nil, err
	}

	// Populate response w/ PR.
	return &v1alpha1.InterceptorResponse{
		Continue: true,
		Extensions: map[string]interface{}{
			"pull_request": pr,
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

// config is a fork of the prow OWNERS config file.
// See https://pkg.go.dev/k8s.io/test-infra/prow/repoowners#Config
type config struct {
	Approvers []string `json:"approvers,omitempty"`
	Reviewers []string `json:"reviewers,omitempty"`
}

// loadSimpleConfig loads SimpleConfig from bytes `b`
func loadConfig(b []byte) (config, error) {
	simple := new(config)
	err := yaml.Unmarshal(b, simple)
	return *simple, err
}

func containsOwner(content, owner string) (bool, error) {
	cfg, err := loadConfig([]byte(content))
	if err != nil {
		return false, err
	}

	for _, o := range cfg.Approvers {
		if owner == o {
			return true, nil
		}
	}
	for _, o := range cfg.Reviewers {
		if owner == o {
			return true, nil
		}
	}
	return false, nil
}
