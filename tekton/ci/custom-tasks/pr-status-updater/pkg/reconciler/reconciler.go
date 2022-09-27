package reconciler

import (
	"context"
	"fmt"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"knative.dev/pkg/logging"
	kreconciler "knative.dev/pkg/reconciler"
)

const (
	commentTag = "<!-- Tekton test report -->"
)

// Reconciler is the core of the implementation of the PR commenter, adding, updating, or deleting comments as needed.
type Reconciler struct {
	SCMClient *scm.Client
	Owner     string
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
		r.Spec.Ref.APIVersion != "custom.tekton.dev/v0" || r.Spec.Ref.Kind != "PRStatusUpdater" {
		// This is not a Run we should have been notified about; do nothing.
		return nil
	}
	if r.Spec.Ref.Name != "" {
		r.Status.MarkRunFailed("UnexpectedName", "Found unexpected ref name: %s", r.Spec.Ref.Name)
		return fmt.Errorf("unexpected ref name: %s", r.Spec.Ref.Name)
	}

	spec, fieldErr := StatusInfoFromRun(r)
	if fieldErr != nil {
		r.Status.MarkRunFailed("InvalidParams", "Invalid parameters: %s", fieldErr.Error())
		return fieldErr
	}

	gitRepoStatus := &scm.StatusInput{
		State:  scm.ToState(spec.State),
		Label:  spec.JobName,
		Desc:   spec.Description,
		Target: spec.TargetURL,
	}

	_, _, err := c.SCMClient.Repositories.CreateStatus(ctx, fmt.Sprintf("%s/%s", c.Owner, spec.Repo), spec.SHA, gitRepoStatus)
	if err != nil {
		r.Status.MarkRunFailed("SCMError", "Error interacting with SCM: %s", err.Error())
		return err
	}

	r.Status.MarkRunSucceeded("Commented", "PR status successfully set")

	// Don't emit events on nop-reconciliations, it causes scale problems.
	return nil
}
