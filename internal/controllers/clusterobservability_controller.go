// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/clusterobservability"
	coStatus "github.com/open-telemetry/opentelemetry-operator/internal/status/clusterobservability"
)

// ClusterObservabilityReconciler reconciles a ClusterObservability object.
type ClusterObservabilityReconciler struct {
	client.Client
	recorder record.EventRecorder
	scheme   *runtime.Scheme
	log      logr.Logger
	config   config.Config
}

// ClusterObservabilityReconcilerParams is the set of options to build a new ClusterObservabilityReconciler.
type ClusterObservabilityReconcilerParams struct {
	client.Client
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Config   config.Config
}

func (r *ClusterObservabilityReconciler) getParams(instance v1alpha1.ClusterObservability) manifests.Params {
	return manifests.Params{
		Config:               r.config,
		Client:               r.Client,
		ClusterObservability: instance,
		Log:                  r.log,
		Scheme:               r.scheme,
		Recorder:             r.recorder,
	}
}

func NewClusterObservabilityReconciler(params ClusterObservabilityReconcilerParams) *ClusterObservabilityReconciler {
	reconciler := &ClusterObservabilityReconciler{
		Client:   params.Client,
		scheme:   params.Scheme,
		log:      params.Log,
		recorder: params.Recorder,
		config:   params.Config,
	}
	return reconciler
}

//+kubebuilder:rbac:groups=opentelemetry.io,resources=clusterobservabilities,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=opentelemetry.io,resources=clusterobservabilities/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=opentelemetry.io,resources=clusterobservabilities/finalizers,verbs=update
//+kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=opentelemetry.io,resources=instrumentations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
//+kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ClusterObservabilityReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithValues("clusterobservability", req.NamespacedName)

	var instance v1alpha1.ClusterObservability
	if err := r.Client.Get(ctx, req.NamespacedName, &instance); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch ClusterObservability")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion
	if deletionTimestamp := instance.GetDeletionTimestamp(); deletionTimestamp != nil {
		return r.handleDeletion(ctx, log, &instance)
	}

	// Validate singleton constraint
	isActive, conflictErr := r.validateSingleton(ctx, log, &instance)
	if conflictErr != nil {
		return ctrl.Result{}, conflictErr
	}

	if !isActive {
		// This instance is conflicted, update status and skip reconciliation
		params := r.getParams(instance)
		return coStatus.HandleReconcileStatus(ctx, log, params, fmt.Errorf("multiple ClusterObservability resources detected"))
	}

	// TODO: Add upgrade support
	// TODO: Support management state like OpenTelemetryCollector

	configChanged, configErr := coStatus.DetectConfigChanges(&instance)
	if configErr != nil {
		log.Error(configErr, "failed to detect config changes")
	}

	if configChanged {
		log.Info("Configuration changes detected - triggering full reconciliation")
		r.recorder.Event(&instance, corev1.EventTypeNormal, "ConfigChanged",
			"Collector configuration has changed, updating managed resources")
	}

	// Add finalizer to ensure proper resource cleanup
	if !controllerutil.ContainsFinalizer(&instance, v1alpha1.ClusterObservabilityFinalizer) {
		if controllerutil.AddFinalizer(&instance, v1alpha1.ClusterObservabilityFinalizer) {
			if err := r.Update(ctx, &instance); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	log.V(2).Info("Reconciling ClusterObservability managed resources")

	params := r.getParams(instance)

	desiredObjects, buildErr := clusterobservability.Build(params)
	if buildErr != nil {
		return ctrl.Result{}, buildErr
	}

	var openTelemetryCRs []client.Object
	var unstructuredObjects []client.Object
	var regularObjects []client.Object

	for _, obj := range desiredObjects {
		switch obj.(type) {
		case *v1beta1.OpenTelemetryCollector, *v1alpha1.Instrumentation:
			openTelemetryCRs = append(openTelemetryCRs, obj)
		case *unstructured.Unstructured:
			unstructuredObjects = append(unstructuredObjects, obj)
		default:
			regularObjects = append(regularObjects, obj)
		}
	}

	// Handle OpenTelemetry CRs - their controllers manage the underlying resources
	for _, crObj := range openTelemetryCRs {
		if err := r.reconcileOpenTelemetryResource(ctx, log, crObj); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile OpenTelemetry CR %s: %w", crObj.GetObjectKind(), err)
		}
	}

	// Handle Unstructured objects (like OpenShift SCC) separately to avoid deep copy issues
	for _, unstructuredObj := range unstructuredObjects {
		if err := r.reconcileUnstructuredResource(ctx, log, unstructuredObj); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile unstructured resource %s: %w", unstructuredObj.GetName(), err)
		}
	}
	// Handle regular Kubernetes resources (currently none - OpenTelemetry CRs handle their own resources)
	if len(regularObjects) > 0 {
		ownedObjects, err := r.findClusterObservabilityOwnedObjects(ctx, params)
		if err != nil {
			return ctrl.Result{}, err
		}
		err = reconcileDesiredObjects(ctx, r.Client, log, &params.ClusterObservability, params.Scheme, regularObjects, ownedObjects)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return coStatus.HandleReconcileStatus(ctx, log, params, nil)
}

