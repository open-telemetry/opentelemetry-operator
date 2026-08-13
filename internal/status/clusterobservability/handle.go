// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clusterobservability

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/clusterobservability/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

const (
	reasonReady              = "Ready"
	reasonConfigured         = "Configured"
	reasonReconcileError     = "ReconcileError"
	reasonComponentsNotReady = "ComponentsNotReady"
	reasonRolloutProgressing = "RolloutProgressing"
	failureObservationWindow = 5 * time.Second

	// Component status keys.
	componentAgentCollector   = "agent"
	componentClusterCollector = "cluster"
	componentInstrumentation  = "instrumentation"
)

// HandleReconcileStatus handles updating the status of the ClusterObservability CRD.
func HandleReconcileStatus(ctx context.Context, log logr.Logger, params manifests.Params, err error) (ctrl.Result, error) {
	log.V(2).Info("updating cluster observability status")

	previous := params.ClusterObservability.DeepCopy()
	changed := params.ClusterObservability.DeepCopy()

	// Check if this is a conflict error
	isConflicted := err != nil && isConflictError(err)

	// Update component status and overall status
	requeueAfter := updateClusterObservabilityStatus(ctx, log, params.Client, changed, isConflicted, err)

	if !apiequality.Semantic.DeepEqual(previous.Status, changed.Status) {
		statusPatch := client.MergeFromWithOptions(&params.ClusterObservability, client.MergeFromWithOptimisticLock{})
		if statusErr := params.Client.Status().Patch(ctx, changed, statusPatch); statusErr != nil {
			if apierrors.IsConflict(statusErr) {
				// SetupWithManager watches all ClusterObservability updates, so the
				// conflicting write will enqueue the latest state for reconciliation.
				log.V(2).Info("ClusterObservability status changed before update; latest state will be reconciled")
				if isConflicted {
					return ctrl.Result{}, nil
				}
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, fmt.Errorf("failed to apply status changes to the ClusterObservability CR: %w", statusErr)
		}
	}

	emitStatusEvents(params, previous, changed, err, isConflicted)

	if isConflicted {
		return ctrl.Result{}, nil // No need to requeue - we watch for changes
	}
	if err == nil && requeueAfter > 0 {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	return ctrl.Result{}, err
}

// isConflictError checks if the error indicates a conflict situation.
func isConflictError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "multiple ClusterObservability resources detected")
}

