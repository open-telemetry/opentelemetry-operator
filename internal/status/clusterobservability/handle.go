// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clusterobservability

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/clusterobservability/config"
)

const (
	reasonError         = "Error"
	reasonStatusFailure = "StatusFailure"
	reasonInfo          = "Info"
	reasonReady         = "Ready"
	reasonConfigured    = "Configured"

	// Component status keys.
	componentAgentCollector   = "agent"
	componentClusterCollector = "cluster"
	componentInstrumentation  = "instrumentation"
)

// HandleReconcileStatus handles updating the status of the ClusterObservability CRD.
func HandleReconcileStatus(ctx context.Context, log logr.Logger, params manifests.Params, err error) (ctrl.Result, error) {
	log.V(2).Info("updating cluster observability status")

	changed := params.ClusterObservability.DeepCopy()

	// Check if this is a conflict error
	isConflicted := err != nil && isConflictError(err)

	if err != nil && !isConflicted {
		params.Recorder.Event(&params.ClusterObservability, corev1.EventTypeWarning, reasonError, err.Error())
		return ctrl.Result{}, err
	}

	// Update component status and overall status
	updateClusterObservabilityStatus(ctx, log, params.Client, changed, isConflicted)

	statusPatch := client.MergeFrom(&params.ClusterObservability)
	if err := params.Client.Status().Patch(ctx, changed, statusPatch); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to apply status changes to the ClusterObservability CR: %w", err)
	}

	if isConflicted {
		params.Recorder.Event(changed, corev1.EventTypeNormal, reasonInfo, "status updated - resource is conflicted")
		return ctrl.Result{}, nil // No need to requeue - we watch for changes
	}

	params.Recorder.Event(changed, corev1.EventTypeNormal, reasonInfo, "applied status changes")
	return ctrl.Result{}, nil
}

// isConflictError checks if the error indicates a conflict situation.
func isConflictError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "multiple ClusterObservability resources detected")
}

