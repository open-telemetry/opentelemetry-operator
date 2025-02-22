// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package controllers contains the main controller, where the reconciliation starts.
package controllers

import (
	"context"
	"sort"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyV1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	internalRbac "github.com/open-telemetry/opentelemetry-operator/internal/rbac"
	collectorStatus "github.com/open-telemetry/opentelemetry-operator/internal/status/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

const resourceOwnerKey = ".metadata.owner"

var (
	ownedClusterObjectTypes = []client.Object{
		&rbacv1.ClusterRole{},
		&rbacv1.ClusterRoleBinding{},
		&rbacv1.Role{},
		&rbacv1.RoleBinding{},
	}
)

// OpenTelemetryCollectorReconciler reconciles a OpenTelemetryCollector object.
type OpenTelemetryCollectorReconciler struct {
	client.Client
	recorder record.EventRecorder
	scheme   *runtime.Scheme
	log      logr.Logger
	config   config.Config
	reviewer *internalRbac.Reviewer
}

// Params is the set of options to build a new OpenTelemetryCollectorReconciler.
type Params struct {
	client.Client
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Config   config.Config
	Reviewer *internalRbac.Reviewer
}

func (r *OpenTelemetryCollectorReconciler) findOtelOwnedObjects(ctx context.Context, params manifests.Params) (map[types.UID]client.Object, error) {
	ownedObjects := map[types.UID]client.Object{}
	collectorConfigMaps := []*corev1.ConfigMap{}
	ownedObjectTypes := r.GetOwnedResourceTypes()
	listOpts := []client.ListOption{
		client.InNamespace(params.OtelCol.Namespace),
		client.MatchingFields{resourceOwnerKey: params.OtelCol.Name},
	}
	rbacObjectsFound := false
	for _, objectType := range ownedObjectTypes {
		var objs map[types.UID]client.Object
		objs, err := getList(ctx, r, objectType, listOpts...)
		if err != nil {
			return nil, err
		}

		// save Collector ConfigMaps into a separate slice, we need to do additional filtering on them
		switch objectType.(type) {
		case *corev1.ConfigMap:
			for _, object := range objs {
				if !featuregate.CollectorUsesTargetAllocatorCR.IsEnabled() && object.GetLabels()["app.kubernetes.io/component"] != "opentelemetry-collector" {
					// we only apply this to collector ConfigMaps
					continue
				}
				configMap := object.(*corev1.ConfigMap)
				collectorConfigMaps = append(collectorConfigMaps, configMap)
			}
		case *rbacv1.ClusterRoleBinding, *rbacv1.ClusterRole, *rbacv1.RoleBinding, *rbacv1.Role:
			if params.Config.CreateRBACPermissions() == rbac.Available && !rbacObjectsFound {
				objs, err = r.findRBACObjects(ctx, params)
				if err != nil {
					return nil, err
				}
				rbacObjectsFound = true
			}
		default:
		}

		for uid, object := range objs {
			ownedObjects[uid] = object
		}
	}
	// at this point we don't know if the most recent ConfigMap will still be the most recent after reconciliation, or
	// if a new one will be created. We keep one additional ConfigMap to account for this. The next reconciliation that
	// doesn't spawn a new ConfigMap will delete the extra one we kept here.
	configVersionsToKeep := max(params.OtelCol.Spec.ConfigVersions, 1) + 1
	configMapsToKeep := getCollectorConfigMapsToKeep(configVersionsToKeep, collectorConfigMaps)
	for _, configMap := range configMapsToKeep {
		delete(ownedObjects, configMap.GetUID())
	}

	return ownedObjects, nil
}

// findRBACObjects finds ClusterRoles, ClusterRoleBindings, Roles, and RoleBindings.
// Those objects do not have owner references.
//   - ClusterRoles and ClusterRoleBindings cannot have owner references
//   - Roles and RoleBindings can exist in a different namespace than the OpenTelemetryCollector
//
// Users might switch off the RBAC creation feature on the operator which should remove existing RBAC.
func (r *OpenTelemetryCollectorReconciler) findRBACObjects(ctx context.Context, params manifests.Params) (map[types.UID]client.Object, error) {
	ownedObjects := map[types.UID]client.Object{}

	listOpsCluster := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			manifestutils.SelectorLabels(params.OtelCol.ObjectMeta, collector.ComponentOpenTelemetryCollector)),
	}
	for _, objectType := range ownedClusterObjectTypes {
		objs, err := getList(ctx, r, objectType, listOpsCluster)
		if err != nil {
			return nil, err
		}
		for uid, object := range objs {
			ownedObjects[uid] = object
		}
	}
	return ownedObjects, nil
}