// updateClusterObservabilityStatus updates the status of ClusterObservability based on component health.
func updateClusterObservabilityStatus(
	ctx context.Context,
	log logr.Logger,
	cli client.Client,
	co *v1alpha1.ClusterObservability,
	isConflicted bool,
	reconcileErr error,
) time.Duration {
	generationChanged := co.Status.ObservedGeneration != co.Generation
	previousComponents := make(map[string]v1alpha1.ComponentStatus, len(co.Status.ComponentsStatus))
	maps.Copy(previousComponents, co.Status.ComponentsStatus)

	// Initialize ComponentsStatus if nil
	if co.Status.ComponentsStatus == nil {
		co.Status.ComponentsStatus = make(map[string]v1alpha1.ComponentStatus)
	}

	if isConflicted {
		// Resource is conflicted, set appropriate status
		co.Status.Phase = "Conflicted"
		co.Status.Message = "Multiple ClusterObservability resources detected. Only the oldest resource is active."

		// Set conflicted condition
		setCondition(co, v1alpha1.ClusterObservabilityCondition{
			Type:               v1alpha1.ClusterObservabilityConditionConflicted,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: co.Generation,
			Reason:             "MultipleInstances",
			Message:            "Multiple ClusterObservability resources exist in cluster",
		})
		setCondition(co, v1alpha1.ClusterObservabilityCondition{
			Type:               v1alpha1.ClusterObservabilityConditionConfigured,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: co.Generation,
			Reason:             "MultipleInstances",
			Message:            "Only the oldest ClusterObservability resource is reconciled",
		})
		setCondition(co, v1alpha1.ClusterObservabilityCondition{
			Type:               v1alpha1.ClusterObservabilityConditionReady,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: co.Generation,
			Reason:             "MultipleInstances",
			Message:            "Resource is conflicted - multiple instances detected",
		})

		// Update observed generation and return
		co.Status.ObservedGeneration = co.Generation
		return 0
	}

	// Remove conflicted condition if it exists (no longer conflicted)
	if findCondition(co.Status.Conditions, v1alpha1.ClusterObservabilityConditionConflicted) != nil {
		setCondition(co, v1alpha1.ClusterObservabilityCondition{
			Type:               v1alpha1.ClusterObservabilityConditionConflicted,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: co.Generation,
			Reason:             "SingleInstance",
			Message:            "No conflicts detected",
		})
	}

	// Check agent collector status (DaemonSet)
	agentCollectorStatus := checkAgentCollectorStatus(ctx, cli, co)
	setComponentStatus(co, componentAgentCollector, agentCollectorStatus, generationChanged)

	// Check cluster collector status.
	clusterCollectorStatus := checkClusterCollectorStatus(ctx, cli, co)
	setComponentStatus(co, componentClusterCollector, clusterCollectorStatus, generationChanged)

	// Check instrumentation status
	instrumentationStatus := checkInstrumentationStatus(ctx, cli, co)
	setComponentStatus(co, componentInstrumentation, instrumentationStatus, generationChanged)

	// Do not report a rollout failure until its current generation is observed.
	allReady := agentCollectorStatus.ready && clusterCollectorStatus.ready && instrumentationStatus.ready
	anyFailed := (!agentCollectorStatus.ready && !agentCollectorStatus.progressing) ||
		(!clusterCollectorStatus.ready && !clusterCollectorStatus.progressing) ||
		(!instrumentationStatus.ready && !instrumentationStatus.progressing)
	previousReady := findCondition(co.Status.Conditions, v1alpha1.ClusterObservabilityConditionReady)
	failureAlreadyKnown := !allReady && previousReady != nil &&
		previousReady.Status == metav1.ConditionFalse &&
		previousReady.ObservedGeneration == co.Generation &&
		previousReady.Reason == reasonComponentsNotReady
	failureConfirmed, requeueAfter := confirmComponentFailure(previousComponents, generationChanged,
		agentCollectorStatus, clusterCollectorStatus, instrumentationStatus)
	failureKnownForGeneration := failureAlreadyKnown || failureConfirmed

	// Update conditions
	if reconcileErr != nil {
		setCondition(co, v1alpha1.ClusterObservabilityCondition{
			Type:               v1alpha1.ClusterObservabilityConditionConfigured,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: co.Generation,
			Reason:             reasonReconcileError,
			Message:            reconcileErr.Error(),
		})
	} else {
		setCondition(co, v1alpha1.ClusterObservabilityCondition{
			Type:               v1alpha1.ClusterObservabilityConditionConfigured,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: co.Generation,
			Reason:             reasonConfigured,
			Message:            "ClusterObservability configuration applied successfully",
		})
	}

	switch {
	case allReady:
		setCondition(co, v1alpha1.ClusterObservabilityCondition{
			Type:               v1alpha1.ClusterObservabilityConditionReady,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: co.Generation,
			Reason:             reasonReady,
			Message:            "All ClusterObservability components are ready",
		})
	case !failureKnownForGeneration:
		setCondition(co, v1alpha1.ClusterObservabilityCondition{
			Type:               v1alpha1.ClusterObservabilityConditionReady,
			Status:             metav1.ConditionUnknown,
			ObservedGeneration: co.Generation,
			Reason:             reasonRolloutProgressing,
			Message:            "ClusterObservability component rollout is progressing",
		})
	default:
		setCondition(co, v1alpha1.ClusterObservabilityCondition{
			Type:               v1alpha1.ClusterObservabilityConditionReady,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: co.Generation,
			Reason:             reasonComponentsNotReady,
			Message:            "Some ClusterObservability components are not ready",
		})
	}

	// Set phase based on conditions
	switch {
	case reconcileErr != nil && allReady:
		co.Status.Phase = "Degraded"
		co.Status.Message = "Managed components are ready, but the latest configuration could not be reconciled"
	case reconcileErr != nil:
		co.Status.Phase = "Failed"
		co.Status.Message = "The latest configuration could not be reconciled and some components are not ready"
	case allReady:
		co.Status.Phase = "Ready"
		co.Status.Message = "All components are ready and collecting observability data"
	case failureKnownForGeneration:
		co.Status.Phase = "Degraded"
		co.Status.Message = "One or more components failed to become ready"
	default:
		co.Status.Phase = "Pending"
		co.Status.Message = "Some components are not ready"
	}

	// Update config versions to track changes
	if reconcileErr == nil {
		if err := updateConfigVersions(co); err != nil {
			// Log warning but don't fail reconciliation
			log.Error(err, "Failed to update config versions")
		}
	}

	// Update observed generation
	co.Status.ObservedGeneration = co.Generation
	if failureAlreadyKnown || !anyFailed {
		return 0
	}
	return requeueAfter
}

