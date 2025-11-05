// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"fmt"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"github.com/open-telemetry/opentelemetry-operator/pkg/sidecar"
)

func updateCollectorStatus(ctx context.Context, cli client.Client, changed *v1beta1.OpenTelemetryCollector) error {
	if changed.Status.Version == "" {
		// a version is not set, otherwise let the upgrade mechanism take care of it!
		changed.Status.Version = version.OpenTelemetryCollector()
	}

	mode := changed.Spec.Mode

	if mode == v1beta1.ModeSidecar {
		changed.Status.Scale.Replicas = 0
		changed.Status.Scale.Selector = ""
		if err := updateSidecarStatus(ctx, cli, changed); err != nil {
			return fmt.Errorf("failed to update sidecar status: %w", err)
		}
		return nil
	}

	name := naming.Collector(changed.Name)

	// Set the scale selector
	labels := manifestutils.Labels(changed.ObjectMeta, name, changed.Spec.Image, collector.ComponentOpenTelemetryCollector, []string{})
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: labels})
	if err != nil {
		return fmt.Errorf("failed to get selector for labelSelector: %w", err)
	}
	changed.Status.Scale.Selector = selector.String()

	// Set the scale replicas
	objKey := client.ObjectKey{
		Namespace: changed.GetNamespace(),
		Name:      naming.Collector(changed.Name),
	}

	var replicas int32
	var readyReplicas int32
	var statusReplicas string
	var statusImage string

	switch mode { // nolint:exhaustive
	case v1beta1.ModeDeployment:
		obj := &appsv1.Deployment{}
		if err := cli.Get(ctx, objKey, obj); err != nil {
			return fmt.Errorf("failed to get deployment status.replicas: %w", err)
		}
		replicas = obj.Status.Replicas
		readyReplicas = obj.Status.ReadyReplicas
		statusReplicas = strconv.Itoa(int(readyReplicas)) + "/" + strconv.Itoa(int(replicas))
		statusImage = obj.Spec.Template.Spec.Containers[0].Image

	case v1beta1.ModeStatefulSet:
		obj := &appsv1.StatefulSet{}
		if err := cli.Get(ctx, objKey, obj); err != nil {
			return fmt.Errorf("failed to get statefulSet status.replicas: %w", err)
		}
		replicas = obj.Status.Replicas
		readyReplicas = obj.Status.ReadyReplicas
		statusReplicas = strconv.Itoa(int(readyReplicas)) + "/" + strconv.Itoa(int(replicas))
		statusImage = obj.Spec.Template.Spec.Containers[0].Image

	case v1beta1.ModeDaemonSet:
		obj := &appsv1.DaemonSet{}
		if err := cli.Get(ctx, objKey, obj); err != nil {
			return fmt.Errorf("failed to get daemonSet status.replicas: %w", err)
		}
		replicas = obj.Status.DesiredNumberScheduled
		readyReplicas = obj.Status.NumberReady
		statusReplicas = strconv.Itoa(int(readyReplicas)) + "/" + strconv.Itoa(int(replicas))
		statusImage = obj.Spec.Template.Spec.Containers[0].Image
	}

	changed.Status.Scale.Replicas = replicas
	changed.Status.Image = statusImage
	changed.Status.Scale.StatusReplicas = statusReplicas

	return nil
}

