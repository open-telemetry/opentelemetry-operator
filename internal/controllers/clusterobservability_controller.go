// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
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
	"k8s.io/client-go/tools/events"
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
	"github.com/open-telemetry/opentelemetry-operator/internal/webhook"
)

// ClusterObservabilityReconciler reconciles a ClusterObservability object.
type ClusterObservabilityReconciler struct {
	client.Client
	recorder events.EventRecorder
	scheme   *runtime.Scheme
	log      logr.Logger
	config   config.Config
}

// ClusterObservabilityReconcilerParams is the set of options to build a new ClusterObservabilityReconciler.
type ClusterObservabilityReconcilerParams struct {
	client.Client
	Recorder events.EventRecorder
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Config   config.Config
}

const clusterObservabilityResourceOwnerKey = ".metadata.owner"

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
//+kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
//+kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ClusterObservabilityReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithValues("clusterobservability", req.NamespacedName)

	var instance v1alpha1.ClusterObservability
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
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
		return coStatus.HandleReconcileStatus(ctx, log, params, errors.New("multiple ClusterObservability resources detected"))
	}

	// Rebuild desired state on every reconcile so manager defaults propagate after upgrades.
	// TODO: Support management state like OpenTelemetryCollector

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

	// Prerequisites such as the OpenShift SCC must exist before child controllers
	// can create their workloads.
	for _, unstructuredObj := range unstructuredObjects {
		if err := r.reconcileUnstructuredResource(ctx, log, unstructuredObj); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to reconcile unstructured resource %s: %w", unstructuredObj.GetName(), err)
		}
	}
	for _, crObj := range openTelemetryCRs {
		if err := r.defaultOpenTelemetryResource(ctx, log, crObj); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to default desired %T %s: %w", crObj, client.ObjectKeyFromObject(crObj), err)
		}
	}

	ownedObjects, err := r.findClusterObservabilityOwnedObjects(ctx, params)
	if err != nil {
		return ctrl.Result{}, err
	}

	managedObjects := append(openTelemetryCRs, regularObjects...)
	if err := reconcileDesiredObjects(ctx, r.Client, log, &params.ClusterObservability, params.Scheme, managedObjects, ownedObjects); err != nil {
		return ctrl.Result{}, err
	}
	return coStatus.HandleReconcileStatus(ctx, log, params, nil)
}

func (r *ClusterObservabilityReconciler) defaultOpenTelemetryResource(
	ctx context.Context,
	log logr.Logger,
	desired client.Object,
) error {
	switch resource := desired.(type) {
	case *v1beta1.OpenTelemetryCollector:
		// Default is nil-safe without the optional admission and event collaborators.
		defaulter := webhook.NewCollectorWebhook(log, r.scheme, r.config, nil, nil, nil, nil, nil)
		if err := defaulter.Default(ctx, resource); err != nil {
			return err
		}
		return normalizeCollectorConfigForAPI(resource)
	case *v1alpha1.Instrumentation:
		defaulter := webhook.NewInstrumentationWebhook(log, r.scheme, r.config)
		return defaulter.Default(ctx, resource)
	default:
		return fmt.Errorf("unsupported CRD type: %T", desired)
	}
}

// normalizeCollectorConfigForAPI round-trips opaque values through JSON so
// webhook defaults use the same dynamic types as API-server values.
func normalizeCollectorConfigForAPI(resource *v1beta1.OpenTelemetryCollector) error {
	// AnyConfig's custom marshaler has a pointer receiver.
	configJSON, err := json.Marshal(&resource.Spec.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal desired collector config: %w", err)
	}
	var normalized v1beta1.Config
	if err := json.Unmarshal(configJSON, &normalized); err != nil {
		return fmt.Errorf("failed to normalize desired collector config: %w", err)
	}
	resource.Spec.Config = normalized
	return nil
}

// reconcileUnstructuredResource handles Unstructured objects (like OpenShift SCCs)
// without deep copy issues that occur with complex nested data.
func (r *ClusterObservabilityReconciler) reconcileUnstructuredResource(ctx context.Context, log logr.Logger, obj client.Object) error {
	unstructuredObj := obj.(*unstructured.Unstructured)
	normalizedDesired, err := normalizeUnstructuredForAPI(unstructuredObj)
	if err != nil {
		return fmt.Errorf("failed to normalize unstructured resource %s: %w", unstructuredObj.GetName(), err)
	}

	// Create a new Unstructured object for fetching existing resource
	// This avoids deep copy issues with the desired object
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(unstructuredObj.GroupVersionKind())

	key := client.ObjectKeyFromObject(unstructuredObj)
	getErr := r.Get(ctx, key, existing)
	if getErr != nil && !apierrors.IsNotFound(getErr) {
		return fmt.Errorf("failed to get existing unstructured resource %s: %w", unstructuredObj.GetName(), getErr)
	}

	if apierrors.IsNotFound(getErr) {
		// Create new resource
		if createErr := r.Create(ctx, normalizedDesired); createErr != nil {
			return fmt.Errorf("failed to create unstructured resource %s: %w", unstructuredObj.GetName(), createErr)
		}
		log.Info("Created unstructured resource",
			"kind", unstructuredObj.GetKind(),
			"name", unstructuredObj.GetName())
		return nil
	}

	updated := false
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &unstructured.Unstructured{}
		latest.SetGroupVersionKind(unstructuredObj.GroupVersionKind())
		if err := r.Get(ctx, key, latest); err != nil {
			return err
		}

		candidate := latest.DeepCopy()
		applyDesiredUnstructuredResource(candidate, normalizedDesired)
		if apiequality.Semantic.DeepEqual(latest.Object, candidate.Object) {
			return nil
		}
		if err := r.Update(ctx, candidate); err != nil {
			return err
		}
		updated = true
		return nil
	}); err != nil {
		return fmt.Errorf("failed to update unstructured resource %s: %w", unstructuredObj.GetName(), err)
	}
	if updated {
		log.Info("Updated unstructured resource",
			"kind", unstructuredObj.GetKind(),
			"name", unstructuredObj.GetName())
	}

	return nil
}