func setComponentStatus(co *v1alpha1.ClusterObservability, component string, status componentStatus, forceUpdate bool) {
	previous, found := co.Status.ComponentsStatus[component]
	if !forceUpdate && found && previous.Ready == status.ready && previous.Message == status.message {
		return
	}
	co.Status.ComponentsStatus[component] = v1alpha1.ComponentStatus{
		Ready:       status.ready,
		Message:     status.message,
		LastUpdated: metav1.Now(),
	}
}

func confirmComponentFailure(
	previous map[string]v1alpha1.ComponentStatus,
	generationChanged bool,
	statuses ...componentStatus,
) (bool, time.Duration) {
	components := []string{componentAgentCollector, componentClusterCollector, componentInstrumentation}
	now := time.Now()
	requeueAfter := failureObservationWindow
	for index, status := range statuses {
		if status.ready || status.progressing {
			continue
		}
		prior, found := previous[components[index]]
		if generationChanged || !found || prior.Ready || prior.Message != status.message {
			continue
		}
		remaining := failureObservationWindow - now.Sub(prior.LastUpdated.Time)
		if remaining <= 0 {
			return true, 0
		}
		if remaining < requeueAfter {
			requeueAfter = remaining
		}
	}
	return false, requeueAfter
}

func setCondition(co *v1alpha1.ClusterObservability, condition v1alpha1.ClusterObservabilityCondition) {
	existing := findCondition(co.Status.Conditions, condition.Type)
	if existing == nil {
		condition.LastTransitionTime = metav1.Now()
		co.Status.Conditions = append(co.Status.Conditions, condition)
		return
	}

	if existing.Status == condition.Status {
		condition.LastTransitionTime = existing.LastTransitionTime
	} else {
		condition.LastTransitionTime = metav1.Now()
	}
	*existing = condition
}

func emitStatusEvents(
	params manifests.Params,
	previous *v1alpha1.ClusterObservability,
	changed *v1alpha1.ClusterObservability,
	reconcileErr error,
	isConflicted bool,
) {
	if params.Recorder == nil || isConflicted {
		return
	}

	previousConfigured := findCondition(previous.Status.Conditions, v1alpha1.ClusterObservabilityConditionConfigured)
	if reconcileErr != nil {
		if previousConfigured == nil || previousConfigured.Status != metav1.ConditionFalse ||
			previousConfigured.ObservedGeneration != changed.Generation || previousConfigured.Message != reconcileErr.Error() {
			params.Recorder.Eventf(changed, nil, corev1.EventTypeWarning, reasonReconcileError, "Reconcile", reconcileErr.Error())
		}
		return
	}

	previousReady := findCondition(previous.Status.Conditions, v1alpha1.ClusterObservabilityConditionReady)
	ready := findCondition(changed.Status.Conditions, v1alpha1.ClusterObservabilityConditionReady)
	if ready == nil {
		return
	}
	if ready.Status == metav1.ConditionFalse && (previousReady == nil || previousReady.Status != metav1.ConditionFalse) {
		params.Recorder.Eventf(changed, nil, corev1.EventTypeWarning, reasonComponentsNotReady, "Observe",
			"One or more managed ClusterObservability components are not ready")
	}
}

