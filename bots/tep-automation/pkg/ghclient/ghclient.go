package ghclient

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/go-github/v41/github"
	"github.com/pkg/errors"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/tep"
	"golang.org/x/oauth2"
)

const (
	// TEPsOwner is the GitHub owner for the repo containing the TEPs.
	TEPsOwner = "tektoncd"
	// TEPsRepo is the GitHub repository containing the TEPs, under TEPsOwner.
	TEPsRepo = "community"
	// TEPsDirectory is the directory containing the TEPs, within TEPsRepo.
	TEPsDirectory = "teps"
	// TEPsBranch is the branch in TEPsRepo we will look at.
	TEPsBranch = "main"
	// TEPsReadmeFile is the filename to find the list of TEPs and statuses in.
	TEPsReadmeFile = "README.md"

	// BotUser is the user we will be making comments with
	BotUser = "tekton-robot"

	// TrackingIssueLabel is a label put on tracking issues to identify them as such.
	TrackingIssueLabel = "tep-tracking"

	// TrackingIssueTitleFmt is used to construct TEP tracking issue titles
	TrackingIssueTitleFmt = "TEP-%s Tracking Issue"
)

var (
	tepPRFilenameRegex      = regexp.MustCompile(`^teps/(\d+)-.*\.md$`)
	trackingIssueTitleRegex = regexp.MustCompile(`^TEP-(\d+) Tracking Issue$`)
)

// TEPGHClient is a wrapper around github.Client that exposes functions needed elsewhere in this tooling.
type TEPGHClient struct {
	client *github.Client
}

// NewTEPGHClientFromToken creates a new client using the given token
func NewTEPGHClientFromToken(ctx context.Context, ghToken string) *TEPGHClient {
	// Allow anonymous for testing purposes
	if ghToken == "" {
		return &TEPGHClient{
			client: github.NewClient(nil),
		}
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghToken})
	tc := oauth2.NewClient(ctx, ts)

	return &TEPGHClient{
		client: github.NewClient(tc),
	}
}

// NewTEPGHClient creates a new client using the given *github.Client
func NewTEPGHClient(ghClient *github.Client) *TEPGHClient {
	return &TEPGHClient{client: ghClient}
}

// GetTEPsFromReadme fetches https://github.com/tektoncd/community/blob/main/teps/README.md, parses out the TEPs from
// the table in that file, and returns a map of TEP IDs (i.e., "1234") to TEPInfo for that TEP.
func (tgc *TEPGHClient) GetTEPsFromReadme(ctx context.Context) (map[string]tep.TEPInfo, error) {
	fc, _, _, err := tgc.client.Repositories.GetContents(ctx, TEPsOwner, TEPsRepo, filepath.Join(TEPsDirectory, TEPsReadmeFile), &github.RepositoryContentGetOptions{
		Ref: TEPsBranch,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "fetching contents of https://github.com/%s/%s/blob/%s/%s/%s", TEPsOwner, TEPsRepo,
			TEPsBranch, TEPsDirectory, TEPsReadmeFile)
	}

	readmeStr, err := fc.GetContent()
	if err != nil {
		return nil, errors.Wrapf(err, "reading content of https://github.com/%s/%s/blob/%s/%s/%s", TEPsOwner, TEPsRepo,
			TEPsBranch, TEPsDirectory, TEPsReadmeFile)
	}

	teps, err := tep.ExtractTEPsFromReadme(readmeStr)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing content of https://github.com/%s/%s/blob/%s/%s/%s", TEPsOwner, TEPsRepo,
			TEPsBranch, TEPsDirectory, TEPsReadmeFile)
	}

	return teps, nil
}

// TEPCommentsOnPR finds all comments on the given PR made by this notifier
func (tgc *TEPGHClient) TEPCommentsOnPR(ctx context.Context, repo string, prNumber int) ([]tep.CommentInfo, error) {
	listOpts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 20,
		},
	}

	var tepComments []tep.CommentInfo

	for {
		comments, resp, err := tgc.client.Issues.ListComments(ctx, TEPsOwner, repo, prNumber, listOpts)
		if err != nil {
			return nil, errors.Wrapf(err, "getting comments for PR #%d in %s/%s", prNumber, TEPsOwner, repo)
		}

		for _, c := range comments {
			if c.ID != nil && c.Body != nil && c.User != nil && c.User.Login != nil && *c.User.Login == BotUser {
				tepsOnComment, toImplemented := tep.GetTEPCommentDetails(*c.Body)

				if len(tepsOnComment) > 0 {
					tci := tep.CommentInfo{
						CommentID:     *c.ID,
						TEPs:          []string{},
						ToImplemented: toImplemented,
					}
					for tID, tStatus := range tepsOnComment {
						if !tep.IsValidStatus(tStatus) {
							return nil, fmt.Errorf("metadata for TEP-%s has invalid status %s", tID, tStatus)
						}
						tci.TEPs = append(tci.TEPs, tID)
					}
					tepComments = append(tepComments, tci)
				}
			}
		}

		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	return tepComments, nil
}

