// Package reconciler contains the reconciler for the PR status updater custom task.
package reconciler

import (
	"context"

	"github.com/jenkins-x/go-scm/scm"
	runinformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/customrun"
	runreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/customrun"
	tkncontroller "github.com/tektoncd/pipeline/pkg/controller"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
)

const (
	// ControllerName is the name of the PR status updater controller
	ControllerName = "pr-status-updater-controller"
)

// NewController instantiates a new controller
func NewController(scmClient *scm.Client, botUser string) func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, _ configmap.Watcher) *controller.Impl {
		r := &Reconciler{
			SCMClient: scmClient,
			BotUser:   botUser,
		}

		impl := runreconciler.NewImpl(ctx, r, func(_ *controller.Impl) controller.Options {
			return controller.Options{
				AgentName: ControllerName,
			}
		})

		if _, err := runinformer.Get(ctx).Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: tkncontroller.FilterCustomRunRef("custom.tekton.dev/v0", "PRStatusUpdater"),
			Handler:    controller.HandleAll(impl.Enqueue),
		}); err != nil {
			panic(err)
		}

		return impl
	}
}