type componentStatus struct {
	ready       bool
	progressing bool
	message     string
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
				ready:       false,
				progressing: true,
				message:     "Agent collector OpenTelemetryCollector not found",
			}
		}
		return componentStatus{
			ready:   false,
			message: fmt.Sprintf("Failed to get agent collector: %v", err),
		}
	}
	if status := checkCollectorReconcileStatus(&agentCollector, "Agent collector"); !status.ready {
		return status
	}

	// Check underlying DaemonSet status
	var daemonSet appsv1.DaemonSet
	dsKey := types.NamespacedName{Name: naming.Collector(agentCollectorName), Namespace: co.Namespace}

	if err := cli.Get(ctx, dsKey, &daemonSet); err != nil {
		if apierrors.IsNotFound(err) {
			return componentStatus{
				ready:       false,
				progressing: true,
				message:     "Agent collector DaemonSet not found",
			}
		}
		return componentStatus{
			ready:   false,
			message: fmt.Sprintf("Failed to get agent collector DaemonSet: %v", err),
		}
	}

	// Aggregate readiness alone can include old pods during a rollout.
	if daemonSet.Status.ObservedGeneration < daemonSet.Generation {
		return componentStatus{
			ready:       false,
			progressing: true,
			message: fmt.Sprintf("Agent collector DaemonSet rollout pending: observed generation %d, current generation %d",
				daemonSet.Status.ObservedGeneration, daemonSet.Generation),
		}
	}

	if daemonSet.Status.DesiredNumberScheduled == 0 {
		return componentStatus{
			ready:       false,
			progressing: true,
			message:     "Agent collector DaemonSet has no scheduled pods",
		}
	}
	currentRevision, revisionErr := currentDaemonSetRevision(ctx, cli, &daemonSet)
	if revisionErr != nil {
		return componentStatus{
			ready:   false,
			message: fmt.Sprintf("Failed to determine agent collector DaemonSet revision: %v", revisionErr),
		}
	}
	if currentRevision != "" {
		if failure := findWorkloadPodFailure(ctx, cli, &daemonSet, daemonSet.Spec.Selector, "Agent collector", currentRevision); failure != nil {
			return *failure
		}
	}
	if daemonSet.Status.UpdatedNumberScheduled != daemonSet.Status.DesiredNumberScheduled {
		return componentStatus{
			ready:       false,
			progressing: true,
			message: fmt.Sprintf("Agent collector DaemonSet rollout incomplete: %d/%d pods updated",
				daemonSet.Status.UpdatedNumberScheduled, daemonSet.Status.DesiredNumberScheduled),
		}
	}
	if currentRevision == "" {
		if failure := findWorkloadPodFailure(ctx, cli, &daemonSet, daemonSet.Spec.Selector, "Agent collector", ""); failure != nil {
			return *failure
		}
	}

	if daemonSet.Status.NumberReady != daemonSet.Status.DesiredNumberScheduled {
		return componentStatus{
			ready:       false,
			progressing: true,
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

// checkClusterCollectorStatus checks the status of the cluster collector StatefulSet.
func checkClusterCollectorStatus(ctx context.Context, cli client.Client, co *v1alpha1.ClusterObservability) componentStatus {
	clusterCollectorName := fmt.Sprintf("%s-cluster", co.Name)

	// Check OpenTelemetryCollector CR status
	var clusterCollector v1beta1.OpenTelemetryCollector
	collectorKey := types.NamespacedName{Name: clusterCollectorName, Namespace: co.Namespace}

	if err := cli.Get(ctx, collectorKey, &clusterCollector); err != nil {
		if apierrors.IsNotFound(err) {
			return componentStatus{
				ready:       false,
				progressing: true,
				message:     "Cluster collector OpenTelemetryCollector not found",
			}
		}
		return componentStatus{
			ready:   false,
			message: fmt.Sprintf("Failed to get cluster collector: %v", err),
		}
	}
	if status := checkCollectorReconcileStatus(&clusterCollector, "Cluster collector"); !status.ready {
		return status
	}

	var sts appsv1.StatefulSet
	stsKey := types.NamespacedName{Name: naming.Collector(clusterCollectorName), Namespace: co.Namespace}
	if err := cli.Get(ctx, stsKey, &sts); err != nil {
		if apierrors.IsNotFound(err) {
			return componentStatus{ready: false, progressing: true, message: "Cluster collector StatefulSet not found"}
		}
		return componentStatus{ready: false, message: fmt.Sprintf("Failed to get cluster collector StatefulSet: %v", err)}
	}
	if sts.Status.ObservedGeneration < sts.Generation {
		return componentStatus{
			ready:       false,
			progressing: true,
			message: fmt.Sprintf("Cluster collector StatefulSet rollout pending: observed generation %d, current generation %d",
				sts.Status.ObservedGeneration, sts.Generation),
		}
	}
	desiredReplicas := int32(1)
	if sts.Spec.Replicas != nil {
		desiredReplicas = *sts.Spec.Replicas
	}
	if desiredReplicas == 0 {
		return componentStatus{ready: false, progressing: true, message: "Cluster collector StatefulSet has no replicas"}
	}
	if sts.Status.UpdateRevision != "" {
		if failure := findWorkloadPodFailure(ctx, cli, &sts, sts.Spec.Selector, "Cluster collector", sts.Status.UpdateRevision); failure != nil {
			return *failure
		}
	}
	if sts.Status.UpdatedReplicas != desiredReplicas ||
		(sts.Status.UpdateRevision != "" && sts.Status.CurrentRevision != sts.Status.UpdateRevision) {
		return componentStatus{
			ready:       false,
			progressing: true,
			message:     fmt.Sprintf("Cluster collector StatefulSet rollout incomplete: %d/%d replicas updated", sts.Status.UpdatedReplicas, desiredReplicas),
		}
	}
	if sts.Status.UpdateRevision == "" {
		if failure := findWorkloadPodFailure(ctx, cli, &sts, sts.Spec.Selector, "Cluster collector", ""); failure != nil {
			return *failure
		}
	}
	if sts.Status.ReadyReplicas != desiredReplicas {
		return componentStatus{
			ready:       false,
			progressing: true,
			message: fmt.Sprintf("Cluster collector StatefulSet not ready: %d/%d replicas ready",
				sts.Status.ReadyReplicas, desiredReplicas),
		}
	}
	return componentStatus{
		ready: true,
		message: fmt.Sprintf("Cluster collector StatefulSet ready: %d/%d replicas ready",
			sts.Status.ReadyReplicas, desiredReplicas),
	}
}

func checkCollectorReconcileStatus(collector *v1beta1.OpenTelemetryCollector, displayName string) componentStatus {
	if collector.Status.ObservedGeneration < collector.Generation {
		return componentStatus{
			ready:       false,
			progressing: true,
			message: fmt.Sprintf("%s reconciliation pending: observed generation %d, current generation %d",
				displayName, collector.Status.ObservedGeneration, collector.Generation),
		}
	}
	if ready := meta.FindStatusCondition(collector.Status.Conditions, "Ready"); ready != nil && ready.ObservedGeneration == collector.Generation && ready.Status == metav1.ConditionFalse {
		return componentStatus{
			ready:   false,
			message: fmt.Sprintf("%s reconciliation failed: %s", displayName, ready.Message),
		}
	}
	return componentStatus{ready: true}
}

func findWorkloadPodFailure(
	ctx context.Context,
	cli client.Client,
	workload client.Object,
	labelSelector *metav1.LabelSelector,
	displayName string,
	currentRevision string,
) *componentStatus {
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return &componentStatus{message: fmt.Sprintf("Failed to build %s pod selector: %v", strings.ToLower(displayName), err)}
	}

	var pods corev1.PodList
	if err := cli.List(ctx, &pods, client.InNamespace(workload.GetNamespace()), client.MatchingLabelsSelector{Selector: selector}); err != nil {
		return &componentStatus{message: fmt.Sprintf("Failed to list %s pods: %v", strings.ToLower(displayName), err)}
	}

	for i := range pods.Items {
		pod := &pods.Items[i]
		if workload.GetUID() != "" && !metav1.IsControlledBy(pod, workload) {
			continue
		}
		if currentRevision != "" && pod.Labels[appsv1.ControllerRevisionHashLabelKey] != currentRevision {
			continue
		}
		for _, statuses := range [][]corev1.ContainerStatus{pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses} {
			for _, status := range statuses {
				if status.State.Waiting != nil && isContainerFailureReason(status.State.Waiting.Reason) {
					return &componentStatus{message: fmt.Sprintf("%s pod %s container %s is in %s",
						displayName, pod.Name, status.Name, status.State.Waiting.Reason)}
				}
				if !status.Ready && status.LastTerminationState.Terminated != nil &&
					status.LastTerminationState.Terminated.ExitCode != 0 {
					return &componentStatus{message: fmt.Sprintf("%s pod %s container %s last exited with code %d (%s)",
						displayName, pod.Name, status.Name, status.LastTerminationState.Terminated.ExitCode,
						status.LastTerminationState.Terminated.Reason)}
				}
			}
		}
	}

	return nil
}

func currentDaemonSetRevision(ctx context.Context, cli client.Client, daemonSet *appsv1.DaemonSet) (string, error) {
	var revisions appsv1.ControllerRevisionList
	if err := cli.List(ctx, &revisions, client.InNamespace(daemonSet.Namespace)); err != nil {
		return "", err
	}

	var current *appsv1.ControllerRevision
	for i := range revisions.Items {
		revision := &revisions.Items[i]
		if !metav1.IsControlledBy(revision, daemonSet) || (current != nil && revision.Revision <= current.Revision) {
			continue
		}
		current = revision
	}
	if current == nil {
		return "", nil
	}
	if hash := current.Labels[appsv1.ControllerRevisionHashLabelKey]; hash != "" {
		return hash, nil
	}

	prefix := daemonSet.Name + "-"
	if !strings.HasPrefix(current.Name, prefix) {
		return "", nil
	}
	return strings.TrimPrefix(current.Name, prefix), nil
}

func isContainerFailureReason(reason string) bool {
	switch reason {
	case "CrashLoopBackOff", "CreateContainerConfigError", "CreateContainerError", "ErrImagePull",
		"ImagePullBackOff", "InvalidImageName", "RunContainerError", "StartError":
		return true
	default:
		return false
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
				ready:       false,
				progressing: true,
				message:     fmt.Sprintf("Instrumentation CR not found: %s/%s", co.Namespace, instrumentationName),
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

	// Update all config versions
	co.Status.ConfigVersions = currentVersions

	// ConfigVersions records embedded config hashes; the condition tracks the parent generation.
	setCondition(co, v1alpha1.ClusterObservabilityCondition{
		Type:               "ConfigurationUpdated",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: co.Generation,
		Reason:             "ConfigCurrent",
		Message:            "Embedded collector configuration is current",
	})

	return nil
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