// AddComment adds a new comment to the PR
func (tgc *TEPGHClient) AddComment(ctx context.Context, repo string, prNumber int, body string) error {
	input := &github.IssueComment{
		Body: github.String(body),
	}
	_, _, err := tgc.client.Issues.CreateComment(ctx, TEPsOwner, repo, prNumber, input)
	return err
}

// EditComment updates an existing comment on the PR
func (tgc *TEPGHClient) EditComment(ctx context.Context, repo string, commentID int64, body string) error {
	input := &github.IssueComment{
		Body: github.String(body),
	}
	_, _, err := tgc.client.Issues.EditComment(ctx, TEPsOwner, repo, commentID, input)
	return err
}

// CreateTrackingIssue creates a tracking issue for a TEP in the community repo
func (tgc *TEPGHClient) CreateTrackingIssue(ctx context.Context, id, comment string, authors []string, status tep.Status) error {
	labels := []string{
		TrackingIssueLabel,
		status.TrackingLabel(),
	}

	var assignees []string
	assignees = append(assignees, authors...)

	input := &github.IssueRequest{
		Title:     github.String(fmt.Sprintf(TrackingIssueTitleFmt, id)),
		Body:      github.String(comment),
		Labels:    &labels,
		Assignees: &assignees,
	}

	_, _, err := tgc.client.Issues.Create(ctx, TEPsOwner, TEPsRepo, input)
	return err
}

// UpdateTrackingIssue updates a tracking issue for a TEP in the community repo
func (tgc *TEPGHClient) UpdateTrackingIssue(ctx context.Context, issueNumber int, id, comment string, authors []string, status tep.Status) error {
	labels := []string{
		TrackingIssueLabel,
		status.TrackingLabel(),
	}

	var assignees []string
	assignees = append(assignees, authors...)

	input := &github.IssueRequest{
		Title:     github.String(fmt.Sprintf(TrackingIssueTitleFmt, id)),
		Body:      github.String(comment),
		Labels:    &labels,
		Assignees: &assignees,
	}

	_, _, err := tgc.client.Issues.Edit(ctx, TEPsOwner, TEPsRepo, issueNumber, input)
	return err
}

// CloseTrackingIssue adds a comment and closes a tracking issue for a TEP in the community repo
func (tgc *TEPGHClient) CloseTrackingIssue(ctx context.Context, issueNumber int, comment string) error {
	_, _, err := tgc.client.Issues.CreateComment(ctx, TEPsOwner, TEPsRepo, issueNumber, &github.IssueComment{Body: github.String(comment)})
	if err != nil {
		return err

	}

	input := &github.IssueRequest{
		State: github.String("closed"),
	}

	_, _, err = tgc.client.Issues.Edit(ctx, TEPsOwner, TEPsRepo, issueNumber, input)
	return err
}

