package performers

import (
	"bytes"
	"context"
	"text/template"

	"github.com/google/go-cmp/cmp"

	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/ghclient"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/tep"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/logging"
	kreconciler "knative.dev/pkg/reconciler"
)

const (
	// CloseTrackingIssueCommentTmpl is a text/template used to generate the comment added to a tracking issue when it's closed.
	CloseTrackingIssueCommentTmpl = `Closing tracking issue for TEP-{{ .id }} because it has reached the terminal status {{ .status }}`
)

// IssueManager is a Performer that responds to newly opened and merged PRs to the community repository by creating or
// updating tracking issues, if appropriate.
type IssueManager struct {
	GHClient *ghclient.TEPGHClient
}

// NewIssueManager creates a new IssueManager instance configured with a GitHub client
func NewIssueManager(ghClient *ghclient.TEPGHClient) *IssueManager {
	return &IssueManager{
		GHClient: ghClient,
	}
}

// NewIssueManagerAsPerformer calls NewIssueManager and returns it as a Performer
func NewIssueManagerAsPerformer(ghClient *ghclient.TEPGHClient) Performer {
	return NewIssueManager(ghClient)
}

// Perform checks if the action for the PR is opened, closed, or synchronize, and if it contains a TEP markdown file. If
// so, it creates, updates, or closes (as appropriate) a tracking issue for it.
func (p *IssueManager) Perform(ctx context.Context, opts *PerformerOptions) kreconciler.Event {
	// Skip out if the repo is not community
	if opts.Repo != "community" {
		return nil
	}

	logger := logging.FromContext(ctx).With(zap.String("performer", "IssueManager"))
	logger.Debugf("Performing tracking issue creation/updating (if needed) for %s/%s", opts.RunNamespace, opts.RunName)

	// Short-circuit for PR actions other than `opened`, `closed`, or `synchronize`.
	if opts.Action != OpenedAction &&
		opts.Action != ClosedAction &&
		opts.Action != SynchronizeAction {
		logger.Infof("Ignoring PR action %s; will do nothing", opts.Action)
		return nil
	}

	// If the PR action is `closed` but not merged, short-circuit.
	if opts.Action == ClosedAction && !opts.IsMerged {
		logger.Infof("Ignoring closed but unmerged PR; will do nothing")
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
			// Update the tracking issue if the PR has been merged, and if the TEP is now in a terminal status, close the issue.
			if opts.Action == ClosedAction {
				if err := p.updateIssue(ctx, t, issue, t.Status, opts, logger); err != nil {
					return err
				}
				if t.Status.IsTerminalStatus() {
					buf := bytes.NewBufferString("")

					if bodyTmpl, err := template.New("trackingIssueComment").Parse(CloseTrackingIssueCommentTmpl); err != nil {
						return kreconciler.NewEvent(corev1.EventTypeWarning, "TrackingIssueComment", "failed to parse template for close tracking issue comment: %v", err)
					} else if err := bodyTmpl.Execute(buf, map[string]interface{}{"id": t.ID, "status": t.Status.ForMarkdown()}); err != nil {
						return kreconciler.NewEvent(corev1.EventTypeWarning, "TrackingIssueComment", "failed to execute template for close tracking issue comment: %v", err)
					}

					if err := p.GHClient.CloseTrackingIssue(ctx, issue.IssueNumber, buf.String()); err != nil {
						return kreconciler.NewEvent(corev1.EventTypeWarning, "CloseTrackingIssue", "Failure closing tracking issue %d: %v", issue.IssueNumber, err)
					}
				}
			} else if opts.Action == SynchronizeAction && issue.TEPStatus == tep.NewStatus {
				// If the PR is introducing a new TEP, that TEP already has a tracking issue, and the tracking issue thinks the TEP is still new,
				// update the issue.
				if err := p.updateIssue(ctx, t, issue, tep.NewStatus, opts, logger); err != nil {
					return err
				}
			}
		} else if opts.Action == OpenedAction {
			// If the PR is newly opened and is referencing a TEP that does not already have a tracking issue, create it.
			issue := tep.TrackingIssue{
				TEPStatus: tep.NewStatus,
				TEPID:     t.ID,
				TEPPRs:    []int{opts.PRNumber},
				Assignees: t.Authors,
			}
			// By default, we'll set the TEP status on the tracking issue to "tep-status/new" for completely new TEPs which
			// haven't been merged as proposed, but we need to handle pre-existing TEPs without tracking issues. In cases
			// where we're creating a tracking issue for a TEP but the TEP in the PR isn't marked as proposed, we'll actually
			// use its status, so that follow-up PRs to pre-existing TEPs will get their tracking issues created properly.
			if t.Status != tep.ProposedStatus {
				issue.TEPStatus = t.Status
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

func (p *IssueManager) updateIssue(ctx context.Context, tepInfo tep.TEPInfo, origIssue *tep.TrackingIssue, desiredStatus tep.Status, opts *PerformerOptions, logger *zap.SugaredLogger) kreconciler.Event {
	updatedIssue := &tep.TrackingIssue{}
	*updatedIssue = *origIssue
	updatedIssue.AddTEPPR(opts.PRNumber)
	for _, a := range tepInfo.Authors {
		updatedIssue.AddAssignee(a)
	}
	updatedIssue.TEPStatus = desiredStatus

	if !cmp.Equal(origIssue, updatedIssue) {
		issueBody, err := updatedIssue.GetBody(tepInfo.Filename)
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

	return nil
}
