// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

// InstrumentationReconciler reconciles Instrumentation objects to trigger
// rolling restarts of workloads when spec.autoUpdate is enabled.
type InstrumentationReconciler struct {
	client.Client
	scheme   *runtime.Scheme
	log      logr.Logger
	recorder record.EventRecorder
}

type InstrumentationReconcilerParams struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder record.EventRecorder
}

func NewInstrumentationReconciler(params InstrumentationReconcilerParams) *InstrumentationReconciler {
	return &InstrumentationReconciler{
		Client:   params.Client,
		scheme:   params.Scheme,
		log:      params.Log,
		recorder: params.Recorder,
	}
}

// inject annotations used to reference Instrumentation CRs from workloads.
var injectAnnotations = []string{
	"instrumentation.opentelemetry.io/inject-java",
	"instrumentation.opentelemetry.io/inject-nodejs",
	"instrumentation.opentelemetry.io/inject-python",
	"instrumentation.opentelemetry.io/inject-dotnet",
	"instrumentation.opentelemetry.io/inject-go",
	"instrumentation.opentelemetry.io/inject-apache-httpd",
	"instrumentation.opentelemetry.io/inject-nginx",
	"instrumentation.opentelemetry.io/inject-sdk",
}

const instrumentationSpecHashAnnotation = "instrumentation.opentelemetry.io/spec-hash"

//+kubebuilder:rbac:groups=opentelemetry.io,resources=instrumentations,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments;statefulsets;daemonsets,verbs=get;list;watch;update;patch

func (r *InstrumentationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithValues("instrumentation", req.NamespacedName)

	var inst v1alpha1.Instrumentation
	if err := r.Get(ctx, req.NamespacedName, &inst); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if inst.Spec.AutoUpdate == nil || !*inst.Spec.AutoUpdate {
		return ctrl.Result{}, nil
	}

	specHash, err := hashSpec(inst.Spec)
	if err != nil {
		log.Error(err, "failed to compute spec hash")
		return ctrl.Result{}, err
	}

	// Find all workloads in the same namespace that reference this Instrumentation CR.
	isOnlyInst, err := r.isOnlyInstrumentationInNamespace(ctx, inst.Namespace)
	if err != nil {
		log.Error(err, "failed to check instrumentation count")
		return ctrl.Result{}, err
	}

	var restartErr error
	restartErr = r.restartMatchingWorkloads(ctx, log, inst.Name, inst.Namespace, isOnlyInst, specHash, &appsv1.DeploymentList{},
		func(obj client.Object) *metav1.ObjectMeta { return &obj.(*appsv1.Deployment).Spec.Template.ObjectMeta })
	if restartErr != nil {
		return ctrl.Result{}, restartErr
	}

	restartErr = r.restartMatchingWorkloads(ctx, log, inst.Name, inst.Namespace, isOnlyInst, specHash, &appsv1.StatefulSetList{},
		func(obj client.Object) *metav1.ObjectMeta { return &obj.(*appsv1.StatefulSet).Spec.Template.ObjectMeta })
	if restartErr != nil {
		return ctrl.Result{}, restartErr
	}

	restartErr = r.restartMatchingWorkloads(ctx, log, inst.Name, inst.Namespace, isOnlyInst, specHash, &appsv1.DaemonSetList{},
		func(obj client.Object) *metav1.ObjectMeta { return &obj.(*appsv1.DaemonSet).Spec.Template.ObjectMeta })
	if restartErr != nil {
		return ctrl.Result{}, restartErr
	}

	return ctrl.Result{}, nil
}

// isOnlyInstrumentationInNamespace returns true if there is exactly one Instrumentation CR in the namespace.
func (r *InstrumentationReconciler) isOnlyInstrumentationInNamespace(ctx context.Context, ns string) (bool, error) {
	list := &v1alpha1.InstrumentationList{}
	if err := r.List(ctx, list, client.InNamespace(ns)); err != nil {
		return false, err
	}
	return len(list.Items) == 1, nil
}

// referencesInstrumentation checks if a workload's pod template annotations
// reference the given Instrumentation CR (by name or "true" when it's the only one).
func referencesInstrumentation(podMeta metav1.ObjectMeta, instName string, isOnlyInst bool) bool {
	for _, ann := range injectAnnotations {
		val, ok := podMeta.Annotations[ann]
		if !ok {
			continue
		}
		if val == instName {
			return true
		}
		if val == "true" && isOnlyInst {
			return true
		}
	}
	return false
}

type podTemplateAccessor func(obj client.Object) *metav1.ObjectMeta

// restartMatchingWorkloads lists workloads of a given type, checks if they reference
// the Instrumentation CR, and patches their pod template with the spec hash to trigger a rollout.
func (r *InstrumentationReconciler) restartMatchingWorkloads(
	ctx context.Context,
	log logr.Logger,
	instName, namespace string,
	isOnlyInst bool,
	specHash string,
	list client.ObjectList,
	getPodMeta podTemplateAccessor,
) error {
	if err := r.List(ctx, list, client.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list workloads: %w", err)
	}

	items := extractItems(list)
	for _, item := range items {
		podMeta := getPodMeta(item)
		if !referencesInstrumentation(*podMeta, instName, isOnlyInst) {
			continue
		}
		// Check if the hash already matches - no restart needed.
		if podMeta.Annotations != nil && podMeta.Annotations[instrumentationSpecHashAnnotation] == specHash {
			continue
		}

		log.Info("triggering rolling restart", "workload", item.GetName(), "kind", item.GetObjectKind().GroupVersionKind().Kind)

		patch := client.MergeFrom(item.DeepCopyObject().(client.Object))
		if podMeta.Annotations == nil {
			podMeta.Annotations = map[string]string{}
		}
		podMeta.Annotations[instrumentationSpecHashAnnotation] = specHash
		podMeta.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

		if err := r.Patch(ctx, item, patch); err != nil {
			log.Error(err, "failed to patch workload", "workload", item.GetName())
			return err
		}
		r.recorder.Eventf(item, "Normal", "InstrumentationUpdated",
			"Rolling restart triggered by Instrumentation %s/%s update", namespace, instName)
	}
	return nil
}

func extractItems(list client.ObjectList) []client.Object {
	var items []client.Object
	switch l := list.(type) {
	case *appsv1.DeploymentList:
		for i := range l.Items {
			items = append(items, &l.Items[i])
		}
	case *appsv1.StatefulSetList:
		for i := range l.Items {
			items = append(items, &l.Items[i])
		}
	case *appsv1.DaemonSetList:
		for i := range l.Items {
			items = append(items, &l.Items[i])
		}
	}
	return items
}

func hashSpec(spec v1alpha1.InstrumentationSpec) (string, error) {
	// Exclude AutoUpdate from the hash so toggling it doesn't trigger restarts.
	specCopy := spec
	specCopy.AutoUpdate = nil
	data, err := json.Marshal(specCopy)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

func (r *InstrumentationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Instrumentation{}).
		Complete(r)
}
