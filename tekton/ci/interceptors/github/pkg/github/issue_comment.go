package github

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-github/v34/github"
	"github.com/tektoncd/plumbing/tekton/ci/interceptors/github/pkg/github/bindings"
	pb "github.com/tektoncd/plumbing/tekton/ci/interceptors/github/pkg/proto/v1alpha1/config_go_proto"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"google.golang.org/grpc/codes"
)

var (
	_ = Interceptor(&IssueComment{})
)

type IssueComment struct{}

func (c *IssueComment) Execute(ctx context.Context, client *github.Client, cfg *pb.Config, req *v1alpha1.InterceptorRequest) (*v1alpha1.InterceptorResponse, error) {
	event := new(github.IssueCommentEvent)
	if err := json.Unmarshal([]byte(req.Body), event); err != nil {
		return nil, Errorf(codes.InvalidArgument, err.Error())
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
	if !containsOwner(owners, commentAuthor) {
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

func containsOwner(content, owner string) bool {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		fmt.Println(scanner.Text(), owner)
		if scanner.Text() == owner {
			return true
		}
	}
	return false
}