// ExtractTEPInfoFromTEPPR checks the given PR for changes to TEP markdown files. If one isn't present, it returns nil,
// and if one or more is present, their metadata is parsed and returned.
func (tgc *TEPGHClient) ExtractTEPInfoFromTEPPR(ctx context.Context, prNumber int, prRef string) ([]tep.TEPInfo, error) {
	listOpts := &github.ListOptions{
		PerPage: 20,
	}

	tepFilenamesToIDs := make(map[string]string)

	for {
		files, resp, err := tgc.client.PullRequests.ListFiles(ctx, TEPsOwner, TEPsRepo, prNumber, listOpts)
		if err != nil {
			return nil, errors.Wrapf(err, "listing files in PR #%d in %s/%s", prNumber, TEPsOwner, TEPsRepo)
		}

		for _, f := range files {
			if f.Status != nil && f.Filename != nil && (*f.Status == "added" || *f.Status == "modified") {
				m := tepPRFilenameRegex.FindStringSubmatch(*f.Filename)
				if len(m) > 0 {
					tepFilenamesToIDs[*f.Filename] = m[1]
				}
			}
		}
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	if len(tepFilenamesToIDs) == 0 {
		return nil, nil
	}

	var teps []tep.TEPInfo

	for fn, tepID := range tepFilenamesToIDs {
		fc, _, _, err := tgc.client.Repositories.GetContents(ctx, TEPsOwner, TEPsRepo, fn, &github.RepositoryContentGetOptions{
			Ref: prRef,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "fetching contents of https://github.com/%s/%s/blob/%s/%s", TEPsOwner, TEPsRepo,
				prRef, fn)
		}

		mdStr, err := fc.GetContent()
		if err != nil {
			return nil, errors.Wrapf(err, "reading contents of https://github.com/%s/%s/blob/%s/%s", TEPsOwner, TEPsRepo,
				prRef, fn)
		}

		fnWithoutPrefix := strings.TrimPrefix(fn, "teps/")
		info, err := tep.TEPInfoFromMarkdown(tepID, fnWithoutPrefix, mdStr)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing contents of https://github.com/%s/%s/blob/%s/%s", TEPsOwner, TEPsRepo,
				prRef, fn)
		}

		teps = append(teps, info)
	}

	return teps, nil
}

// GetTrackingIssuesOptions allows configuring calls to GetTrackingIssues to filter on issue state, TEP status, or TEP ID
type GetTrackingIssuesOptions struct {
	IssueState string
	TEPStatus  tep.Status
	TEPID      string
}

// GetTrackingIssues returns issues in tektoncd/community with the "tep-tracking" label, optionally filtering by issue
// state, current known TEP status, or TEP ID.
func (tgc *TEPGHClient) GetTrackingIssues(ctx context.Context, opts *GetTrackingIssuesOptions) (map[string]*tep.TrackingIssue, error) {
	listOpts := &github.IssueListByRepoOptions{
		Labels: []string{TrackingIssueLabel},
		ListOptions: github.ListOptions{
			PerPage: 20,
		},
	}

	tepID := ""
	if opts != nil {
		if opts.IssueState != "" {
			listOpts.State = opts.IssueState
		}
		if opts.TEPStatus != "" {
			listOpts.Labels = append(listOpts.Labels, opts.TEPStatus.TrackingLabel())
		}
		tepID = opts.TEPID
	}

	foundIssues := make(map[string]*tep.TrackingIssue)

	for {
		issues, resp, err := tgc.client.Issues.ListByRepo(ctx, TEPsOwner, TEPsRepo, listOpts)
		if err != nil {
			return nil, errors.Wrapf(err, "listing tracking issues in %s/%s", TEPsOwner, TEPsRepo)
		}

		for _, iss := range issues {
			if iss.IsPullRequest() {
				continue
			}

			if tepID != "" {
				if iss.GetTitle() != fmt.Sprintf(TrackingIssueTitleFmt, tepID) {
					continue
				}
			}

			ti := &tep.TrackingIssue{
				IssueNumber: iss.GetNumber(),
				IssueState:  iss.GetState(),
				Assignees:   []string{},
			}

			if m := trackingIssueTitleRegex.FindStringSubmatch(iss.GetTitle()); len(m) > 0 {
				ti.TEPID = m[1]
			}

			for _, assignee := range iss.Assignees {
				ti.Assignees = append(ti.Assignees, assignee.GetLogin())
			}

			for _, label := range iss.Labels {
				if strings.HasPrefix(label.GetName(), tep.TrackingIssueStatusLabelPrefix) {
					tepStatus := tep.FromTrackingIssueLabel(label.GetName())
					if tepStatus == nil {
						return nil, fmt.Errorf("label %s is not a valid TEP status", label.GetName())
					}
					ti.TEPStatus = *tepStatus
				}
			}

			tepPRIDs, implPRs, err := tep.PRsForTrackingIssue(iss.GetBody())
			if err != nil {
				return nil, errors.Wrapf(err, "parsing TEP and implementation PRs from tracking issue body '%s'", iss.GetBody())
			}

			ti.TEPPRs = tepPRIDs
			ti.ImplementationPRs = implPRs

			foundIssues[ti.TEPID] = ti
		}
		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	return foundIssues, nil
}