// updateClusterObservabilityStatus updates the status of ClusterObservability based on component health.
func updateClusterObservabilityStatus(ctx context.Context, log logr.Logger, cli client.Client, co *v1alpha1.ClusterObservability, isConflicted bool) {
	// Initialize ComponentsStatus if nil
	if co.Status.ComponentsStatus == nil {
		co.Status.ComponentsStatus = make(map[string]v1alpha1.ComponentStatus)
	}

	now := metav1.Now()

	if isConflicted {
		// Resource is conflicted, set appropriate status
		co.Status.Phase = "Conflicted"
		co.Status.Message = "Multiple ClusterObservability resources detected. Only the oldest resource is active."

		// Set conflicted condition
		conflictedCondition := findCondition(co.Status.Conditions, v1alpha1.ClusterObservabilityConditionConflicted)
		if conflictedCondition == nil {
			co.Status.Conditions = append(co.Status.Conditions, v1alpha1.ClusterObservabilityCondition{
				Type:               v1alpha1.ClusterObservabilityConditionConflicted,
				Status:             metav1.ConditionTrue,
				LastTransitionTime: now,
				Reason:             reasonConfigured,
				Message:            "Multiple ClusterObservability resources exist in cluster",
			})
		} else if conflictedCondition.Status != metav1.ConditionTrue {
			conflictedCondition.Status = metav1.ConditionTrue
			conflictedCondition.Message = "Multiple ClusterObservability resources exist in cluster"
			conflictedCondition.LastTransitionTime = now
		}

		// Set ready condition to false
		readyCondition := findCondition(co.Status.Conditions, v1alpha1.ClusterObservabilityConditionReady)
		if readyCondition != nil && readyCondition.Status != metav1.ConditionFalse {
			readyCondition.Status = metav1.ConditionFalse
			readyCondition.Message = "Resource is conflicted - multiple instances detected"
			readyCondition.LastTransitionTime = now
		}

		// Update observed generation and return
		co.Status.ObservedGeneration = co.Generation
		return
	}

	// Remove conflicted condition if it exists (no longer conflicted)
	conflictedCondition := findCondition(co.Status.Conditions, v1alpha1.ClusterObservabilityConditionConflicted)
	if conflictedCondition != nil && conflictedCondition.Status == metav1.ConditionTrue {
		conflictedCondition.Status = metav1.ConditionFalse
		conflictedCondition.Message = "No conflicts detected"
		conflictedCondition.LastTransitionTime = now
	}

	// Check agent collector status (DaemonSet)
	agentCollectorStatus := checkAgentCollectorStatus(ctx, cli, co)
	co.Status.ComponentsStatus[componentAgentCollector] = v1alpha1.ComponentStatus{
		Ready:       agentCollectorStatus.ready,
		Message:     agentCollectorStatus.message,
		LastUpdated: now,
	}

	// Check cluster collector status (Deployment)
	clusterCollectorStatus := checkClusterCollectorStatus(ctx, cli, co)
	co.Status.ComponentsStatus[componentClusterCollector] = v1alpha1.ComponentStatus{
		Ready:       clusterCollectorStatus.ready,
		Message:     clusterCollectorStatus.message,
		LastUpdated: now,
	}

	// Check instrumentation status
	instrumentationStatus := checkInstrumentationStatus(ctx, cli, co)
	co.Status.ComponentsStatus[componentInstrumentation] = v1alpha1.ComponentStatus{
		Ready:       instrumentationStatus.ready,
		Message:     instrumentationStatus.message,
		LastUpdated: now,
	}

	// Update overall status based on component status
	allReady := agentCollectorStatus.ready && clusterCollectorStatus.ready && instrumentationStatus.ready

	// Update conditions
	configuredCondition := findCondition(co.Status.Conditions, v1alpha1.ClusterObservabilityConditionConfigured)
	if configuredCondition == nil {
		co.Status.Conditions = append(co.Status.Conditions, v1alpha1.ClusterObservabilityCondition{
			Type:               v1alpha1.ClusterObservabilityConditionConfigured,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: now,
			Reason:             reasonConfigured,
			Message:            "ClusterObservability configuration applied successfully",
		})
	}

	readyCondition := findCondition(co.Status.Conditions, v1alpha1.ClusterObservabilityConditionReady)
	if readyCondition == nil && allReady {
		co.Status.Conditions = append(co.Status.Conditions, v1alpha1.ClusterObservabilityCondition{
			Type:               v1alpha1.ClusterObservabilityConditionReady,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: now,
			Reason:             reasonReady,
			Message:            "All ClusterObservability components are ready",
		})
	} else if readyCondition != nil {
		// Update existing ready condition
		newStatus := metav1.ConditionTrue
		message := "All ClusterObservability components are ready"
		if !allReady {
			newStatus = metav1.ConditionFalse
			message = "Some ClusterObservability components are not ready"
		}

		if readyCondition.Status != newStatus {
			readyCondition.Status = newStatus
			readyCondition.Message = message
			readyCondition.LastTransitionTime = now
		}
	}

	// Set phase based on conditions
	if allReady {
		co.Status.Phase = "Ready"
		co.Status.Message = "All components are ready and collecting observability data"
	} else {
		co.Status.Phase = "Pending"
		co.Status.Message = "Some components are not ready"
	}

	// Update config versions to track changes
	if err := updateConfigVersions(co); err != nil {
		// Log warning but don't fail reconciliation
		log.Error(err, "Failed to update config versions")
	}

	// Update observed generation
	co.Status.ObservedGeneration = co.Generation
}

type componentStatus struct {
	ready   bool
	message string
}

