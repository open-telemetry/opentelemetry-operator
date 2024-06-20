package controllers

import (
    "context"
    "fmt"

	"github.com/go-logr/logr"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    "k8s.io/apimachinery/pkg/runtime"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
    client.Client
    scheme *runtime.Scheme
	log      logr.Logger
	config   config.Config
}

// PodReconcilerParams is the set of options to build a new PodReconciler.
type PodReconcilerParams struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Config   config.Config
}

// NewPodReconciler creates a new pod reconciler for pod objects.
func NewPodReconciler(p PodReconcilerParams) *PodReconciler {
	r := &PodReconciler{
		Client:   p.Client,
		log:      p.Log,
		scheme:   p.Scheme,
		config:   p.Config,
	}
	return r
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := r.log.WithValues("opentelemetrypodcontroller", req.NamespacedName)

    // Fetch the Pod instance
    pod := &corev1.Pod{}
    err := r.Get(ctx, req.NamespacedName, pod)
    if err != nil {
        if errors.IsNotFound(err) {
            // Pod not found. Return and don't requeue
            log.Info("Pod resource not found. Ignoring since object must be deleted")
            return ctrl.Result{}, nil
        }
        // Error reading the object - requeue the request.
        log.Error(err, "Failed to get Pod")
        return ctrl.Result{}, err
    }

	fmt.Println("Mithun", pod)

    // Return and requeue after a specified duration
    return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&corev1.Pod{}).
        Complete(r)
}