// getCollectorConfigMapsToKeep gets ConfigMaps the controller would normally delete, but which we want to keep around
// anyway. This is part of a feature to keep around previous ConfigMap versions to make rollbacks easier.
// Fundamentally, this just sorts by time created and picks configVersionsToKeep latest ones.
func getCollectorConfigMapsToKeep(configVersionsToKeep int, configMaps []*corev1.ConfigMap) []*corev1.ConfigMap {
	configVersionsToKeep = max(1, configVersionsToKeep)
	sort.Slice(configMaps, func(i, j int) bool {
		iTime := configMaps[i].GetCreationTimestamp().Time
		jTime := configMaps[j].GetCreationTimestamp().Time
		// sort the ConfigMaps newest to oldest
		return iTime.After(jTime)
	})

	configMapsToKeep := min(configVersionsToKeep, len(configMaps))
	// return the first configVersionsToKeep items
	return configMaps[:configMapsToKeep]
}

func (r *OpenTelemetryCollectorReconciler) GetParams(ctx context.Context, instance v1beta1.OpenTelemetryCollector) (manifests.Params, error) {
	p := manifests.Params{
		Config:   r.config,
		Client:   r.Client,
		OtelCol:  instance,
		Log:      r.log,
		Scheme:   r.scheme,
		Recorder: r.recorder,
		Reviewer: r.reviewer,
	}

	// generate the target allocator CR from the collector CR
	targetAllocator, err := r.getTargetAllocator(ctx, p)
	if err != nil {
		return p, err
	}
	p.TargetAllocator = targetAllocator
	return p, nil
}

func (r *OpenTelemetryCollectorReconciler) getTargetAllocator(ctx context.Context, params manifests.Params) (*v1alpha1.TargetAllocator, error) {
	if taName, ok := params.OtelCol.GetLabels()[constants.LabelTargetAllocator]; ok {
		targetAllocator := &v1alpha1.TargetAllocator{}
		taKey := client.ObjectKey{Name: taName, Namespace: params.OtelCol.GetNamespace()}
		err := r.Client.Get(ctx, taKey, targetAllocator)
		if err != nil {
			return nil, err
		}
		return targetAllocator, nil
	}
	return collector.TargetAllocator(params)
}

// NewReconciler creates a new reconciler for OpenTelemetryCollector objects.
func NewReconciler(p Params) *OpenTelemetryCollectorReconciler {
	r := &OpenTelemetryCollectorReconciler{
		Client:   p.Client,
		log:      p.Log,
		scheme:   p.Scheme,
		config:   p.Config,
		recorder: p.Recorder,
		reviewer: p.Reviewer,
	}
	return r
}

// +kubebuilder:rbac:groups="",resources=pods;configmaps;services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=daemonsets;deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors;podmonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes;routes/custom-host,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=config.openshift.io,resources=infrastructures;infrastructures/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=opentelemetry.io,resources=opentelemetrycollectors/finalizers,verbs=get;update;patch
// +kubebuilder:rbac:groups=opentelemetry.io,resources=targetallocators,verbs=get;list;watch;create;update;patch;delete