// checkAgentCollectorStatus checks the status of the agent collector DaemonSet.
func checkAgentCollectorStatus(ctx context.Context, cli client.Client, co *v1alpha1.ClusterObservability) componentStatus {
	agentCollectorName := fmt.Sprintf("%s-agent", co.Name)

	// Check OpenTelemetryCollector CR status
	var agentCollector v1beta1.OpenTelemetryCollector
	collectorKey := types.NamespacedName{Name: agentCollectorName, Namespace: co.Namespace}

	if err := cli.Get(ctx, collectorKey, &agentCollector); err != nil {
		if apierrors.IsNotFound(err) {
			return componentStatus{
				ready:   false,
				message: "Agent collector OpenTelemetryCollector not found",
			}
		}
		return componentStatus{
			ready:   false,
			message: fmt.Sprintf("Failed to get agent collector: %v", err),
		}
	}

	// Check underlying DaemonSet status
	var daemonSet appsv1.DaemonSet
	dsKey := types.NamespacedName{Name: agentCollectorName + "-collector", Namespace: co.Namespace}

	if err := cli.Get(ctx, dsKey, &daemonSet); err != nil {
		if apierrors.IsNotFound(err) {
			return componentStatus{
				ready:   false,
				message: "Agent collector DaemonSet not found",
			}
		}
		return componentStatus{
			ready:   false,
			message: fmt.Sprintf("Failed to get agent collector DaemonSet: %v", err),
		}
	}

	// Check if DaemonSet is ready
	if daemonSet.Status.DesiredNumberScheduled == 0 {
		return componentStatus{
			ready:   false,
			message: "Agent collector DaemonSet has no scheduled pods",
		}
	}

	if daemonSet.Status.NumberReady != daemonSet.Status.DesiredNumberScheduled {
		return componentStatus{
			ready: false,
			message: fmt.Sprintf("Agent collector DaemonSet not ready: %d/%d pods ready",
				daemonSet.Status.NumberReady, daemonSet.Status.DesiredNumberScheduled),
		}
	}

	return componentStatus{
		ready: true,
		message: fmt.Sprintf("Agent collector DaemonSet ready: %d/%d pods ready",
			daemonSet.Status.NumberReady, daemonSet.Status.DesiredNumberScheduled),
	}
}

// checkClusterCollectorStatus checks the status of the cluster collector Deployment.
func checkClusterCollectorStatus(ctx context.Context, cli client.Client, co *v1alpha1.ClusterObservability) componentStatus {
	clusterCollectorName := fmt.Sprintf("%s-cluster", co.Name)

	// Check OpenTelemetryCollector CR status
	var clusterCollector v1beta1.OpenTelemetryCollector
	collectorKey := types.NamespacedName{Name: clusterCollectorName, Namespace: co.Namespace}

	if err := cli.Get(ctx, collectorKey, &clusterCollector); err != nil {
		if apierrors.IsNotFound(err) {
			return componentStatus{
				ready:   false,
				message: "Cluster collector OpenTelemetryCollector not found",
			}
		}
		return componentStatus{
			ready:   false,
			message: fmt.Sprintf("Failed to get cluster collector: %v", err),
		}
	}

	// Check underlying Deployment status
	var deployment appsv1.Deployment
	deployKey := types.NamespacedName{Name: clusterCollectorName + "-collector", Namespace: co.Namespace}

	if err := cli.Get(ctx, deployKey, &deployment); err != nil {
		if apierrors.IsNotFound(err) {
			return componentStatus{
				ready:   false,
				message: "Cluster collector Deployment not found",
			}
		}
		return componentStatus{
			ready:   false,
			message: fmt.Sprintf("Failed to get cluster collector Deployment: %v", err),
		}
	}

	// Check if Deployment is ready
	if deployment.Status.Replicas == 0 {
		return componentStatus{
			ready:   false,
			message: "Cluster collector Deployment has no replicas",
		}
	}

	if deployment.Status.ReadyReplicas != deployment.Status.Replicas {
		return componentStatus{
			ready: false,
			message: fmt.Sprintf("Cluster collector Deployment not ready: %d/%d replicas ready",
				deployment.Status.ReadyReplicas, deployment.Status.Replicas),
		}
	}

	return componentStatus{
		ready: true,
		message: fmt.Sprintf("Cluster collector Deployment ready: %d/%d replicas ready",
			deployment.Status.ReadyReplicas, deployment.Status.Replicas),
	}
}

// checkInstrumentationStatus checks the status of the single Instrumentation CR.
func checkInstrumentationStatus(ctx context.Context, cli client.Client, co *v1alpha1.ClusterObservability) componentStatus {
	instrumentationName := co.Name

	// Check Instrumentation CR in the same namespace as ClusterObservability
	var instrumentation v1alpha1.Instrumentation
	instrKey := types.NamespacedName{Name: instrumentationName, Namespace: co.Namespace}

	if err := cli.Get(ctx, instrKey, &instrumentation); err != nil {
		if apierrors.IsNotFound(err) {
			return componentStatus{
				ready:   false,
				message: fmt.Sprintf("Instrumentation CR not found: %s/%s", co.Namespace, instrumentationName),
			}
		}
		return componentStatus{
			ready:   false,
			message: fmt.Sprintf("Failed to get Instrumentation CR: %v", err),
		}
	}

	// Check if instrumentation is managed by our ClusterObservability
	if !isOwnedByClusterObservability(&instrumentation, co) {
		return componentStatus{
			ready:   false,
			message: fmt.Sprintf("Instrumentation CR %s/%s is not managed by this ClusterObservability", co.Namespace, instrumentationName),
		}
	}

	return componentStatus{
		ready:   true,
		message: fmt.Sprintf("Instrumentation CR ready: %s/%s", co.Namespace, instrumentationName),
	}
}

