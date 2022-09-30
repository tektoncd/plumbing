package reconciler

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"go.uber.org/zap"
	"knative.dev/pkg/logging"
	kreconciler "knative.dev/pkg/reconciler"
)

const (
	commentTag = "<!-- Tekton test report -->"
)

// Reconciler is the core of the implementation of the PR commenter, adding, updating, or deleting comments as needed.
type Reconciler struct {
	SCMClient *scm.Client
	BotUser   string
}

// ReconcileKind implements Interface.ReconcileKind.
func (c *Reconciler) ReconcileKind(ctx context.Context, r *v1alpha1.Run) kreconciler.Event {
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling %s/%s", r.Namespace, r.Name)

	// Ignore completed waits.
	if r.IsDone() {
		logger.Info("Run is finished, done reconciling")
		return nil
	}

	if r.Spec.Ref == nil ||
		r.Spec.Ref.APIVersion != "custom.tekton.dev/v0" || r.Spec.Ref.Kind != "PRCommenter" {
		// This is not a Run we should have been notified about; do nothing.
		return nil
	}
	if r.Spec.Ref.Name != "" {
		r.Status.MarkRunFailed("UnexpectedName", "Found unexpected ref name: %s", r.Spec.Ref.Name)
		return fmt.Errorf("unexpected ref name: %s", r.Spec.Ref.Name)
	}

	spec, err := ReportInfoFromRun(r)
	if err != nil {
		r.Status.MarkRunFailed("InvalidParams", "Invalid parameters: %s", err.Error())
		return err
	}

	// Don't do anything unless the result is failure or success
	if spec.Result != "pending" {
		fieldErr := c.reportComment(ctx, spec, logger)
		if fieldErr != nil {
			r.Status.MarkRunFailed("SCMError", "Error interacting with SCM: %s", fieldErr.Error())
			return fieldErr
		}
	}

	r.Status.MarkRunSucceeded("Commented", "PR comment successfully added/updated/deleted")

	// Don't emit events on nop-reconciliations, it causes scale problems.
	return nil
}

func createComment(entries []string) (string, error) {
	if len(entries) == 0 {
		return "", nil
	}
	plural := ""
	if len(entries) > 1 {
		plural = "s"
	}
	lines := []string{
		fmt.Sprintf("The following Tekton test%s **failed**:", plural),
		"",
		"Test name | Commit | Details | Required | Rerun command",
		"--- | --- | --- | --- | ---",
	}
	lines = append(lines, entries...)
	lines = append(lines, []string{
		"",
		commentTag,
	}...)
	return strings.Join(lines, "\n"), nil
}

func parseIssueComments(report *ReportInfo, botUser string, ics []*scm.Comment) ([]int, []string, int) {
	var deleteComments []int
	var previousComments []int
	var latestComment int
	var entries []string
	// First accumulate result entries and comment IDs
	for _, ic := range ics {
		if botUser != ic.Author.Login {
			continue
		}
		if !strings.Contains(ic.Body, commentTag) {
			continue
		}
		if latestComment != 0 {
			previousComments = append(previousComments, latestComment)
		}
		latestComment = ic.ID
		var tracking bool
		for _, line := range strings.Split(ic.Body, "\n") {
			line = strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(line, "---"):
				tracking = true
			case len(line) == 0:
				tracking = false
			case tracking:
				entries = append(entries, line)
			}
		}
	}
	var newEntries []string

	// Next decide which entries to keep.
	for i := range entries {
		f1 := strings.Split(entries[i], " | ")
		if f1[0] != report.JobName {
			newEntries = append(newEntries, entries[i])
		}
	}
	var createNewComment bool

	if jobEntry := createEntry(report); jobEntry != "" {
		createNewComment = true
		newEntries = append(newEntries, jobEntry)
	}
	deleteComments = append(deleteComments, previousComments...)
	if (createNewComment || len(newEntries) == 0) && latestComment != 0 {
		deleteComments = append(deleteComments, latestComment)
		latestComment = 0
	}
	return deleteComments, newEntries, latestComment
}

func createEntry(report *ReportInfo) string {
	if report.Result == "success" {
		return ""
	}

	required := strconv.FormatBool(!report.IsOptional)

	retestPrefix := os.Getenv("RETEST_PREFIX")
	if retestPrefix == "" {
		retestPrefix = "test"
	}

	return strings.Join([]string{
		report.JobName,
		report.SHA,
		fmt.Sprintf("[link](%s)", report.LogURL),
		required,
		fmt.Sprintf("`/%s %s`", retestPrefix, report.JobName),
	}, " | ")
}

func (c *Reconciler) reportComment(ctx context.Context, report *ReportInfo, logger *zap.SugaredLogger) error {
	ics, err := c.listPullRequestComments(ctx, report.Repo, report.PRNumber)
	if err != nil {
		return fmt.Errorf("error listing comments: %w", err)
	}
	deletes, entries, updateID := parseIssueComments(report, c.BotUser, ics)
	for _, deleteCmt := range deletes {
		logger.Infof("Deleting stale comment %d for %s #%d", deleteCmt, report.Repo, report.PRNumber)
		if _, err := c.SCMClient.PullRequests.DeleteComment(ctx, report.Repo, report.PRNumber, deleteCmt); err != nil {
			return fmt.Errorf("error deleting comment: %w", err)
		}
	}
	if len(entries) > 0 {
		comment, err := createComment(entries)
		if err != nil {
			return fmt.Errorf("generating comment: %w", err)
		}
		if comment == "" {
			logger.Infof("No failures for %s #%d, skipping creation", report.Repo, report.PRNumber)
			return nil
		}
		if updateID == 0 {
			logger.Infof("Creating new comment for %s #%d", report.Repo, report.PRNumber)
			if _, _, err := c.SCMClient.PullRequests.CreateComment(ctx, report.Repo, report.PRNumber, &scm.CommentInput{Body: comment}); err != nil {
				return fmt.Errorf("error creating comment: %w", err)
			}
		} else {
			logger.Infof("Updating existing comment %d for %s #%d", updateID, report.Repo, report.PRNumber)
			if _, _, err := c.SCMClient.PullRequests.EditComment(ctx, report.Repo, report.PRNumber, updateID, &scm.CommentInput{Body: comment}); err != nil {
				if err == scm.ErrNotSupported {
					logger.Infof("updating comments not supported, falling back on delete/create")
					if _, err = c.SCMClient.PullRequests.DeleteComment(ctx, report.Repo, report.PRNumber, updateID); err != nil {
						return fmt.Errorf("error deleting comment: %w", err)
					}
					if _, _, err = c.SCMClient.PullRequests.CreateComment(ctx, report.Repo, report.PRNumber, &scm.CommentInput{Body: comment}); err != nil {
						return fmt.Errorf("error creating comment: %w", err)
					}
				} else {
					return fmt.Errorf("error updating comment: %w", err)
				}
			}
		}
	}
	return nil
}

func (c *Reconciler) listPullRequestComments(ctx context.Context, repo string, number int) ([]*scm.Comment, error) {
	var allComments []*scm.Comment
	var resp *scm.Response
	var comments []*scm.Comment
	var err error
	firstRun := false
	opts := scm.ListOptions{
		Page: 1,
	}
	for !firstRun || (resp != nil && opts.Page <= resp.Page.Last) {
		comments, resp, err = c.SCMClient.PullRequests.ListComments(ctx, repo, number, &opts)
		if err != nil {
			return nil, err
		}
		firstRun = true
		allComments = append(allComments, comments...)
		opts.Page++
	}
	return allComments, nil
}
