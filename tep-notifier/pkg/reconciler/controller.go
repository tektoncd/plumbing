package reconciler

import (
	"context"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	runinformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1alpha1/run"
	runreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	tkncontroller "github.com/tektoncd/pipeline/pkg/controller"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
)

const (
	// ControllerName is the name for this controller. Kind of obvious, really. =)
	ControllerName = "tep-notifier-controller"
	// Kind is used in the TaskRef for Runs using the TEP notifier
	Kind = "TEPNotifier"
)

// NewController instantiates a new controller.Impl from knative.dev/pkg/controller
func NewController(ghToken string) func(context.Context, configmap.Watcher) *controller.Impl {
	return func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)
		runInformer := runinformer.Get(ctx)

		r := NewReconciler(ctx, ghToken)

		impl := runreconciler.NewImpl(ctx, r, func(impl *controller.Impl) controller.Options {
			return controller.Options{
				AgentName: ControllerName,
			}
		})

		logger.Info("Setting up event handlers")

		// Add event handler for Runs
		runInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
			FilterFunc: tkncontroller.FilterRunRef(v1beta1.SchemeGroupVersion.String(), Kind),
			Handler:    controller.HandleAll(impl.Enqueue),
		})

		return impl
	}
}
