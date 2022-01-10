package performers

import (
	"context"
	"strconv"
	"strings"

	"github.com/tektoncd/plumbing/bots/tep-automation/pkg/tep"
	corev1 "k8s.io/api/core/v1"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	kreconciler "knative.dev/pkg/reconciler"
)

const (
	closedAction = "closed"
	openedAction = "opened"
	editedAction = "edited"
)

// Performer is an interface to actually perform the action for a given event
type Performer interface {
	Perform(ctx context.Context, opts *PerformerOptions) kreconciler.Event
}

// PerformerOptions contains the parameters passed to the Run, processed into an appropriate form for Performers to use.
type PerformerOptions struct {
	RunName      string
	RunNamespace string
	Action       string
	PRNumber     int
	Title        string
	Body         string
	Repo         string
	IsMerged     bool
	GitRevision  string
}

// ToPerformerOptions extracts known parameters from a run's spec, and returns a configured PerformerOptions, or an error
// if a parameter is missing or an invalid value, or has too many parameters.
func ToPerformerOptions(run *v1alpha1.Run) (*PerformerOptions, error) {
	if len(run.Spec.Params) > 7 {
		var found []string
		for _, p := range run.Spec.Params {
			if p.Name == tep.ActionParamName ||
				p.Name == tep.PRNumberParamName ||
				p.Name == tep.PRTitleParamName ||
				p.Name == tep.PRBodyParamName ||
				p.Name == tep.PackageParamName ||
				p.Name == tep.PRIsMergedParamName ||
				p.Name == tep.GitRevisionParamName {
				continue
			}
			found = append(found, p.Name)
		}
		return nil, kreconciler.NewEvent(corev1.EventTypeWarning, "UnexpectedParams", "Found unexpected params: %v", found)
	}

	opts := &PerformerOptions{
		RunName:      run.Name,
		RunNamespace: run.Namespace,
	}

	prActionParam := run.Spec.GetParam(tep.ActionParamName)
	if prActionParam == nil || prActionParam.Value.StringVal == "" {
		return nil, kreconciler.NewEvent(corev1.EventTypeWarning, "MissingPullRequestAction", "The %s param was not passed", tep.ActionParamName)
	}
	opts.Action = prActionParam.Value.StringVal

	prNumberParam := run.Spec.GetParam(tep.PRNumberParamName)
	if prNumberParam == nil || prNumberParam.Value.StringVal == "" {
		return nil, kreconciler.NewEvent(corev1.EventTypeWarning, "MissingPullRequestNumber", "The %s param was not passed", tep.PRNumberParamName)
	}
	prNumber, err := strconv.Atoi(prNumberParam.Value.StringVal)
	if err != nil {
		return nil, kreconciler.NewEvent(corev1.EventTypeWarning, "InvalidPullRequestNumber", "%s is not a valid value for the %s param", prNumberParam.Value.StringVal, tep.PRNumberParamName)
	}
	opts.PRNumber = prNumber

	prTitleParam := run.Spec.GetParam(tep.PRTitleParamName)
	if prTitleParam == nil || prTitleParam.Value.StringVal == "" {
		return nil, kreconciler.NewEvent(corev1.EventTypeWarning, "MissingPullRequestTitle", "The %s param was not passed", tep.PRTitleParamName)
	}
	opts.Title = prTitleParam.Value.StringVal

	prBodyParam := run.Spec.GetParam(tep.PRBodyParamName)
	if prBodyParam == nil || prBodyParam.Value.StringVal == "" {
		return nil, kreconciler.NewEvent(corev1.EventTypeWarning, "MissingPullRequestBody", "The %s param was not passed", tep.PRBodyParamName)
	}
	opts.Body = prBodyParam.Value.StringVal

	orgAndRepoParam := run.Spec.GetParam(tep.PackageParamName)
	if orgAndRepoParam == nil || orgAndRepoParam.Value.StringVal == "" {
		return nil, kreconciler.NewEvent(corev1.EventTypeWarning, "MissingPackage", "The %s param was not passed", tep.PackageParamName)
	}
	splitOrgAndRepo := strings.Split(orgAndRepoParam.Value.StringVal, "/")
	if len(splitOrgAndRepo) < 2 {
		return nil, kreconciler.NewEvent(corev1.EventTypeWarning, "InvalidPackage", "The %s param value %s does not contain an owner and a repository seperated by '/'",
			tep.PackageParamName, orgAndRepoParam.Value.StringVal)
	}
	opts.Repo = splitOrgAndRepo[1]

	isMergedParam := run.Spec.GetParam(tep.PRIsMergedParamName)
	if isMergedParam == nil || isMergedParam.Value.StringVal == "" {
		return nil, kreconciler.NewEvent(corev1.EventTypeWarning, "MissingPullRequestIsMerged", "The %s param was not passed", tep.PRIsMergedParamName)
	}
	isMerged, err := strconv.ParseBool(isMergedParam.Value.StringVal)
	if err != nil {
		return nil, kreconciler.NewEvent(corev1.EventTypeWarning, "InvalidPullRequestIsMerged", "%s is not a valid value for the %s param", isMergedParam.Value.StringVal, tep.PRIsMergedParamName)
	}
	opts.IsMerged = isMerged

	gitRevParam := run.Spec.GetParam(tep.GitRevisionParamName)
	if gitRevParam == nil || gitRevParam.Value.StringVal == "" {
		return nil, kreconciler.NewEvent(corev1.EventTypeWarning, "MissingPullRequestSHA", "The %s param was not passed", tep.GitRevisionParamName)
	}
	opts.GitRevision = gitRevParam.Value.StringVal

	return opts, nil
}