// reconcileOpenTelemetryResource creates/updates OpenTelemetry CRs.
// Their respective controllers handle the underlying Kubernetes resources.
// TODO: fix issue with resourceVersion becoming stale due to updates from OpenTelemetryCollector/Instrumentation controllers.
func (r *ClusterObservabilityReconciler) reconcileOpenTelemetryResource(ctx context.Context, log logr.Logger, desired client.Object) error {
	key := client.ObjectKeyFromObject(desired)

	var existing client.Object
	switch desired.(type) {
	case *v1beta1.OpenTelemetryCollector:
		existing = &v1beta1.OpenTelemetryCollector{}
	case *v1alpha1.Instrumentation:
		existing = &v1alpha1.Instrumentation{}
	default:
		return fmt.Errorf("unsupported CRD type: %T", desired)
	}

	getErr := r.Get(ctx, key, existing)

	if getErr != nil {
		if apierrors.IsNotFound(getErr) {
			if createErr := r.Create(ctx, desired); createErr != nil {
				return fmt.Errorf("failed to create %s %s: %w", desired.GetObjectKind().GroupVersionKind().Kind, key, createErr)
			}
			log.Info("Created CR", "kind", desired.GetObjectKind().GroupVersionKind().Kind, "name", key.Name, "namespace", key.Namespace)
			return nil
		}
		return fmt.Errorf("failed to get %s %s: %w", desired.GetObjectKind().GroupVersionKind().Kind, key, getErr)
	}
	switch existingCRD := existing.(type) {
	case *v1beta1.OpenTelemetryCollector:
		desiredCRD := desired.(*v1beta1.OpenTelemetryCollector)
		if !apiequality.Semantic.DeepEqual(existingCRD.Spec, desiredCRD.Spec) {
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				latest := &v1beta1.OpenTelemetryCollector{}
				if err := r.Get(ctx, key, latest); err != nil {
					return err
				}

				// Only update if still different (another controller might have updated it)
				if apiequality.Semantic.DeepEqual(latest.Spec, desiredCRD.Spec) {
					log.Info("OpenTelemetryCollector already matches desired state", "name", key.Name, "namespace", key.Namespace)
					return nil
				}

				// Update the latest version with our desired changes
				latest.Spec = desiredCRD.Spec
				latest.Labels = desiredCRD.Labels
				latest.Annotations = desiredCRD.Annotations

				return r.Update(ctx, latest)
			})

			if err != nil {
				return fmt.Errorf("failed to update OpenTelemetryCollector %s: %w", key, err)
			}

			log.Info("Updated OpenTelemetryCollector", "name", key.Name, "namespace", key.Namespace)
		}

	case *v1alpha1.Instrumentation:
		desiredCRD := desired.(*v1alpha1.Instrumentation)
		if !apiequality.Semantic.DeepEqual(existingCRD.Spec, desiredCRD.Spec) {
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				latest := &v1alpha1.Instrumentation{}
				if err := r.Get(ctx, key, latest); err != nil {
					return err
				}

				// Only update if still different (another controller might have updated it)
				if apiequality.Semantic.DeepEqual(latest.Spec, desiredCRD.Spec) {
					log.Info("Instrumentation already matches desired state", "name", key.Name, "namespace", key.Namespace)
					return nil
				}

				// Update the latest version with our desired changes
				latest.Spec = desiredCRD.Spec
				latest.Labels = desiredCRD.Labels
				latest.Annotations = desiredCRD.Annotations

				return r.Update(ctx, latest)
			})

			if err != nil {
				return fmt.Errorf("failed to update Instrumentation %s: %w", key, err)
			}

			log.Info("Updated Instrumentation", "name", key.Name, "namespace", key.Namespace)
		}

	default:
		return fmt.Errorf("unsupported CRD type: %T", existing)
	}

	return nil
}