// Reconcile the current state of an OpenTelemetry collector resource with the desired state.
func (r *OpenTelemetryCollectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithValues("opentelemetrycollector", req.NamespacedName)

	var instance v1beta1.OpenTelemetryCollector
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch OpenTelemetryCollector")
		}

		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	params, err := r.GetParams(ctx, instance)
	if err != nil {
		log.Error(err, "Failed to create manifest.Params")
		return ctrl.Result{}, err
	}

	// We have a deletion, short circuit and let the deletion happen
	if deletionTimestamp := instance.GetDeletionTimestamp(); deletionTimestamp != nil {
		if controllerutil.ContainsFinalizer(&instance, collectorFinalizer) {
			// If the finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err = r.finalizeCollector(ctx, params); err != nil {
				return ctrl.Result{}, err
			}

			// Once all finalizers have been
			// removed, the object will be deleted.
			if controllerutil.RemoveFinalizer(&instance, collectorFinalizer) {
				err = r.Update(ctx, &instance)
				if err != nil {
					return ctrl.Result{}, err
				}
			}
		}

		return ctrl.Result{}, nil
	}

	if instance.Spec.ManagementState == v1beta1.ManagementStateUnmanaged {
		log.Info("Skipping reconciliation for unmanaged OpenTelemetryCollector resource", "name", req.String())
		// Stop requeueing for unmanaged OpenTelemetryCollector custom resources
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(&instance, collectorFinalizer) {
		if controllerutil.AddFinalizer(&instance, collectorFinalizer) {
			err = r.Update(ctx, &instance)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	desiredObjects, buildErr := BuildCollector(params)
	if buildErr != nil {
		return ctrl.Result{}, buildErr
	}

	ownedObjects, err := r.findOtelOwnedObjects(ctx, params)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconcileDesiredObjects(ctx, r.Client, log, &instance, params.Scheme, desiredObjects, ownedObjects)
	return collectorStatus.HandleReconcileStatus(ctx, log, params, instance, err)
}

// SetupWithManager tells the manager what our controller is interested in.
func (r *OpenTelemetryCollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := r.SetupCaches(mgr)
	if err != nil {
		return err
	}

	ownedResources := r.GetOwnedResourceTypes()
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.OpenTelemetryCollector{})

	for _, resource := range ownedResources {
		builder.Owns(resource)
	}

	return builder.Complete(r)
}

// SetupCaches sets up caching and indexing for our controller.
func (r *OpenTelemetryCollectorReconciler) SetupCaches(cluster cluster.Cluster) error {
	ownedResources := r.GetOwnedResourceTypes()
	for _, resource := range ownedResources {
		if err := cluster.GetCache().IndexField(context.Background(), resource, resourceOwnerKey, func(rawObj client.Object) []string {
			owner := metav1.GetControllerOf(rawObj)
			if owner == nil {
				return nil
			}
			// make sure it's an OpenTelemetryCollector
			if owner.Kind != "OpenTelemetryCollector" {
				return nil
			}

			return []string{owner.Name}
		}); err != nil {
			return err
		}
	}
	return nil
}

// GetOwnedResourceTypes returns all the resource types the controller can own. Even though this method returns an array
// of client.Object, these are (empty) example structs rather than actual resources.
func (r *OpenTelemetryCollectorReconciler) GetOwnedResourceTypes() []client.Object {
	ownedResources := []client.Object{
		&corev1.ConfigMap{},
		&corev1.ServiceAccount{},
		&corev1.Service{},
		&appsv1.Deployment{},
		&appsv1.DaemonSet{},
		&appsv1.StatefulSet{},
		&networkingv1.Ingress{},
		&autoscalingv2.HorizontalPodAutoscaler{},
		&policyV1.PodDisruptionBudget{},
	}

	if r.config.CreateRBACPermissions() == rbac.Available {
		ownedResources = append(ownedResources, &rbacv1.ClusterRole{})
		ownedResources = append(ownedResources, &rbacv1.ClusterRoleBinding{})
		ownedResources = append(ownedResources, &rbacv1.Role{})
		ownedResources = append(ownedResources, &rbacv1.RoleBinding{})
	}

	if featuregate.PrometheusOperatorIsAvailable.IsEnabled() && r.config.PrometheusCRAvailability() == prometheus.Available {
		ownedResources = append(ownedResources, &monitoringv1.PodMonitor{})
		ownedResources = append(ownedResources, &monitoringv1.ServiceMonitor{})
	}

	if r.config.OpenShiftRoutesAvailability() == openshift.RoutesAvailable {
		ownedResources = append(ownedResources, &routev1.Route{})
	}

	return ownedResources
}

const collectorFinalizer = "opentelemetrycollector.opentelemetry.io/finalizer"

func (r *OpenTelemetryCollectorReconciler) finalizeCollector(ctx context.Context, params manifests.Params) error {
	// The cluster scope objects do not have owner reference. They need to be deleted explicitly
	if params.Config.CreateRBACPermissions() == rbac.Available {
		objects, err := r.findRBACObjects(ctx, params)
		if err != nil {
			return err
		}
		return deleteObjects(ctx, r.Client, r.log, objects)
	}
	return nil
}
