// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"context"
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/instrumentation"
	_ "github.com/open-telemetry/opentelemetry-operator/pkg/instrumentation"
)

const annotationRevision = "instrumentation.opentelemetry.io/revision"

// InstrumentationReconciler reconciles a Instrumentation object
type InstrumentationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=opentelemetry.io,resources=instrumentations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=opentelemetry.io,resources=instrumentations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=opentelemetry.io,resources=instrumentations/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=list;watch;update;patch
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get
//+kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get

// Reconcile the current state of an Instrumentation resource with the desired state.
func (r *InstrumentationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("instrumentation-reconciler", req.NamespacedName)

	var instance v1alpha1.Instrumentation
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "unable to fetch Instrumentation")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("instance.Spec", "AutoUpdate", instance.Spec.AutoUpdate)

	// NOTE: currently, we perform an update whenever the instrumentation object
	// is changed. This also applies to changes that do not affect the
	// configuration in the spec, such as changes to annotations or labels.
	// We could optimize this by creating a checksum of the relevant configuration
	// parameters and updating only if this has changed.
	if !instance.Spec.AutoUpdate {
		return ctrl.Result{}, nil
	}

	var ns corev1.Namespace
	if err := r.Client.Get(ctx, types.NamespacedName{Name: instance.GetNamespace()}, &ns); err != nil {
		logger.Error(err, "unable to fetch namespace")
		return ctrl.Result{}, err
	}

	var pods corev1.PodList
	if err := r.Client.List(ctx, &pods, client.InNamespace(instance.GetNamespace())); err != nil {
		logger.Error(err, "unable to list pods")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// NOTE: if it turns out that the mutator makes too many API requests,
	// we can use a client implementation with cache at this point to reduce
	// the requests.
	m := instrumentation.NewMutator(logger, r.Client)
	for _, pod := range pods.Items {
		p, err := m.Mutate(ctx, ns, *pod.DeepCopy())
		if err != nil {
			return ctrl.Result{}, err
		}
		if equality.Semantic.DeepEqual(pod.Spec, p.Spec) {
			continue
		}

		owner, err := retrieveOwner(ctx, r.Client, logger, &pod)
		if err != nil {
			logger.Info("could not retrieve owner", "pod", pod.GetName())
			continue
		}

		obj, err := owner.mutatePodSpecAnnotation()
		if err != nil {
			return ctrl.Result{}, err
		}

		if err := r.Client.Update(ctx, obj); err != nil {
			return ctrl.Result{}, err
		}

	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InstrumentationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Instrumentation{}).
		Owns(&v1alpha1.Instrumentation{}).
		Complete(r)
}

type owner struct {
	Deployment  *appsv1.Deployment
	DaemonSet   *appsv1.DaemonSet
	StatefulSet *appsv1.StatefulSet
}

func (o *owner) mutatePodSpecAnnotation() (client.Object, error) {
	if o.Deployment != nil {
		increaseRevision(o.Deployment.Spec.Template.Annotations)
		return o.Deployment, nil
	}

	if o.DaemonSet != nil {
		increaseRevision(o.DaemonSet.Spec.Template.Annotations)
		return o.DaemonSet, nil
	}

	if o.StatefulSet != nil {
		increaseRevision(o.StatefulSet.Spec.Template.Annotations)
		return o.StatefulSet, nil
	}

	return nil, fmt.Errorf("missing owner")
}

// increaseRevision increases the revision counter if a inject annoation exists.
// If no number exists, it is initialized with 0.
func increaseRevision(annotations map[string]string) {
	if annotations == nil {
		return
	}
	revStr := "0"
	v := annotations[annotationRevision]
	if rev, err := strconv.Atoi(v); err == nil {
		revStr = strconv.Itoa(rev + 1)
	}
	annotations[annotationRevision] = revStr
}

func retrieveOwner(ctx context.Context, c client.Client, logger logr.Logger, pod *corev1.Pod) (*owner, error) {
	podRefList := pod.GetOwnerReferences()
	if len(podRefList) != 1 || podRefList[0].Kind != "ReplicaSet" {
		return nil, fmt.Errorf("missing single ReplicaSet as owner")
	}

	namespaceName := pod.GetNamespace()
	replicaName := podRefList[0].Name
	replicaSet := &appsv1.ReplicaSet{}
	logger = logger.WithValues("replicasetName", replicaName, "replicasetNamespace", namespaceName)
	logger.V(3).Info("fetch replicaset")

	key := types.NamespacedName{Namespace: namespaceName, Name: replicaName}
	if err := c.Get(ctx, key, replicaSet); err != nil {
		return nil, fmt.Errorf("failed to get the available Pod ReplicaSet")
	}

	repRefList := replicaSet.GetOwnerReferences()
	if len(repRefList) != 1 {
		return nil, fmt.Errorf("could not determine owner, number of owner: %d", len(repRefList))
	}

	var obj client.Object
	ref := repRefList[0]
	owner := &owner{}
	switch ref.Kind {
	case "Deployment":
		owner.Deployment = &appsv1.Deployment{}
		obj = owner.Deployment
	case "DaemonSet":
		owner.DaemonSet = &appsv1.DaemonSet{}
		obj = owner.DaemonSet
	case "StatefulSet":
		owner.StatefulSet = &appsv1.StatefulSet{}
		obj = owner.StatefulSet
	default:
		return nil, fmt.Errorf("unsupported owner type: %s", ref.Kind)
	}

	key = types.NamespacedName{Namespace: namespaceName, Name: ref.Name}
	if err := c.Get(ctx, key, obj); err != nil {
		return nil, fmt.Errorf("failed to get the Pod owner")
	}

	return owner, nil
}