// reconcileUnstructuredResource handles Unstructured objects (like OpenShift SCCs)
// without deep copy issues that occur with complex nested data.
func (r *ClusterObservabilityReconciler) reconcileUnstructuredResource(ctx context.Context, log logr.Logger, obj client.Object) error {
	unstructuredObj := obj.(*unstructured.Unstructured)

	// Create a new Unstructured object for fetching existing resource
	// This avoids deep copy issues with the desired object
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(unstructuredObj.GroupVersionKind())

	key := client.ObjectKeyFromObject(unstructuredObj)
	getErr := r.Client.Get(ctx, key, existing)
	if getErr != nil && !apierrors.IsNotFound(getErr) {
		return fmt.Errorf("failed to get existing unstructured resource %s: %w", unstructuredObj.GetName(), getErr)
	}

	if apierrors.IsNotFound(getErr) {
		// Create new resource
		if createErr := r.Client.Create(ctx, unstructuredObj); createErr != nil {
			return fmt.Errorf("failed to create unstructured resource %s: %w", unstructuredObj.GetName(), createErr)
		}
		log.Info("Created unstructured resource",
			"kind", unstructuredObj.GetKind(),
			"name", unstructuredObj.GetName())
	} else {
		// Check if update is needed by comparing specs
		if !apiequality.Semantic.DeepEqual(existing.Object, unstructuredObj.Object) {
			unstructuredObj.SetResourceVersion(existing.GetResourceVersion())
			if updateErr := r.Client.Update(ctx, unstructuredObj); updateErr != nil {
				return fmt.Errorf("failed to update unstructured resource %s: %w", unstructuredObj.GetName(), updateErr)
			}
			log.Info("Updated unstructured resource",
				"kind", unstructuredObj.GetKind(),
				"name", unstructuredObj.GetName())
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterObservabilityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := r.SetupCaches(mgr)
	if err != nil {
		return err
	}

	ownedResources := r.GetOwnedResourceTypes()
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClusterObservability{}).
		Watches(
			&corev1.Namespace{},
			handler.EnqueueRequestsFromMapFunc(r.findClusterObservabilityForNamespace),
		)

	for _, resource := range ownedResources {
		builder.Owns(resource)
	}

	return builder.Complete(r)
}

// SetupCaches sets up field indexing for efficient owned object queries.
func (r *ClusterObservabilityReconciler) SetupCaches(mgr ctrl.Manager) error {
	const clusterObservabilityResourceOwnerKey = ".metadata.owner"

	ownedResources := r.GetOwnedResourceTypes()
	for _, resource := range ownedResources {
		if err := mgr.GetCache().IndexField(context.Background(), resource, clusterObservabilityResourceOwnerKey, func(rawObj client.Object) []string {
			owner := metav1.GetControllerOf(rawObj)
			if owner == nil {
				return nil
			}
			// Make sure it's a ClusterObservability
			if owner.APIVersion != v1alpha1.GroupVersion.String() || owner.Kind != "ClusterObservability" {
				return nil
			}
			return []string{owner.Name}
		}); err != nil {
			return err
		}
	}
	return nil
}

// findClusterObservabilityForNamespace finds ClusterObservability instances when namespaces change.
func (r *ClusterObservabilityReconciler) findClusterObservabilityForNamespace(_ context.Context, obj client.Object) []ctrl.Request {
	ctx := context.Background()

	var clusterObservabilityList v1alpha1.ClusterObservabilityList
	if err := r.List(ctx, &clusterObservabilityList); err != nil {
		r.log.Error(err, "failed to list ClusterObservability resources")
		return nil
	}

	var requests []ctrl.Request
	for _, co := range clusterObservabilityList.Items {
		requests = append(requests, ctrl.Request{
			NamespacedName: client.ObjectKeyFromObject(&co),
		})
	}
	return requests
}

// validateSingleton ensures only one ClusterObservability resource is active in the cluster.
// Returns true if this instance is the active one, false if conflicted.
func (r *ClusterObservabilityReconciler) validateSingleton(ctx context.Context, log logr.Logger, instance *v1alpha1.ClusterObservability) (bool, error) {
	var clusterObservabilityList v1alpha1.ClusterObservabilityList
	if err := r.List(ctx, &clusterObservabilityList); err != nil {
		log.Error(err, "failed to list ClusterObservability resources for singleton validation")
		return false, err
	}

	// Filter out deleted resources and find the oldest active resource
	var activeResources []v1alpha1.ClusterObservability
	for _, co := range clusterObservabilityList.Items {
		if co.DeletionTimestamp == nil {
			activeResources = append(activeResources, co)
		}
	}

	if len(activeResources) <= 1 {
		// No conflict, this is the only active resource
		return true, nil
	}

	// Multiple resources exist, determine which one should be active
	// Use oldest by creation timestamp as the winner
	// If timestamps are equal, use lexicographical name comparison as tie-breaker
	oldestResource := &activeResources[0]
	for i := 1; i < len(activeResources); i++ {
		candidate := &activeResources[i]

		if candidate.CreationTimestamp.Before(&oldestResource.CreationTimestamp) {
			oldestResource = candidate
		} else if candidate.CreationTimestamp.Equal(&oldestResource.CreationTimestamp) {
			candidateKey := candidate.Namespace + "/" + candidate.Name
			oldestKey := oldestResource.Namespace + "/" + oldestResource.Name
			if candidateKey < oldestKey {
				oldestResource = candidate
			}
		}
	}

	isWinner := oldestResource.UID == instance.UID

	if !isWinner {
		// This resource is conflicted, emit an event and update status
		r.recorder.Event(instance, corev1.EventTypeWarning, "Conflicted",
			fmt.Sprintf("Multiple ClusterObservability resources detected. Only %s/%s (oldest) is active",
				oldestResource.Namespace, oldestResource.Name))
		log.Info("ClusterObservability resource is conflicted",
			"active", fmt.Sprintf("%s/%s", oldestResource.Namespace, oldestResource.Name),
			"conflicted", fmt.Sprintf("%s/%s", instance.Namespace, instance.Name))
	} else {
		// This resource is the winner, emit events for conflicted ones
		for _, conflicted := range activeResources {
			if conflicted.UID != instance.UID {
				r.recorder.Event(&conflicted, corev1.EventTypeWarning, "Conflicted",
					fmt.Sprintf("Multiple ClusterObservability resources detected. Only %s/%s (oldest) is active",
						instance.Namespace, instance.Name))
			}
		}
		log.Info("ClusterObservability resource is active", "conflicted-count", len(activeResources)-1)
	}

	return isWinner, nil
}

// handleDeletion handles the cleanup of ClusterObservability resources and managed objects.
func (r *ClusterObservabilityReconciler) handleDeletion(ctx context.Context, log logr.Logger, instance *v1alpha1.ClusterObservability) (ctrl.Result, error) {
	log.Info("Handling ClusterObservability deletion")

	if !controllerutil.ContainsFinalizer(instance, v1alpha1.ClusterObservabilityFinalizer) {
		// Finalizer already removed, nothing to do
		return ctrl.Result{}, nil
	}

	// Clean up all managed resources
	if err := r.cleanupManagedResources(ctx, log, instance); err != nil {
		log.Error(err, "failed to cleanup managed resources")
		r.recorder.Event(instance, corev1.EventTypeWarning, "CleanupFailed",
			fmt.Sprintf("Failed to cleanup managed resources: %v", err))
		return ctrl.Result{RequeueAfter: time.Second * 30}, err
	}

	// Remove finalizer to allow deletion
	latest := &v1alpha1.ClusterObservability{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(instance), latest); err != nil {
		log.Error(err, "failed to get latest ClusterObservability for finalizer removal")
		return ctrl.Result{}, err
	}

	controllerutil.RemoveFinalizer(latest, v1alpha1.ClusterObservabilityFinalizer)
	if err := r.Update(ctx, latest); err != nil {
		log.Error(err, "failed to remove finalizer")
		return ctrl.Result{}, err
	}

	log.Info("Successfully cleaned up ClusterObservability resources")
	r.recorder.Event(instance, corev1.EventTypeNormal, "Deleted", "ClusterObservability and all managed resources cleaned up")

	return ctrl.Result{}, nil
}

// cleanupManagedResources deletes cluster-scoped resources managed by ClusterObservability.
// Namespace-scoped resources (OpenTelemetryCollector and Instrumentation CRs) are automatically
// cleaned up by Kubernetes garbage collection via owner references.
func (r *ClusterObservabilityReconciler) cleanupManagedResources(ctx context.Context, log logr.Logger, instance *v1alpha1.ClusterObservability) error {
	// Only clean up cluster-scoped resources that cannot use owner references
	if err := r.cleanupClusterScopedResources(ctx, log, instance); err != nil {
		return fmt.Errorf("failed to cleanup cluster-scoped resources: %w", err)
	}

	log.Info("Cluster-scoped resources cleaned up successfully")
	return nil
}

// cleanupClusterScopedResources removes cluster-scoped resources that can't use owner references.
func (r *ClusterObservabilityReconciler) cleanupClusterScopedResources(ctx context.Context, log logr.Logger, instance *v1alpha1.ClusterObservability) error {

	if r.config.OpenShiftRoutesAvailability == openshift.RoutesAvailable {
		agentCollectorName := fmt.Sprintf("%s-%s", instance.Name, clusterobservability.AgentCollectorSuffix)
		sccName := fmt.Sprintf("%s-hostaccess", agentCollectorName)

		scc := &unstructured.Unstructured{}
		scc.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "security.openshift.io",
			Version: "v1",
			Kind:    "SecurityContextConstraints",
		})
		scc.SetName(sccName)

		if err := r.Delete(ctx, scc); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete SecurityContextConstraints %s: %w", sccName, err)
		}
		log.Info("Deleted SecurityContextConstraints", "name", sccName)
	}

	return nil
}

