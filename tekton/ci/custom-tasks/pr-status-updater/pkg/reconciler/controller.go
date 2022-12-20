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
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		r := &Reconciler{
			SCMClient: scmClient,
			BotUser:   botUser,
		}

		impl := runreconciler.NewImpl(ctx, r, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName: ControllerName,
			}
		})

		runinformer.Get(ctx).Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: tkncontroller.FilterRunRef("custom.tekton.dev/v0", "PRStatusUpdater"),
			Handler:    controller.HandleAll(impl.Enqueue),
		})

		return impl
	}
}