func normalizeUnstructuredForAPI(resource *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	data, err := json.Marshal(resource.Object)
	if err != nil {
		return nil, err
	}
	normalized := &unstructured.Unstructured{}
	if err := json.Unmarshal(data, normalized); err != nil {
		return nil, err
	}
	normalized.SetGroupVersionKind(resource.GroupVersionKind())
	return normalized, nil
}

func applyDesiredUnstructuredResource(existing, desired *unstructured.Unstructured) {
	if desiredLabels := desired.GetLabels(); len(desiredLabels) > 0 {
		labels := maps.Clone(existing.GetLabels())
		if labels == nil {
			labels = map[string]string{}
		}
		maps.Copy(labels, desiredLabels)
		existing.SetLabels(labels)
	}

	if desiredAnnotations := desired.GetAnnotations(); len(desiredAnnotations) > 0 {
		annotations := maps.Clone(existing.GetAnnotations())
		if annotations == nil {
			annotations = map[string]string{}
		}
		maps.Copy(annotations, desiredAnnotations)
		existing.SetAnnotations(annotations)
	}

	for key, value := range desired.Object {
		if key == "apiVersion" || key == "kind" || key == "metadata" {
			continue
		}
		existing.Object[key] = runtime.DeepCopyJSONValue(value)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterObservabilityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.SetupCaches(mgr.GetCache()); err != nil {
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

// SetupCaches indexes resources by their ClusterObservability controller owner.
func (r *ClusterObservabilityReconciler) SetupCaches(indexer client.FieldIndexer) error {
	for _, resource := range r.GetOwnedResourceTypes() {
		if err := indexer.IndexField(context.Background(), resource, clusterObservabilityResourceOwnerKey, clusterObservabilityOwnerName); err != nil {
			return err
		}
	}
	return nil
}

func clusterObservabilityOwnerName(rawObj client.Object) []string {
	owner := metav1.GetControllerOf(rawObj)
	if owner == nil || owner.APIVersion != v1alpha1.GroupVersion.String() || owner.Kind != "ClusterObservability" {
		return nil
	}
	return []string{owner.Name}
}

// findClusterObservabilityForNamespace finds ClusterObservability instances when namespaces change.
func (r *ClusterObservabilityReconciler) findClusterObservabilityForNamespace(context.Context, client.Object) []ctrl.Request {
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
		r.recorder.Eventf(instance, nil, corev1.EventTypeWarning, "Conflicted", "Conflicted",
			"Multiple ClusterObservability resources detected. Only %s/%s (oldest) is active",
			oldestResource.Namespace, oldestResource.Name)
		log.Info("ClusterObservability resource is conflicted",
			"active", fmt.Sprintf("%s/%s", oldestResource.Namespace, oldestResource.Name),
			"conflicted", fmt.Sprintf("%s/%s", instance.Namespace, instance.Name))
	} else {
		// This resource is the winner, emit events for conflicted ones
		for _, conflicted := range activeResources {
			if conflicted.UID != instance.UID {
				r.recorder.Eventf(&conflicted, nil, corev1.EventTypeWarning, "Conflicted", "Conflicted",
					"Multiple ClusterObservability resources detected. Only %s/%s (oldest) is active",
					instance.Namespace, instance.Name)
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
		r.recorder.Eventf(instance, nil, corev1.EventTypeWarning, "CleanupFailed", "CleanupFailed",
			"Failed to cleanup managed resources: %v", err)
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
	r.recorder.Eventf(instance, nil, corev1.EventTypeNormal, "Deleted", "Deleted", "ClusterObservability and all managed resources cleaned up")

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
func (*ClusterObservabilityReconciler) GetOwnedResourceTypes() []client.Object {
	return []client.Object{
		&v1beta1.OpenTelemetryCollector{},
		&v1alpha1.Instrumentation{},
	}
}

func (r *ClusterObservabilityReconciler) findClusterObservabilityOwnedObjects(ctx context.Context, params manifests.Params) (map[types.UID]client.Object, error) {
	ownedObjects := map[types.UID]client.Object{}
	listOpts := []client.ListOption{
		client.InNamespace(params.ClusterObservability.Namespace),
		client.MatchingFields{clusterObservabilityResourceOwnerKey: params.ClusterObservability.Name},
	}

	for _, objectType := range r.GetOwnedResourceTypes() {
		objects, err := getList(ctx, r.Client, objectType, listOpts...)
		if err != nil {
			return nil, err
		}
		maps.Copy(ownedObjects, objects)
	}
	return ownedObjects, nil
}
