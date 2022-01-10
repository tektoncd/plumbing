package performers

import (
	"context"
	"fmt"
	"strings"

	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/ghclient"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/tep"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/logging"
	kreconciler "knative.dev/pkg/reconciler"
)

const (
	// TEPCommentAndStatusRegexFmt is the format used for adding the metadata for TEP and status to the comment.
	TEPCommentAndStatusRegexFmt = "<!-- TEP update: TEP-%s status: %s -->\n"

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
)

// PRNotifier handles the actual parsing of PR title and description for TEP identifiers, and adds comments to the PR
// as appropriate.
type PRNotifier struct {
	GHClient *ghclient.TEPGHClient
}

// NewPRNotifier creates a new PRNotifier instance configured with a GitHub client
func NewPRNotifier(ghClient *ghclient.TEPGHClient) *PRNotifier {
	return &PRNotifier{
		GHClient: ghClient,
	}
}

// NewPRNotifierAsPerformer calls NewPRNotifier and returns it as a Performer
func NewPRNotifierAsPerformer(ghClient *ghclient.TEPGHClient) Performer {
	return NewPRNotifier(ghClient)
}

// Perform checks if the PR needs to have a TEP-related commented added or updated, and if so, does so.
func (n *PRNotifier) Perform(ctx context.Context, opts *PerformerOptions) kreconciler.Event {
	logger := logging.FromContext(ctx).With(zap.String("performer", "PRNotifier"))
	logger.Debugf("Performing implementation PR notification (if needed) for %s/%s", opts.RunNamespace, opts.RunName)

	// Short-circuit for PR actions other than `closed`, `edited`, or `opened`.
	if opts.Action != closedAction && opts.Action != editedAction && opts.Action != openedAction {
		logger.Infof("Ignoring PR action %s; will do nothing", opts.Action)
		return nil
	}

	// Short-circuit if the action is "closed" but the PR is not merged.
	if opts.Action == closedAction && !opts.IsMerged {
		logger.Info("Ignoring closed PR because the PR was not merged; will do nothing")
		return nil
	}

	tepsOnPR, err := n.TEPsInPR(ctx, opts.Title, opts.Body)
	if err != nil {
		return kreconciler.NewEvent(corev1.EventTypeWarning, "LoadingPRTEPs", "Failure loading TEPs for %s/%s PR #%d: %s", ghclient.TEPsOwner, opts.Repo, opts.PRNumber, err)
	}

	var tepsForComment []tep.TEPInfo
	var commentFunc func([]tep.TEPInfo) string
	var transitionStates []tep.Status

	if opts.Action == closedAction {
		commentFunc = PRMergedComment
		transitionStates = append(transitionStates, tep.ImplementingStatus)
	} else {
		commentFunc = PROpenedComment
		transitionStates = append(transitionStates, tep.ProposedStatus, tep.ImplementableStatus)
	}

	for _, t := range tepsOnPR {
		shouldInclude := false
		for _, ts := range transitionStates {
			if ts == t.Status {
				shouldInclude = true
			}
		}

		if shouldInclude {
			tepsForComment = append(tepsForComment, t)
		}
	}

	if len(tepsForComment) > 0 {
		commentBody := commentFunc(tepsForComment)

		existingComments, err := n.GHClient.TEPCommentsOnPR(ctx, opts.Repo, opts.PRNumber)
		if err != nil {
			return kreconciler.NewEvent(corev1.EventTypeWarning, "CheckingPRComments", "Failure checking for TEP comments for %s/%s PR #%d: %s",
				ghclient.TEPsOwner, opts.Repo, opts.PRNumber, err)
		}

		var commentToUpdate *tep.CommentInfo
		for _, cmt := range existingComments {
			if opts.Action == closedAction && cmt.ToImplemented {
				commentToUpdate = &cmt
				break
			} else if opts.Action == editedAction || opts.Action == openedAction {
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
				err = n.GHClient.EditComment(ctx, opts.Repo, commentToUpdate.CommentID, commentBody)
				if err != nil {
					return kreconciler.NewEvent(corev1.EventTypeWarning, "UpdatingPRComment", "Failure updating existing comment %d for %s/%s PR #%d: %s",
						commentToUpdate.CommentID, ghclient.TEPsOwner, opts.Repo, opts.PRNumber, err)
				}
				return kreconciler.NewEvent(corev1.EventTypeNormal, "CommentUpdated", "Existing comment %d for %s/%s PR #%d updated",
					commentToUpdate.CommentID, ghclient.TEPsOwner, opts.Repo, opts.PRNumber)
			}
		} else {
			err = n.GHClient.AddComment(ctx, opts.Repo, opts.PRNumber, commentBody)
			if err != nil {
				return kreconciler.NewEvent(corev1.EventTypeWarning, "AddingPRComment", "Failure adding new comment for %s/%s PR #%d: %s",
					ghclient.TEPsOwner, opts.Repo, opts.PRNumber, err)
			}
			return kreconciler.NewEvent(corev1.EventTypeNormal, "CommentAdded", "Comment for %s/%s PR #%d",
				ghclient.TEPsOwner, opts.Repo, opts.PRNumber)
		}
	}

	// If we got here, then we didn't need to do anything.
	logger.Infof("No TEPs found in title or body for %s/%s PR #%d; nothing to do", ghclient.TEPsOwner, opts.Repo, opts.PRNumber)
	return nil
}

// TEPsInPR returns all TEPs referenced in the PR title or body in a map with the TEP ID as key and information about the
// TEP as value.
func (n *PRNotifier) TEPsInPR(ctx context.Context, prTitle, prBody string) (map[string]tep.TEPInfo, error) {
	tepsWithInfo := make(map[string]tep.TEPInfo)

	tepIDs := tep.GetTEPIDsFromPR(prTitle, prBody)

	if len(tepIDs) == 0 {
		return tepsWithInfo, nil
	}

	allTEPs, err := n.GHClient.GetTEPsFromReadme(ctx)
	if err != nil {
		return nil, err
	}

	for _, tID := range tepIDs {
		tepsWithInfo[tID] = allTEPs[tID]
	}

	return tepsWithInfo, nil
}

// PROpenedComment returns the appropriate GitHub Markdown-formatted content for the PR comment on opened/edited PRs
// referencing TEPs in `proposed` or `implementable` states.
func PROpenedComment(teps []tep.TEPInfo) string {
	return generatePRComment(ToImplementingCommentHeader, teps)
}

// PRMergedComment returns the appropriate GitHub Markdown-formatted content for the PR comment on merged PRs
// referencing TEPs in the `implementing` state.
func PRMergedComment(teps []tep.TEPInfo) string {
	return generatePRComment(ToImplementedCommentHeader, teps)
}

func generatePRComment(header string, teps []tep.TEPInfo) string {
	var commentLines []string
	var listLines []string
	var metadataLines []string

	for _, t := range teps {
		listLines = append(listLines, fmt.Sprintf(" * [TEP-%s (%s)](%s%s), current status: `%s`\n", t.ID, t.Title, tep.CommunityBlobBaseURL, t.Filename, t.Status))
		metadataLines = append(metadataLines, fmt.Sprintf(TEPCommentAndStatusRegexFmt, t.ID, t.Status))
	}

	commentLines = append(commentLines, header)
	commentLines = append(commentLines, listLines...)
	commentLines = append(commentLines, "\n") // Newline between the list of TEPs and the metadata
	commentLines = append(commentLines, metadataLines...)

	return strings.Join(commentLines, "") // TODO(abayer): Probably should get rid of trailing newlines on the header and the list/metadata lines and just join on "\n" here.
}
