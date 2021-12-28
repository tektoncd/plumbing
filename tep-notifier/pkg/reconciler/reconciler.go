package reconciler

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v41/github"
	"github.com/pkg/errors"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"golang.org/x/oauth2"
	"knative.dev/pkg/logging"
	kreconciler "knative.dev/pkg/reconciler"
)

const (
	// TEPCommentAndStatusRegexFmt is the format used for adding the metadata for TEP and status to the comment.
	TEPCommentAndStatusRegexFmt = "<!-- TEP update: TEP-%s status: %s -->\n"

	// CommunityBlobBaseURL is the base URL for links to blobs in github.com/tektoncd/community/teps
	CommunityBlobBaseURL = "https://github.com/tektoncd/community/blob/main/teps/"

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

	// ProposedStatus is the "proposed" TEP status.
	ProposedStatus TEPStatus = "proposed"
	// ImplementableStatus is the "implementable" status.
	ImplementableStatus TEPStatus = "implementable"
	// ImplementingStatus is the "implementing" status.
	ImplementingStatus TEPStatus = "implementing"
	// ImplementedStatus is the "implemented" status.
	ImplementedStatus TEPStatus = "implemented"
	// WithdrawnStatus is the "withdrawn" status.
	WithdrawnStatus TEPStatus = "withdrawn"
	// ReplacedStatus is the "replaced" status.
	ReplacedStatus TEPStatus = "replaced"

	// BotUser is the user we will be making comments with
	BotUser = "tekton-robot"

	// ToImplementedCommentHeader is the header for PR comments reminding to update a given TEP from the
	// `implementing` status to the `implemented` status.
	ToImplementedCommentHeader = "This merged pull request appears to be referencing one or more [Tekton Enhancement Proposals](https://github.com/tektoncd/community/tree/main/teps#readme) " +
		"which are currently in the `implementing` state. If this PR finished the work towards implementing these TEPs, " +
		"please update their state(s) to `implemented`.\n" +
		"<!-- TEP Notifier: implemented -->\n" +
		"TEPs:\n"
	// ToImplementingCommentHeader is the header for PR comments reminding to update a given TEP from the
	// `proposed` or `implementable` status to the `implementing` status.
	ToImplementingCommentHeader = "This pull request appears to be referencing one or more [Tekton Enhancement Proposals](https://github.com/tektoncd/community/tree/main/teps#readme) " +
		"which are currently in the `proposed` or `implementable` states. If this PR contains " +
		"work towards implementing these TEPs, please update their state(s) to `implementing`.\n" +
		"<!-- TEP Notifier: implementing -->\n" +
		"TEPs:\n"

	// ActionParamName is the param name that will hold the pull request webhook action
	ActionParamName = "pullRequestAction"
	// PRNumberParamName is the param name that will hold the pull request number
	PRNumberParamName = "pullRequestNumber"
	// PRTitleParamName is the param name that will hold the pull request title
	PRTitleParamName = "pullRequestTitle"
	// PRBodyParamName is the param name that will hold the pull request body
	PRBodyParamName = "pullRequestBody"
	// PackageParamName is the param name that will hold the pull request's repository owner/name
	PackageParamName = "package"
	// PRIsMergedParamName is the param name that will hold whether the pull request is merged
	PRIsMergedParamName = "pullRequestIsMerged"

	closedAction = "closed"
	openedAction = "opened"
	editedAction = "edited"
)

var (
	// TEPCommentAndStatusRegex is used to detect whether a comment already is being used for the reminder message for
	// a particular TEP and status.
	TEPCommentAndStatusRegex = regexp.MustCompile(`<!-- TEP update: TEP-(\d+) status: (\w+) -->`)
	// NotifierActionRegex is used to detect whether a comment is referring to transitioning to implementing or to implemented.
	NotifierActionRegex = regexp.MustCompile(`<!-- TEP Notifier Action: (\w+) -->`)
	// TEPRegex is used to parse out "TEP-1234" from PR bodies and titles.
	TEPRegex = regexp.MustCompile(`TEP-(\d+)`)
	// TEPURLRegex is used to parse out TEP URLs, like https://github.com/tektoncd/community/blob/main/teps/0002-custom-tasks.md, from
	// PR bodies. We ignore the branch when looking for matches.
	TEPURLRegex = regexp.MustCompile(`https://github\.com/tektoncd/community/blob/.*?/teps/(\d+)-.*?\.md`)
	// TEPsInReadme matches rows in the table in https://github.com/tektoncd/community/blob/main/teps/README.md
	TEPsInReadme = regexp.MustCompile(`\|\[TEP-(\d+)]\((.*?\.md)\) \| (.*?) \| (.*?) \| (\d\d\d\d-\d\d-\d\d) \|`)

	// TEPStatuses is a list of all valid TEP statuses
	TEPStatuses = []TEPStatus{
		ProposedStatus,
		ImplementableStatus,
		ImplementingStatus,
		ImplementedStatus,
		WithdrawnStatus,
		ReplacedStatus,
	}
)

