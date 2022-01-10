package performers

import (
	"context"

	"github.com/google/go-cmp/cmp"

	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/ghclient"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/tep"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/logging"
	kreconciler "knative.dev/pkg/reconciler"
)

// IssueCreator is a Performer that responds to newly opened or modified PRs to the community repository by creating or
// updating tracking issues, if appropriate.
type IssueCreator struct {
	GHClient *ghclient.TEPGHClient
}

// NewIssueCreator creates a new IssueCreator instance configured with a GitHub client
func NewIssueCreator(ghClient *ghclient.TEPGHClient) *IssueCreator {
	return &IssueCreator{
		GHClient: ghClient,
	}
}

// NewIssueCreatorAsPerformer calls NewIssueCreator and returns it as a Performer
func NewIssueCreatorAsPerformer(ghClient *ghclient.TEPGHClient) Performer {
	return NewIssueCreator(ghClient)
}

// Perform checks if the action for the PR is opened or edited and if it contains a TEP markdown file. If so, it creates
// (or updates) a tracking issue for it.
func (p *IssueCreator) Perform(ctx context.Context, opts *PerformerOptions) kreconciler.Event {
	// Skip out if the repo is not community
	if opts.Repo != "community" {
		return nil
	}

	logger := logging.FromContext(ctx).With(zap.String("performer", "IssueCreator"))
	logger.Debugf("Performing tracking issue creation/updating (if needed) for %s/%s", opts.RunNamespace, opts.RunName)

	// Short-circuit for PR actions other than `edited` or `opened`.
	if opts.Action != openedAction && opts.Action != editedAction {
		logger.Infof("Ignoring PR action %s; will do nothing", opts.Action)
		return nil
	}

	tepsInPR, err := p.GHClient.ExtractTEPInfoFromTEPPR(ctx, opts.PRNumber, opts.GitRevision)
	if err != nil {
		return kreconciler.NewEvent(corev1.EventTypeWarning, "LoadingPRTEPs", "Failure finding TEP markdown changes for %s/%s PR #%d: %s",
			ghclient.TEPsOwner, opts.Repo, opts.PRNumber, err.Error())
	}

	if len(tepsInPR) == 0 {
		logger.Infof("No TEPs added or modified in PR %d", opts.PRNumber)
		return nil
	}

	trackingIssues, err := p.GHClient.GetTrackingIssues(ctx, &ghclient.GetTrackingIssuesOptions{IssueState: "open"})
	if err != nil {
		return kreconciler.NewEvent(corev1.EventTypeWarning, "LoadingTrackingIssues", "Failure loading existing tracking issues: %s", err)
	}

	for _, t := range tepsInPR {
		if issue, ok := trackingIssues[t.ID]; ok {
			updatedIssue := &tep.TrackingIssue{}
			*updatedIssue = *issue
			updatedIssue.AddTEPPR(opts.PRNumber)
			for _, a := range t.Authors {
				updatedIssue.AddAssignee(a)
			}
			updatedIssue.TEPStatus = t.Status

			if !cmp.Equal(issue, updatedIssue) {
				issueBody, err := updatedIssue.GetBody(t.Filename)
				if err != nil {
					return kreconciler.NewEvent(corev1.EventTypeWarning, "TrackingIssueBody", "Failure generating tracking issue body for issue %d: %s", updatedIssue.IssueNumber, err)
				}
				logger.Infof("Updating tracking issue %d for TEP-%s in PR tektoncd/%s #%d", updatedIssue.IssueNumber, updatedIssue.TEPID, opts.Repo, opts.PRNumber)
				if err := p.GHClient.UpdateTrackingIssue(ctx, updatedIssue.IssueNumber, updatedIssue.TEPID, issueBody, updatedIssue.Assignees, updatedIssue.TEPStatus); err != nil {
					return kreconciler.NewEvent(corev1.EventTypeWarning, "UpdatingTrackingIssue", "Failure updating tracking issue for issue %d: %s", updatedIssue.IssueNumber, err)
				}
			} else {
				logger.Infof("No changes needed for tracking issue %d for TEP-%s", updatedIssue.IssueNumber, updatedIssue.TEPID)
			}

		} else {
			issue := tep.TrackingIssue{
				TEPStatus: tep.NewStatus,
				TEPID:     t.ID,
				TEPPRs:    []int{opts.PRNumber},
				Assignees: t.Authors,
			}
			issueBody, err := issue.GetBody(t.Filename)
			if err != nil {
				return kreconciler.NewEvent(corev1.EventTypeWarning, "TrackingIssueBody", "Failure generating tracking issue body for TEP-%s: %s", t.ID, err)
			}
			logger.Infof("Creating new tracking issue for TEP-%s in PR tektoncd/%s #%d", issue.TEPID, opts.Repo, opts.PRNumber)
			if err := p.GHClient.CreateTrackingIssue(ctx, issue.TEPID, issueBody, issue.Assignees, issue.TEPStatus); err != nil {
				return kreconciler.NewEvent(corev1.EventTypeWarning, "CreatingTrackingIssue", "Failure updating tracking issue for TEP-%s: %s", t.ID, err)
			}
		}
	}

	if len(tepsInPR) > 0 {
		return kreconciler.NewEvent(corev1.EventTypeNormal, "TrackingIssuesUpdatedOrCreated",
			"Tracking issues created or updated for PR #%d", opts.PRNumber)
	}

	return nil
}