// isOwnedByClusterObservability checks if an instrumentation is managed by the given ClusterObservability instance.
func isOwnedByClusterObservability(obj client.Object, instance *v1alpha1.ClusterObservability) bool {
	labels := obj.GetLabels()
	if labels == nil {
		return false
	}

	if managedBy, ok := labels["app.kubernetes.io/managed-by"]; !ok || managedBy != "opentelemetry-operator" {
		return false
	}

	if component, ok := labels["app.kubernetes.io/component"]; !ok || component != "cluster-observability" {
		return false
	}

	for _, owner := range obj.GetOwnerReferences() {
		if owner.UID == instance.UID {
			return true
		}
	}

	return false
}

// updateConfigVersions updates the config version tracking in the ClusterObservability status.
func updateConfigVersions(co *v1alpha1.ClusterObservability) error {
	configLoader := config.NewConfigLoader()

	// Get current config versions
	currentVersions, err := configLoader.GetAllConfigVersions()
	if err != nil {
		return fmt.Errorf("failed to get current config versions: %w", err)
	}

	// Initialize ConfigVersions map if nil
	if co.Status.ConfigVersions == nil {
		co.Status.ConfigVersions = make(map[string]string)
	}

	// Check if any config versions have changed
	configChanged := false
	for versionKey, currentVersion := range currentVersions {
		if existingVersion, exists := co.Status.ConfigVersions[versionKey]; exists {
			if configLoader.CompareConfigVersions(existingVersion, currentVersion) {
				configChanged = true
				break
			}
		} else {
			// New version key (first time or new distro added)
			configChanged = true
		}
	}

	// Update all config versions
	co.Status.ConfigVersions = currentVersions

	// If config changed, add a condition indicating config update
	if configChanged {
		now := metav1.Now()
		configCondition := findCondition(co.Status.Conditions, "ConfigurationUpdated")
		if configCondition == nil {
			co.Status.Conditions = append(co.Status.Conditions, v1alpha1.ClusterObservabilityCondition{
				Type:               "ConfigurationUpdated",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: now,
				Reason:             "ConfigChanged",
				Message:            "Collector configuration has been updated - managed collectors will be reconciled",
			})
		} else {
			configCondition.Status = metav1.ConditionTrue
			configCondition.LastTransitionTime = now
			configCondition.Reason = "ConfigChanged"
			configCondition.Message = "Collector configuration has been updated - managed collectors will be reconciled"
		}
	}

	return nil
}

// DetectConfigChanges returns true if the config versions in the status differ from current embedded configs.
// This is used to trigger reconciliation when operator is upgraded with new configs.
func DetectConfigChanges(co *v1alpha1.ClusterObservability) (bool, error) {
	if co.Status.ConfigVersions == nil {
		// First time - consider it a change
		return true, nil
	}

	configLoader := config.NewConfigLoader()
	currentVersions, err := configLoader.GetAllConfigVersions()
	if err != nil {
		return false, fmt.Errorf("failed to get current config versions: %w", err)
	}

	// Check if any versions differ
	for versionKey, currentVersion := range currentVersions {
		if existingVersion, exists := co.Status.ConfigVersions[versionKey]; !exists || configLoader.CompareConfigVersions(existingVersion, currentVersion) {
			return true, nil
		}
	}

	// Check if any old versions are no longer present (distro removed)
	for versionKey := range co.Status.ConfigVersions {
		if _, exists := currentVersions[versionKey]; !exists {
			return true, nil
		}
	}

	return false, nil
}

// findCondition finds a condition by type in the conditions slice.
func findCondition(conditions []v1alpha1.ClusterObservabilityCondition, condType v1alpha1.ClusterObservabilityConditionType) *v1alpha1.ClusterObservabilityCondition {
	for i := range conditions {
		if conditions[i].Type == condType {
			return &conditions[i]
		}
	}
	return nil
}