// TEPStatus is a valid TEP status
type TEPStatus string

// TEPCommentInfo stores the PR comment ID for a comment the notifier made about a TEP, the TEP's status when the
// comment was made, and the TEP ID
type TEPCommentInfo struct {
	CommentID     int64
	TEPs          []string
	ToImplemented bool
}

// TEPInfo stores information about a TEP parsed from https://github.com/tektoncd/community/blob/main/teps/README.md
type TEPInfo struct {
	ID           string
	Title        string
	Status       TEPStatus
	Filename     string
	LastModified time.Time
}

// Reconciler handles the actual parsing of PR title and description for TEP identifiers, and adds comments to the PR
// as appropriate.
type Reconciler struct {
	GHClient *github.Client
}

// NewReconciler creates a new Reconciler instance configured with a GitHub client
func NewReconciler(ctx context.Context, ghToken string) *Reconciler {
	// Allow anonymous for testing purposes
	if ghToken == "" {
		return &Reconciler{
			GHClient: github.NewClient(nil),
		}
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghToken})
	tc := oauth2.NewClient(ctx, ts)

	return &Reconciler{
		GHClient: github.NewClient(tc),
	}
}

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, run *v1alpha1.Run) kreconciler.Event {
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling %s/%s", run.Namespace, run.Name)

	// Ignore completed waits.
	if run.IsDone() {
		logger.Info("Run is finished, done reconciling")
		return nil
	}

	if run.Spec.Ref == nil ||
		run.Spec.Ref.APIVersion != v1beta1.SchemeGroupVersion.String() || run.Spec.Ref.Kind != Kind {
		logger.Warnf("Should not have been notified about Run %s/%s; will do nothing", run.Namespace, run.Name)
		return nil
	}

	prActionParam := run.Spec.GetParam(ActionParamName)
	if prActionParam == nil || prActionParam.Value.StringVal == "" {
		run.Status.MarkRunFailed("MissingPullRequestAction", "The %s param was not passed", ActionParamName)
		return nil
	}
	prAction := prActionParam.Value.StringVal

	// Short-circuit for PR actions other than `closed`, `edited`, or `opened`.
	if prAction != closedAction && prAction != editedAction && prAction != openedAction {
		logger.Warnf("Ignoring PR action %s; will do nothing", prAction)
		return nil
	}

	prNumberParam := run.Spec.GetParam(PRNumberParamName)
	if prNumberParam == nil || prNumberParam.Value.StringVal == "" {
		run.Status.MarkRunFailed("MissingPullRequestNumber", "The %s param was not passed", PRNumberParamName)
		return nil
	}
	prNumber, err := strconv.Atoi(prNumberParam.Value.StringVal)
	if err != nil {
		run.Status.MarkRunFailed("InvalidPullRequestNumber", "%s is not a valid value for the %s param", prNumberParam.Value.StringVal, PRNumberParamName)
		return nil
	}

	prTitleParam := run.Spec.GetParam(PRTitleParamName)
	if prTitleParam == nil || prTitleParam.Value.StringVal == "" {
		run.Status.MarkRunFailed("MissingPullRequestTitle", "The %s param was not passed", PRTitleParamName)
		return nil
	}
	prTitle := prTitleParam.Value.StringVal

	prBodyParam := run.Spec.GetParam(PRBodyParamName)
	if prBodyParam == nil || prBodyParam.Value.StringVal == "" {
		run.Status.MarkRunFailed("MissingPullRequestBody", "The %s param was not passed", PRBodyParamName)
		return nil
	}
	prBody := prBodyParam.Value.StringVal

	orgAndRepoParam := run.Spec.GetParam(PackageParamName)
	if orgAndRepoParam == nil || orgAndRepoParam.Value.StringVal == "" {
		run.Status.MarkRunFailed("MissingPackage", "The %s param was not passed", PackageParamName)
		return nil
	}
	splitOrgAndRepo := strings.Split(orgAndRepoParam.Value.StringVal, "/")
	if len(splitOrgAndRepo) < 2 {
		run.Status.MarkRunFailed("InvalidPackage", "The %s param value %s does not contain an owner and a repository seperated by '/'",
			PackageParamName, orgAndRepoParam.Value.StringVal)
		return nil
	}
	repo := splitOrgAndRepo[1]

	isMergedParam := run.Spec.GetParam(PRIsMergedParamName)
	if isMergedParam == nil || isMergedParam.Value.StringVal == "" {
		run.Status.MarkRunFailed("MissingPullRequestIsMerged", "The %s param was not passed", PRIsMergedParamName)
		return nil
	}
	isMerged, err := strconv.ParseBool(isMergedParam.Value.StringVal)
	if err != nil {
		run.Status.MarkRunFailed("InvalidPullRequestIsMerged", "%s is not a valid value for the %s param", isMergedParam.Value.StringVal, PRIsMergedParamName)
		return nil
	}

	// Short-circuit if the action is "closed" but the PR is not merged.
	if prAction == closedAction && !isMerged {
		logger.Warn("Ignoring closed PR because the PR was not merged; will do nothing")
		return nil
	}

	if len(run.Spec.Params) > 6 {
		var found []string
		for _, p := range run.Spec.Params {
			if p.Name == ActionParamName ||
				p.Name == PRNumberParamName ||
				p.Name == PRTitleParamName ||
				p.Name == PRBodyParamName ||
				p.Name == PackageParamName ||
				p.Name == PRIsMergedParamName {
				continue
			}
			found = append(found, p.Name)
		}
		run.Status.MarkRunFailed("UnexpectedParams", "Found unexpected params: %v", found)
		return nil
	}

	tepsOnPR, err := r.TEPsInPR(ctx, prTitle, prBody)
	if err != nil {
		run.Status.MarkRunFailed("LoadingPRTEPs", "Failure loading TEPs for %s/%s PR #%d: %s", TEPsOwner, repo, prNumber, err)
		return nil
	}

	var tepsForComment []TEPInfo
	var commentFunc func([]TEPInfo) string
	var transitionStates []TEPStatus

	if prAction == closedAction {
		commentFunc = PRMergedComment
		transitionStates = append(transitionStates, ImplementingStatus)
	} else {
		commentFunc = PROpenedComment
		transitionStates = append(transitionStates, ProposedStatus, ImplementableStatus)
	}

	for _, tep := range tepsOnPR {
		shouldInclude := false
		for _, ts := range transitionStates {
			if ts == tep.Status {
				shouldInclude = true
			}
		}

		if shouldInclude {
			tepsForComment = append(tepsForComment, tep)
		}
	}

	if len(tepsForComment) > 0 {
		commentBody := commentFunc(tepsForComment)

		existingComments, err := r.TEPCommentsOnPR(ctx, repo, prNumber)
		if err != nil {
			run.Status.MarkRunFailed("CheckingPRComments", "Failure checking for TEP comments for %s/%s PR #%d: %s", TEPsOwner, repo, prNumber, err)
			return nil
		}

		var commentToUpdate *TEPCommentInfo
		for _, cmt := range existingComments {
			if prAction == closedAction && cmt.ToImplemented {
				commentToUpdate = &cmt
				break
			} else if prAction == editedAction || prAction == openedAction {
				commentToUpdate = &cmt
			}
		}

		if commentToUpdate != nil {
			needsUpdate := false
			for _, newTep := range tepsForComment {
				found := false
				for _, id := range commentToUpdate.TEPs {
					if id == newTep.ID {
						found = true
						break
					}
				}
				if !found {
					needsUpdate = true
					break
				}
			}

			if needsUpdate {
				err = r.EditComment(ctx, repo, commentToUpdate.CommentID, commentBody)
				if err != nil {
					run.Status.MarkRunFailed("UpdatingPRComment", "Failure updating existing comment %d for %s/%s PR #%d: %s",
						commentToUpdate.CommentID, TEPsOwner, repo, prNumber, err)
					return nil
				}
				run.Status.MarkRunSucceeded("CommentUpdated", "Existing comment %d for %s/%s PR #%d updated",
					commentToUpdate.CommentID, TEPsOwner, repo, prNumber)
				return nil
			}
		} else {
			err = r.AddComment(ctx, repo, prNumber, commentBody)
			if err != nil {
				run.Status.MarkRunFailed("AddingPRComment", "Failure adding new comment for %s/%s PR #%d: %s",
					TEPsOwner, repo, prNumber, err)
				return nil
			}
			run.Status.MarkRunSucceeded("CommentAdded", "Comment for %s/%s PR #%d",
				TEPsOwner, repo, prNumber)
			return nil
		}
	}

	// If we got here, then we didn't need to do anything.
	logger.Warnf("No TEPs found in title or body for %s/%s PR #%d; nothing to do", TEPsOwner, repo, prNumber)
	return nil
}