// updateSidecarStatus gathers information about sidecar injection and updates the status fields.
func updateSidecarStatus(ctx context.Context, cli client.Client, otelcol *v1beta1.OpenTelemetryCollector) error {
	otelcol.Status.ObservedGeneration = otelcol.Generation

	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(otelcol.Namespace),
	}

	if err := cli.List(ctx, podList, listOpts...); err != nil {
		otelcol.Status.LastInjectionError = fmt.Sprintf("Failed to list pods: %v", err)
		otelcol.Status.InjectionStatus = "Failed"
		updateSidecarConditions(otelcol, 0)
		return fmt.Errorf("failed to list pods: %w", err)
	}

	var podsWithSidecar int32
	var lastInjectionTime *metav1.Time
	var lastError string

	for _, pod := range podList.Items {
		if !podRequestsSidecar(&pod, otelcol) {
			continue
		}
		if hasSidecarContainer(&pod) {
			podsWithSidecar++
			if lastInjectionTime == nil || pod.CreationTimestamp.After(lastInjectionTime.Time) {
				lastInjectionTime = &pod.CreationTimestamp
			}
		} else {
			if podError := extractPodError(&pod); podError != "" {
				lastError = podError
			}
		}
	}

	otelcol.Status.PodsInjected = podsWithSidecar
	otelcol.Status.SidecarInjected = podsWithSidecar > 0

	if podsWithSidecar > 0 {
		otelcol.Status.InjectionStatus = "Injected"
		if lastInjectionTime != nil {
			otelcol.Status.LastInjectionTime = lastInjectionTime.Format(time.RFC3339)
		}
		otelcol.Status.LastInjectionError = ""
	} else if lastError != "" {
		otelcol.Status.InjectionStatus = "Failed"
		otelcol.Status.LastInjectionError = lastError
	} else {
		otelcol.Status.InjectionStatus = "Pending"
	}

	updateSidecarConditions(otelcol, podsWithSidecar)

	return nil
}

// podRequestsSidecar checks if a pod has annotation requesting a sidecar for this collector.
func podRequestsSidecar(pod *corev1.Pod, otelcol *v1beta1.OpenTelemetryCollector) bool {
	annValue, hasAnn := pod.Annotations[sidecar.Annotation]
	if !hasAnn {
		return false
	}

	if annValue == "false" {
		return false
	}

	// Check if annotation targets as per otelcollector
	return annValue == "true" ||
		annValue == otelcol.Name ||
		annValue == otelcol.Namespace+"/"+otelcol.Name
}

// hasSidecarContainer checks if the pod actually has the sidecar container running.
func hasSidecarContainer(pod *corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		if container.Name == "otc-container" {
			return true
		}
	}

	return false
}

// extractPodError extracts error messages from pod status indicating why injection failed.
func extractPodError(pod *corev1.Pod) string {
	for _, condition := range pod.Status.Conditions {
		if condition.Status == corev1.ConditionFalse && condition.Reason != "" {
			if condition.Type == corev1.PodScheduled && condition.Reason == "Unschedulable" {
				return fmt.Sprintf("Pod %s: %s", pod.Name, condition.Message)
			}
		}
	}

	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Waiting != nil && containerStatus.State.Waiting.Reason != "" {
			reason := containerStatus.State.Waiting.Reason
			if reason == "ImagePullBackOff" || reason == "ErrImagePull" || reason == "CreateContainerError" {
				return fmt.Sprintf("Pod %s: Container %s: %s", pod.Name, containerStatus.Name, containerStatus.State.Waiting.Message)
			}
		}
	}

	if pod.Status.Phase == corev1.PodFailed && pod.Status.Message != "" {
		return fmt.Sprintf("Pod %s failed: %s", pod.Name, pod.Status.Message)
	}

	return ""
}

// updateSidecarConditions updates the conditions in the status based on injection state.
func updateSidecarConditions(otelcol *v1beta1.OpenTelemetryCollector, podsInjected int32) {
	now := metav1.Now()

	injectionCondition := metav1.Condition{
		Type:               "SidecarInjected",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: otelcol.Generation,
		LastTransitionTime: now,
		Reason:             "SidecarActive",
		Message:            fmt.Sprintf("Sidecar injected into %d pod(s)", podsInjected),
	}

	if podsInjected == 0 {
		injectionCondition.Status = metav1.ConditionFalse
		injectionCondition.Reason = "NoPods"
		injectionCondition.Message = "No pods found with sidecar injection"
	}

	updated := false
	for i, condition := range otelcol.Status.Conditions {
		if condition.Type == "SidecarInjected" {
			// Only update if status changed no need to update timestamp otherwise
			if condition.Status != injectionCondition.Status {
				otelcol.Status.Conditions[i] = injectionCondition
			} else {
				injectionCondition.LastTransitionTime = condition.LastTransitionTime
				injectionCondition.Message = fmt.Sprintf("Sidecar injected into %d pod(s)", podsInjected)
				otelcol.Status.Conditions[i] = injectionCondition
			}
			updated = true
			break
		}
	}

	if !updated {
		otelcol.Status.Conditions = append(otelcol.Status.Conditions, injectionCondition)
	}
}
