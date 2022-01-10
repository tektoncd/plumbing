package reconciler

import (
	"context"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/ghclient"
	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/performers"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/logging"
	kreconciler "knative.dev/pkg/reconciler"
)

// Reconciler handles the actual parsing of PR title and description for TEP identifiers, and adds comments to the PR
// as appropriate.
type Reconciler struct {
	GHClient   *ghclient.TEPGHClient
	Performers []performers.Performer
}

// NewReconciler creates a new Reconciler instance configured with a GitHub client
func NewReconciler(ctx context.Context, ghToken string, performerFuncs ...func(*ghclient.TEPGHClient) performers.Performer) *Reconciler {
	tgc := ghclient.NewTEPGHClientFromToken(ctx, ghToken)
	r := &Reconciler{
		GHClient:   tgc,
		Performers: []performers.Performer{},
	}

	for _, pf := range performerFuncs {
		r.Performers = append(r.Performers, pf(tgc))
	}

	return r
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

	performerOpts, err := performers.ToPerformerOptions(run)
	if err != nil {
		if recEvt, ok := err.(*kreconciler.ReconcilerEvent); ok {
			run.Status.MarkRunFailed(recEvt.Reason, recEvt.Format, recEvt.Args)
		} else {
			run.Status.MarkRunFailed("UnknownError", "unknown error occurred: %s", err)
		}
		return nil
	}

	var results []error
	mu := &sync.Mutex{}
	var wg sync.WaitGroup
	wg.Add(len(r.Performers))

	for _, performer := range r.Performers {
		go func(p performers.Performer) {
			defer wg.Done()
			res := p.Perform(ctx, performerOpts)
			mu.Lock()
			results = append(results, res)
			mu.Unlock()
		}(performer)
	}

	wg.Wait()

	hasResult := false
	var failures []error
	for _, res := range results {
		if res != nil {
			hasResult = true
			recEvt, ok := res.(*kreconciler.ReconcilerEvent)
			if !ok {
				failures = append(failures, res)
			} else if recEvt.EventType != corev1.EventTypeNormal {
				failures = append(failures, res)
			}
		}
	}

	// If none of the performers returned a non-nil event, don't do anything to the run status
	if !hasResult {
		return nil
	}

	switch len(failures) {
	case 0:
		run.Status.MarkRunSucceeded("AllSucceeded", "TEP Automation successful")
	case 1:
		if recEvt, ok := failures[0].(*kreconciler.ReconcilerEvent); ok {
			run.Status.MarkRunFailed(recEvt.Reason, recEvt.Format, recEvt.Args)
		} else {
			run.Status.MarkRunFailed("UnknownError", "unknown error occurred: %s", failures[0])
		}
	default:
		var allErrors error
		allErrors = multierror.Append(allErrors, failures...)
		run.Status.MarkRunFailed("MultipleErrors", "multiple errors: %s", allErrors)
	}

	return nil
}