// GetOwnedResourceTypes returns CRs directly created by ClusterObservability.
// Note: We only track OpenTelemetry CRs we create, not the underlying K8s resources
// (those are managed by OpenTelemetryCollector controller).
func (r *ClusterObservabilityReconciler) GetOwnedResourceTypes() []client.Object {
	return []client.Object{
		&v1beta1.OpenTelemetryCollector{},
		&v1alpha1.Instrumentation{},
	}
}

// findClusterObservabilityOwnedObjects finds OpenTelemetry CRs owned by ClusterObservability for cleanup.
func (r *ClusterObservabilityReconciler) findClusterObservabilityOwnedObjects(ctx context.Context, params manifests.Params) (map[types.UID]client.Object, error) {
	const clusterObservabilityResourceOwnerKey = ".metadata.owner"
	ownedObjects := map[types.UID]client.Object{}

	listOpts := []client.ListOption{
		client.InNamespace(params.ClusterObservability.Namespace),
		client.MatchingFields{clusterObservabilityResourceOwnerKey: params.ClusterObservability.Name},
	}

	ownedObjectTypes := r.GetOwnedResourceTypes()
	for _, objectType := range ownedObjectTypes {
		objs, err := getList(ctx, r.Client, objectType, listOpts...)
		if err != nil {
			return nil, err
		}
		for uid, object := range objs {
			ownedObjects[uid] = object
		}
	}

	return ownedObjects, nil
}