// TEPCommentsOnPR finds all comments on the given PR made by this notifier
func (r *Reconciler) TEPCommentsOnPR(ctx context.Context, repo string, prNumber int) ([]TEPCommentInfo, error) {
	listOpts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 20,
		},
	}

	var tepComments []TEPCommentInfo

	for {
		comments, resp, err := r.GHClient.Issues.ListComments(ctx, TEPsOwner, repo, prNumber, listOpts)
		if err != nil {
			return nil, errors.Wrapf(err, "getting comments for PR #%d in %s/%s", prNumber, TEPsOwner, repo)
		}

		for _, c := range comments {
			if c.ID != nil && c.Body != nil && c.User != nil && c.User.Login != nil && *c.User.Login == BotUser {
				tepsOnComment, toImplemented := GetTEPCommentDetails(*c.Body)

				if len(tepsOnComment) > 0 {
					tci := TEPCommentInfo{
						CommentID:     *c.ID,
						TEPs:          []string{},
						ToImplemented: toImplemented,
					}
					for tID, tStatus := range tepsOnComment {
						if !isValidStatus(tStatus) {
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

// GetTEPsFromReadme fetches https://github.com/tektoncd/community/blob/main/teps/README.md, parses out the TEPs from
// the table in that file, and returns a map of TEP IDs (i.e., "1234") to TEPInfo for that TEP.
func (r *Reconciler) GetTEPsFromReadme(ctx context.Context) (map[string]TEPInfo, error) {
	fc, _, _, err := r.GHClient.Repositories.GetContents(ctx, TEPsOwner, TEPsRepo, filepath.Join(TEPsDirectory, TEPsReadmeFile), &github.RepositoryContentGetOptions{
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

	teps, err := ExtractTEPsFromReadme(readmeStr)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing content of https://github.com/%s/%s/blob/%s/%s/%s", TEPsOwner, TEPsRepo,
			TEPsBranch, TEPsDirectory, TEPsReadmeFile)
	}

	return teps, nil
}

// TEPsInPR returns all TEPs referenced in the PR title or body in a map with the TEP ID as key and information about the
// TEP as value.
func (r *Reconciler) TEPsInPR(ctx context.Context, prTitle, prBody string) (map[string]TEPInfo, error) {
	tepsWithInfo := make(map[string]TEPInfo)

	tepIDs := GetTEPIDsFromPR(prTitle, prBody)

	if len(tepIDs) == 0 {
		return tepsWithInfo, nil
	}

	allTEPs, err := r.GetTEPsFromReadme(ctx)
	if err != nil {
		return nil, err
	}

	for _, tID := range tepIDs {
		tepsWithInfo[tID] = allTEPs[tID]
	}

	return tepsWithInfo, nil
}

// AddComment adds a new comment to the PR
func (r *Reconciler) AddComment(ctx context.Context, repo string, prNumber int, body string) error {
	input := &github.IssueComment{
		Body: github.String(body),
	}
	_, _, err := r.GHClient.Issues.CreateComment(ctx, TEPsOwner, repo, prNumber, input)
	return err
}

// EditComment updates an existing comment on the PR
func (r *Reconciler) EditComment(ctx context.Context, repo string, commentID int64, body string) error {
	input := &github.IssueComment{
		Body: github.String(body),
	}
	_, _, err := r.GHClient.Issues.EditComment(ctx, TEPsOwner, repo, commentID, input)
	return err
}

// GetTEPCommentDetails looks at a PR comment and extracts any TEP IDs and their status in the comment and whether this
// comment is for transitioning TEPs to `implementing` or to `implemented`
func GetTEPCommentDetails(comment string) (map[string]string, bool) {
	teps := make(map[string]string)

	toImplemented := false
	notifierMatch := NotifierActionRegex.FindStringSubmatch(comment)
	if len(notifierMatch) > 1 && notifierMatch[1] == string(ImplementedStatus) {
		toImplemented = true
	}

	for _, m := range TEPCommentAndStatusRegex.FindAllStringSubmatch(comment, -1) {
		if len(m) > 2 {
			teps[m[1]] = m[2]
		}
	}

	return teps, toImplemented
}

// GetTEPIDsFromPR extracts all TEP IDs and URLs in the given PR title and body
func GetTEPIDsFromPR(prTitle, prBody string) []string {
	var tepIDs []string

	// Find "TEP-1234" in PR title
	for _, m := range TEPRegex.FindAllStringSubmatch(prTitle, -1) {
		if len(m) > 1 {
			tepIDs = append(tepIDs, m[1])
		}
	}
	// Find TEP URLs in title
	for _, m := range TEPURLRegex.FindAllStringSubmatch(prTitle, -1) {
		if len(m) > 1 {
			tepIDs = append(tepIDs, m[1])
		}
	}

	// Find "TEP-1234" in PR body
	for _, m := range TEPRegex.FindAllStringSubmatch(prBody, -1) {
		if len(m) > 1 {
			tepIDs = append(tepIDs, m[1])
		}
	}
	// Find TEP URLs in body
	for _, m := range TEPURLRegex.FindAllStringSubmatch(prBody, -1) {
		if len(m) > 1 {
			tepIDs = append(tepIDs, m[1])
		}
	}

	return tepIDs
}

// ExtractTEPsFromReadme takes the body of https://github.com/tektoncd/community/blob/main/teps/README.md and extracts a
// map of all TEP IDs (i.e., "1234" for TEP-1234) and their statuses.
func ExtractTEPsFromReadme(readmeBody string) (map[string]TEPInfo, error) {
	teps := make(map[string]TEPInfo)

	for _, m := range TEPsInReadme.FindAllStringSubmatch(readmeBody, -1) {
		if len(m) > 5 {
			// TODO(abayer): For some reason, I can't ever get time.Parse to handle a format of "2021-12-20" so let's just pad it and do it as RFC3339.
			lastMod, err := time.Parse(time.RFC3339, fmt.Sprintf("%sT00:00:00Z", m[5]))
			if err != nil {
				return nil, err
			}

			if !isValidStatus(m[4]) {
				return nil, fmt.Errorf("%s is not a valid status", m[4])
			}
			teps[m[1]] = TEPInfo{
				ID:           m[1],
				Title:        m[3],
				Status:       TEPStatus(m[4]),
				Filename:     m[2],
				LastModified: lastMod,
			}
		}
	}

	return teps, nil
}

// GetTEPsWithStatus filters a map of TEP ID to TEPInfo for all TEPs in a given status
func GetTEPsWithStatus(input map[string]TEPInfo, desiredStatus TEPStatus) map[string]TEPInfo {
	teps := make(map[string]TEPInfo)

	for k, v := range input {
		if v.Status == desiredStatus {
			teps[k] = v
		}
	}

	return teps
}

// PROpenedComment returns the appropriate GitHub Markdown-formatted content for the PR comment on opened/edited PRs
// referencing TEPs in `proposed` or `implementable` states.
func PROpenedComment(teps []TEPInfo) string {
	return generatePRComment(ToImplementingCommentHeader, teps)
}

// PRMergedComment returns the appropriate GitHub Markdown-formatted content for the PR comment on merged PRs
// referencing TEPs in the `implementing` state.
func PRMergedComment(teps []TEPInfo) string {
	return generatePRComment(ToImplementedCommentHeader, teps)
}

func generatePRComment(header string, teps []TEPInfo) string {
	var commentLines []string
	var listLines []string
	var metadataLines []string

	for _, tep := range teps {
		listLines = append(listLines, fmt.Sprintf(" * [TEP-%s (%s)](%s%s), current status: `%s`\n", tep.ID, tep.Title, CommunityBlobBaseURL, tep.Filename, tep.Status))
		metadataLines = append(metadataLines, fmt.Sprintf(TEPCommentAndStatusRegexFmt, tep.ID, tep.Status))
	}

	commentLines = append(commentLines, header)
	commentLines = append(commentLines, listLines...)
	commentLines = append(commentLines, "\n") // Newline between the list of TEPs and the metadata
	commentLines = append(commentLines, metadataLines...)

	return strings.Join(commentLines, "") // TODO(abayer): Probably should get rid of trailing newlines on the header and the list/metadata lines and just join on "\n" here.
}

func isValidStatus(s string) bool {
	for _, status := range TEPStatuses {
		if s == string(status) {
			return true
		}
	}

	return false
}
