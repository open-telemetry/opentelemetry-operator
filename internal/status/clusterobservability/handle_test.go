// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clusterobservability

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func TestUpdateClusterObservabilityStatusReady(t *testing.T) {
	const (
		name      = "cluster-obs"
		namespace = "observability"
	)
	uid := types.UID("cluster-observability-uid")
	co := &v1alpha1.ClusterObservability{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			UID:        uid,
			Generation: 3,
		},
	}

	cli := newStatusTestClient(t,
		&v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-agent", Namespace: namespace, Generation: 1},
			Status:     v1beta1.OpenTelemetryCollectorStatus{ObservedGeneration: 1},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-agent-collector", Namespace: namespace, Generation: 2},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 2,
				UpdatedNumberScheduled: 2,
				NumberReady:            2,
				ObservedGeneration:     2,
			},
		},
		&v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-cluster", Namespace: namespace, Generation: 1},
			Status:     v1beta1.OpenTelemetryCollectorStatus{ObservedGeneration: 1},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-cluster-collector", Namespace: namespace, Generation: 2},
			Spec:       appsv1.StatefulSetSpec{Replicas: ptr.To[int32](1)},
			Status: appsv1.StatefulSetStatus{
				Replicas:           1,
				ReadyReplicas:      1,
				UpdatedReplicas:    1,
				ObservedGeneration: 2,
				CurrentRevision:    "revision-2",
				UpdateRevision:     "revision-2",
			},
		},
		&v1alpha1.Instrumentation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
					"app.kubernetes.io/component":  "cluster-observability",
				},
				OwnerReferences: []metav1.OwnerReference{{UID: uid}},
			},
		},
	)

	updateClusterObservabilityStatus(context.Background(), logr.Discard(), cli, co, false, nil)

	assert.Equal(t, "Ready", co.Status.Phase)
	assert.Equal(t, co.Generation, co.Status.ObservedGeneration)
	for _, component := range []string{componentAgentCollector, componentClusterCollector, componentInstrumentation} {
		assert.Truef(t, co.Status.ComponentsStatus[component].Ready, "component %q should be ready", component)
	}

	ready := findCondition(co.Status.Conditions, v1alpha1.ClusterObservabilityConditionReady)
	require.NotNil(t, ready)
	assert.Equal(t, metav1.ConditionTrue, ready.Status)
	assert.Equal(t, reasonReady, ready.Reason)
	assert.Equal(t, co.Generation, ready.ObservedGeneration)
	configured := findCondition(co.Status.Conditions, v1alpha1.ClusterObservabilityConditionConfigured)
	require.NotNil(t, configured)
	assert.Equal(t, co.Generation, configured.ObservedGeneration)
}

func TestCollectorStatusUsesManagedWorkloadNames(t *testing.T) {
	const namespace = "observability"
	name := strings.Repeat("cluster-observability-", 3) + "test"
	agentCollectorName := name + "-agent"
	clusterCollectorName := name + "-cluster"
	require.NotEqual(t, agentCollectorName+"-collector", naming.Collector(agentCollectorName))
	require.NotEqual(t, clusterCollectorName+"-collector", naming.Collector(clusterCollectorName))

	co := &v1alpha1.ClusterObservability{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	cli := newStatusTestClient(t,
		&v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{Name: agentCollectorName, Namespace: namespace, Generation: 1},
			Status:     v1beta1.OpenTelemetryCollectorStatus{ObservedGeneration: 1},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: naming.Collector(agentCollectorName), Namespace: namespace, Generation: 1},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 1,
				UpdatedNumberScheduled: 1,
				NumberReady:            1,
				ObservedGeneration:     1,
			},
		},
		&v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{Name: clusterCollectorName, Namespace: namespace, Generation: 1},
			Status:     v1beta1.OpenTelemetryCollectorStatus{ObservedGeneration: 1},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: naming.Collector(clusterCollectorName), Namespace: namespace, Generation: 1},
			Spec:       appsv1.StatefulSetSpec{Replicas: ptr.To[int32](1)},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas:      1,
				UpdatedReplicas:    1,
				ObservedGeneration: 1,
				CurrentRevision:    "revision-1",
				UpdateRevision:     "revision-1",
			},
		},
	)

	assert.True(t, checkAgentCollectorStatus(context.Background(), cli, co).ready)
	assert.True(t, checkClusterCollectorStatus(context.Background(), cli, co).ready)
}

func TestCheckClusterCollectorStatusTreatsUnreadyStatefulSetAsProgressing(t *testing.T) {
	const (
		name      = "cluster-obs"
		namespace = "observability"
	)
	co := &v1alpha1.ClusterObservability{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	cli := newStatusTestClient(t,
		&v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-cluster", Namespace: namespace, Generation: 1},
			Status:     v1beta1.OpenTelemetryCollectorStatus{ObservedGeneration: 1},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-cluster-collector", Namespace: namespace, Generation: 2},
			Spec:       appsv1.StatefulSetSpec{Replicas: ptr.To[int32](1)},
			Status: appsv1.StatefulSetStatus{
				Replicas:           1,
				ReadyReplicas:      0,
				UpdatedReplicas:    1,
				ObservedGeneration: 2,
				CurrentRevision:    "revision-2",
				UpdateRevision:     "revision-2",
			},
		},
	)

	status := checkClusterCollectorStatus(context.Background(), cli, co)
	assert.False(t, status.ready)
	assert.True(t, status.progressing)
	assert.Equal(t, "Cluster collector StatefulSet not ready: 0/1 replicas ready", status.message)
}

func TestCheckAgentCollectorStatusDetectsIncompleteRollout(t *testing.T) {
	const (
		name      = "cluster-obs"
		namespace = "observability"
	)
	co := &v1alpha1.ClusterObservability{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	cli := newStatusTestClient(t,
		&v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-agent", Namespace: namespace, Generation: 2},
			Status:     v1beta1.OpenTelemetryCollectorStatus{ObservedGeneration: 2},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-agent-collector", Namespace: namespace, Generation: 3},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 2,
				UpdatedNumberScheduled: 1,
				NumberReady:            2,
				ObservedGeneration:     3,
			},
		},
	)

	status := checkAgentCollectorStatus(context.Background(), cli, co)
	assert.False(t, status.ready)
	assert.True(t, status.progressing)
	assert.Equal(t, "Agent collector DaemonSet rollout incomplete: 1/2 pods updated", status.message)
}

func TestUpdateClusterObservabilityStatusKeepsKnownFailureForGeneration(t *testing.T) {
	const (
		name      = "cluster-obs"
		namespace = "observability"
	)
	co := &v1alpha1.ClusterObservability{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Generation: 2},
		Status: v1alpha1.ClusterObservabilityStatus{Conditions: []v1alpha1.ClusterObservabilityCondition{{
			Type:               v1alpha1.ClusterObservabilityConditionReady,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: 2,
			Reason:             reasonComponentsNotReady,
			Message:            "Some ClusterObservability components are not ready",
		}}},
	}
	cli := newStatusTestClient(t, &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{Name: name + "-agent", Namespace: namespace, Generation: 1},
		Status: v1beta1.OpenTelemetryCollectorStatus{
			ObservedGeneration: 1,
			Conditions: []metav1.Condition{{
				Type:               "Ready",
				Status:             metav1.ConditionFalse,
				ObservedGeneration: 1,
				Reason:             reasonReconcileError,
				Message:            "collector configuration is invalid",
			}},
		},
	})

	updateClusterObservabilityStatus(context.Background(), logr.Discard(), cli, co, false, nil)

	ready := findCondition(co.Status.Conditions, v1alpha1.ClusterObservabilityConditionReady)
	require.NotNil(t, ready)
	assert.Equal(t, metav1.ConditionFalse, ready.Status)
	assert.Equal(t, reasonComponentsNotReady, ready.Reason)
	assert.Equal(t, "Degraded", co.Status.Phase)

	co.Generation++
	updateClusterObservabilityStatus(context.Background(), logr.Discard(), cli, co, false, nil)
	ready = findCondition(co.Status.Conditions, v1alpha1.ClusterObservabilityConditionReady)
	require.NotNil(t, ready)
	assert.Equal(t, metav1.ConditionUnknown, ready.Status)
	assert.Equal(t, reasonRolloutProgressing, ready.Reason)
	assert.Equal(t, "Pending", co.Status.Phase)
}

func TestUpdateClusterObservabilityStatusConfirmsPersistentFailure(t *testing.T) {
	const (
		name      = "cluster-obs"
		namespace = "observability"
	)
	co := &v1alpha1.ClusterObservability{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Generation: 2},
		Status: v1alpha1.ClusterObservabilityStatus{
			ObservedGeneration: 2,
			Conditions: []v1alpha1.ClusterObservabilityCondition{{
				Type:               v1alpha1.ClusterObservabilityConditionReady,
				Status:             metav1.ConditionTrue,
				ObservedGeneration: 2,
				Reason:             reasonReady,
			}},
		},
	}
	cli := newStatusTestClient(t, &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{Name: name + "-agent", Namespace: namespace, Generation: 1},
		Status: v1beta1.OpenTelemetryCollectorStatus{
			ObservedGeneration: 1,
			Conditions: []metav1.Condition{{
				Type:               "Ready",
				Status:             metav1.ConditionFalse,
				ObservedGeneration: 1,
				Reason:             reasonReconcileError,
				Message:            "collector configuration is invalid",
			}},
		},
	})

	requeueAfter := updateClusterObservabilityStatus(context.Background(), logr.Discard(), cli, co, false, nil)
	ready := findCondition(co.Status.Conditions, v1alpha1.ClusterObservabilityConditionReady)
	require.NotNil(t, ready)
	assert.Equal(t, metav1.ConditionUnknown, ready.Status)
	assert.Positive(t, requeueAfter)
	assert.LessOrEqual(t, requeueAfter, failureObservationWindow)

	for component, status := range co.Status.ComponentsStatus {
		status.LastUpdated = metav1.NewTime(time.Now().Add(-failureObservationWindow))
		co.Status.ComponentsStatus[component] = status
	}
	requeueAfter = updateClusterObservabilityStatus(context.Background(), logr.Discard(), cli, co, false, nil)
	ready = findCondition(co.Status.Conditions, v1alpha1.ClusterObservabilityConditionReady)
	require.NotNil(t, ready)
	assert.Equal(t, metav1.ConditionFalse, ready.Status)
	assert.Equal(t, reasonComponentsNotReady, ready.Reason)
	assert.Zero(t, requeueAfter)
}

func TestCheckAgentCollectorStatusDoesNotReportOldPodFailureDuringRollout(t *testing.T) {
	const (
		name      = "cluster-obs"
		namespace = "observability"
	)
	workloadUID := types.UID("agent-daemonset")
	labels := map[string]string{"app.kubernetes.io/name": name + "-agent"}
	oldPodLabels := map[string]string{
		"app.kubernetes.io/name":              name + "-agent",
		appsv1.ControllerRevisionHashLabelKey: "revision-old",
	}
	co := &v1alpha1.ClusterObservability{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	cli := newStatusTestClient(t,
		&v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-agent", Namespace: namespace, Generation: 2},
			Status:     v1beta1.OpenTelemetryCollectorStatus{ObservedGeneration: 2},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-agent-collector", Namespace: namespace, UID: workloadUID, Generation: 3},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{MatchLabels: labels},
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 2,
				UpdatedNumberScheduled: 1,
				ObservedGeneration:     3,
			},
		},
		&appsv1.ControllerRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:            name + "-agent-collector-revision-new",
				Namespace:       namespace,
				Labels:          map[string]string{appsv1.ControllerRevisionHashLabelKey: "revision-new"},
				OwnerReferences: []metav1.OwnerReference{{UID: workloadUID, Controller: new(true)}},
			},
			Revision: 2,
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:            name + "-agent-collector-old",
				Namespace:       namespace,
				Labels:          oldPodLabels,
				OwnerReferences: []metav1.OwnerReference{{UID: workloadUID, Controller: new(true)}},
			},
			Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
				Name:  "otc-container",
				State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
			}}},
		},
	)

	status := checkAgentCollectorStatus(context.Background(), cli, co)
	assert.False(t, status.ready)
	assert.True(t, status.progressing)
	assert.Equal(t, "Agent collector DaemonSet rollout incomplete: 1/2 pods updated", status.message)
}

func TestCheckAgentCollectorStatusReportsCurrentPodFailureDuringRollout(t *testing.T) {
	const (
		name      = "cluster-obs"
		namespace = "observability"
	)
	workloadUID := types.UID("agent-daemonset")
	labels := map[string]string{"app.kubernetes.io/name": name + "-agent"}
	currentPodLabels := map[string]string{
		"app.kubernetes.io/name":              name + "-agent",
		appsv1.ControllerRevisionHashLabelKey: "revision-new",
	}
	co := &v1alpha1.ClusterObservability{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	cli := newStatusTestClient(t,
		&v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-agent", Namespace: namespace, Generation: 2},
			Status:     v1beta1.OpenTelemetryCollectorStatus{ObservedGeneration: 2},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-agent-collector", Namespace: namespace, UID: workloadUID, Generation: 3},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{MatchLabels: labels},
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 2,
				UpdatedNumberScheduled: 1,
				NumberReady:            1,
				ObservedGeneration:     3,
			},
		},
		&appsv1.ControllerRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:            name + "-agent-collector-revision-new",
				Namespace:       namespace,
				Labels:          map[string]string{appsv1.ControllerRevisionHashLabelKey: "revision-new"},
				OwnerReferences: []metav1.OwnerReference{{UID: workloadUID, Controller: new(true)}},
			},
			Revision: 2,
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:            name + "-agent-collector-new",
				Namespace:       namespace,
				Labels:          currentPodLabels,
				OwnerReferences: []metav1.OwnerReference{{UID: workloadUID, Controller: new(true)}},
			},
			Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
				Name:  "otc-container",
				State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
			}}},
		},
	)

	status := checkAgentCollectorStatus(context.Background(), cli, co)
	assert.False(t, status.ready)
	assert.False(t, status.progressing)
	assert.Equal(t, "Agent collector pod cluster-obs-agent-collector-new container otc-container is in CrashLoopBackOff", status.message)
}

func TestCheckClusterCollectorStatusDoesNotReportOldPodFailureDuringRollout(t *testing.T) {
	const (
		name      = "cluster-obs"
		namespace = "observability"
	)
	workloadUID := types.UID("cluster-statefulset")
	labels := map[string]string{"app.kubernetes.io/name": name + "-cluster"}
	oldPodLabels := map[string]string{
		"app.kubernetes.io/name":              name + "-cluster",
		appsv1.ControllerRevisionHashLabelKey: "revision-2",
	}
	co := &v1alpha1.ClusterObservability{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	cli := newStatusTestClient(t,
		&v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-cluster", Namespace: namespace, Generation: 2},
			Status:     v1beta1.OpenTelemetryCollectorStatus{ObservedGeneration: 2},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-cluster-collector", Namespace: namespace, UID: workloadUID, Generation: 3},
			Spec: appsv1.StatefulSetSpec{
				Replicas: ptr.To[int32](1),
				Selector: &metav1.LabelSelector{MatchLabels: labels},
			},
			Status: appsv1.StatefulSetStatus{
				ObservedGeneration: 3,
				CurrentRevision:    "revision-2",
				UpdateRevision:     "revision-3",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:            name + "-cluster-collector-0",
				Namespace:       namespace,
				Labels:          oldPodLabels,
				OwnerReferences: []metav1.OwnerReference{{UID: workloadUID, Controller: new(true)}},
			},
			Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
				Name:  "otc-container",
				State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
			}}},
		},
	)

	status := checkClusterCollectorStatus(context.Background(), cli, co)
	assert.False(t, status.ready)
	assert.True(t, status.progressing)
	assert.Equal(t, "Cluster collector StatefulSet rollout incomplete: 0/1 replicas updated", status.message)
}

func TestCheckAgentCollectorStatusReportsCrashLoopBetweenBackoffIntervals(t *testing.T) {
	const (
		name      = "cluster-obs"
		namespace = "observability"
	)
	workloadUID := types.UID("agent-daemonset")
	labels := map[string]string{"app.kubernetes.io/name": name + "-agent"}
	co := &v1alpha1.ClusterObservability{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	cli := newStatusTestClient(t,
		&v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-agent", Namespace: namespace, Generation: 2},
			Status:     v1beta1.OpenTelemetryCollectorStatus{ObservedGeneration: 2},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-agent-collector", Namespace: namespace, UID: workloadUID, Generation: 3},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{MatchLabels: labels},
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 1,
				UpdatedNumberScheduled: 1,
				ObservedGeneration:     3,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name + "-agent-collector-failed",
				Namespace: namespace,
				Labels:    labels,
				OwnerReferences: []metav1.OwnerReference{{
					UID:        workloadUID,
					Controller: new(true),
				}},
			},
			Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
				Name:         "otc-container",
				RestartCount: 3,
				State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{
					StartedAt: metav1.Now(),
				}},
				LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
					ExitCode: 1,
					Reason:   "Error",
				}},
			}}},
		},
	)

	status := checkAgentCollectorStatus(context.Background(), cli, co)
	assert.False(t, status.ready)
	assert.False(t, status.progressing)
	assert.Equal(t,
		"Agent collector pod cluster-obs-agent-collector-failed container otc-container last exited with code 1 (Error)",
		status.message)
}

func TestCheckCollectorReconcileStatusReportsCurrentFailure(t *testing.T) {
	collector := &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{Generation: 2},
		Status: v1beta1.OpenTelemetryCollectorStatus{
			ObservedGeneration: 2,
			Conditions: []metav1.Condition{{
				Type:               "Ready",
				Status:             metav1.ConditionFalse,
				ObservedGeneration: 2,
				Reason:             "ReconcileError",
				Message:            "collector configuration is invalid",
			}},
		},
	}

	status := checkCollectorReconcileStatus(collector, "Agent collector")
	assert.False(t, status.ready)
	assert.False(t, status.progressing)
	assert.Equal(t, "Agent collector reconciliation failed: collector configuration is invalid", status.message)
}

func TestHandleReconcileStatusReportsFailedGenerationWithoutHidingAvailableComponents(t *testing.T) {
	const (
		name      = "cluster-obs"
		namespace = "observability"
	)
	uid := types.UID("cluster-observability-uid")
	co := &v1alpha1.ClusterObservability{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: uid, Generation: 2},
		Status: v1alpha1.ClusterObservabilityStatus{
			ObservedGeneration: 1,
			Conditions: []v1alpha1.ClusterObservabilityCondition{
				{Type: v1alpha1.ClusterObservabilityConditionConfigured, Status: metav1.ConditionTrue, ObservedGeneration: 1},
				{Type: v1alpha1.ClusterObservabilityConditionReady, Status: metav1.ConditionTrue, ObservedGeneration: 1},
			},
		},
	}
	objects := []client.Object{
		co,
		&v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-agent", Namespace: namespace, Generation: 1},
			Status:     v1beta1.OpenTelemetryCollectorStatus{ObservedGeneration: 1},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-agent-collector", Namespace: namespace, Generation: 1},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 1,
				UpdatedNumberScheduled: 1,
				NumberReady:            1,
				ObservedGeneration:     1,
			},
		},
		&v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-cluster", Namespace: namespace, Generation: 1},
			Status:     v1beta1.OpenTelemetryCollectorStatus{ObservedGeneration: 1},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-cluster-collector", Namespace: namespace, Generation: 1},
			Spec:       appsv1.StatefulSetSpec{Replicas: ptr.To[int32](1)},
			Status: appsv1.StatefulSetStatus{
				Replicas:           1,
				ReadyReplicas:      1,
				UpdatedReplicas:    1,
				ObservedGeneration: 1,
				CurrentRevision:    "revision-1",
				UpdateRevision:     "revision-1",
			},
		},
		&v1alpha1.Instrumentation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
					"app.kubernetes.io/component":  "cluster-observability",
				},
				OwnerReferences: []metav1.OwnerReference{{UID: uid}},
			},
		},
	}
	cli := newStatusTestClient(t, objects...)
	recorder := events.NewFakeRecorder(5)
	reconcileErr := errors.New("invalid managed collector configuration")

	_, err := HandleReconcileStatus(context.Background(), logr.Discard(), manifests.Params{
		Client:               cli,
		Recorder:             recorder,
		ClusterObservability: *co,
	}, reconcileErr)
	require.ErrorIs(t, err, reconcileErr)

	updated := &v1alpha1.ClusterObservability{}
	require.NoError(t, cli.Get(context.Background(), client.ObjectKeyFromObject(co), updated))
	assert.Equal(t, updated.Generation, updated.Status.ObservedGeneration)
	assert.Equal(t, "Degraded", updated.Status.Phase)
	configured := findCondition(updated.Status.Conditions, v1alpha1.ClusterObservabilityConditionConfigured)
	require.NotNil(t, configured)
	assert.Equal(t, metav1.ConditionFalse, configured.Status)
	assert.Equal(t, reasonReconcileError, configured.Reason)
	assert.Equal(t, updated.Generation, configured.ObservedGeneration)
	ready := findCondition(updated.Status.Conditions, v1alpha1.ClusterObservabilityConditionReady)
	require.NotNil(t, ready)
	assert.Equal(t, metav1.ConditionTrue, ready.Status)

	select {
	case event := <-recorder.Events:
		assert.Contains(t, event, "Warning ReconcileError")
		assert.Contains(t, event, reconcileErr.Error())
	default:
		t.Fatal("expected a reconciliation warning event")
	}
}

func TestHandleReconcileStatusHandlesStaleStatusWithoutEvents(t *testing.T) {
	const (
		name      = "cluster-obs"
		namespace = "observability"
	)
	uid := types.UID("cluster-observability-uid")
	co := &v1alpha1.ClusterObservability{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, UID: uid, Generation: 1},
		Status: v1alpha1.ClusterObservabilityStatus{
			ObservedGeneration: 1,
			ComponentsStatus: map[string]v1alpha1.ComponentStatus{
				componentAgentCollector: {
					Ready:       false,
					Message:     "Agent collector reconciliation failed: collector configuration is invalid",
					LastUpdated: metav1.NewTime(time.Now().Add(-failureObservationWindow)),
				},
			},
			Conditions: []v1alpha1.ClusterObservabilityCondition{{
				Type:               v1alpha1.ClusterObservabilityConditionReady,
				Status:             metav1.ConditionUnknown,
				ObservedGeneration: 1,
				Reason:             reasonRolloutProgressing,
			}},
		},
	}
	cli := newStatusTestClient(t,
		co,
		&v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-agent", Namespace: namespace, Generation: 1},
			Status: v1beta1.OpenTelemetryCollectorStatus{
				ObservedGeneration: 1,
				Conditions: []metav1.Condition{{
					Type:               "Ready",
					Status:             metav1.ConditionFalse,
					ObservedGeneration: 1,
					Reason:             reasonReconcileError,
					Message:            "collector configuration is invalid",
				}},
			},
		},
		&v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-cluster", Namespace: namespace, Generation: 1},
			Status:     v1beta1.OpenTelemetryCollectorStatus{ObservedGeneration: 1},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-cluster-collector", Namespace: namespace, Generation: 1},
			Spec:       appsv1.StatefulSetSpec{Replicas: ptr.To[int32](1)},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas:      1,
				UpdatedReplicas:    1,
				ObservedGeneration: 1,
				CurrentRevision:    "revision-1",
				UpdateRevision:     "revision-1",
			},
		},
		&v1alpha1.Instrumentation{ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
				"app.kubernetes.io/component":  "cluster-observability",
			},
			OwnerReferences: []metav1.OwnerReference{{UID: uid}},
		}},
	)

	first := &v1alpha1.ClusterObservability{}
	stale := &v1alpha1.ClusterObservability{}
	require.NoError(t, cli.Get(context.Background(), client.ObjectKeyFromObject(co), first))
	require.NoError(t, cli.Get(context.Background(), client.ObjectKeyFromObject(co), stale))
	recorder := events.NewFakeRecorder(5)

	result, err := HandleReconcileStatus(context.Background(), logr.Discard(), manifests.Params{
		Client:               cli,
		Recorder:             recorder,
		ClusterObservability: *first,
	}, nil)
	require.NoError(t, err)
	assert.Zero(t, result)
	select {
	case event := <-recorder.Events:
		assert.Contains(t, event, "Warning ComponentsNotReady")
	default:
		t.Fatal("expected the readiness transition event")
	}

	reconcileErr := errors.New("managed resource update failed")
	for _, tt := range []struct {
		name         string
		reconcileErr error
		wantErr      error
	}{
		{name: "successful reconciliation"},
		{name: "failed reconciliation", reconcileErr: reconcileErr, wantErr: reconcileErr},
		{name: "inactive singleton", reconcileErr: errors.New("multiple ClusterObservability resources detected")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HandleReconcileStatus(context.Background(), logr.Discard(), manifests.Params{
				Client:               cli,
				Recorder:             recorder,
				ClusterObservability: *stale,
			}, tt.reconcileErr)
			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tt.wantErr)
			}
			assert.Zero(t, result)
			select {
			case event := <-recorder.Events:
				t.Fatalf("unexpected event from stale status: %s", event)
			default:
			}
		})
	}
}

func TestEmitStatusEventsSuppressesUnchangedConditions(t *testing.T) {
	recorder := events.NewFakeRecorder(5)
	co := &v1alpha1.ClusterObservability{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-obs", Namespace: "observability", Generation: 3},
		Status: v1alpha1.ClusterObservabilityStatus{
			ObservedGeneration: 3,
			Conditions: []v1alpha1.ClusterObservabilityCondition{
				{
					Type:               v1alpha1.ClusterObservabilityConditionConfigured,
					Status:             metav1.ConditionTrue,
					ObservedGeneration: 3,
					Reason:             reasonConfigured,
				},
				{
					Type:               v1alpha1.ClusterObservabilityConditionReady,
					Status:             metav1.ConditionTrue,
					ObservedGeneration: 3,
					Reason:             reasonReady,
				},
			},
		},
	}
	params := manifests.Params{Recorder: recorder}

	emitStatusEvents(params, co, co.DeepCopy(), nil, false)
	select {
	case event := <-recorder.Events:
		t.Fatalf("unexpected event for unchanged successful status: %s", event)
	default:
	}

	reconcileErr := errors.New("managed resource update failed")
	failed := co.DeepCopy()
	configured := findCondition(failed.Status.Conditions, v1alpha1.ClusterObservabilityConditionConfigured)
	configured.Status = metav1.ConditionFalse
	configured.Reason = reasonReconcileError
	configured.Message = reconcileErr.Error()
	emitStatusEvents(params, failed, failed.DeepCopy(), reconcileErr, false)
	select {
	case event := <-recorder.Events:
		t.Fatalf("unexpected event for unchanged reconciliation failure: %s", event)
	default:
	}

	progressing := co.DeepCopy()
	findCondition(progressing.Status.Conditions, v1alpha1.ClusterObservabilityConditionReady).Status = metav1.ConditionUnknown
	emitStatusEvents(params, progressing, co.DeepCopy(), nil, false)
	select {
	case event := <-recorder.Events:
		t.Fatalf("unexpected readiness event after a progressing rollout: %s", event)
	default:
	}

	notReady := co.DeepCopy()
	findCondition(notReady.Status.Conditions, v1alpha1.ClusterObservabilityConditionReady).Status = metav1.ConditionFalse
	emitStatusEvents(params, co, notReady, nil, false)
	select {
	case event := <-recorder.Events:
		assert.Contains(t, event, "Warning ComponentsNotReady")
	default:
		t.Fatal("expected a component failure warning event")
	}

	recovered := co.DeepCopy()
	emitStatusEvents(params, notReady, recovered, nil, false)
	select {
	case event := <-recorder.Events:
		t.Fatalf("unexpected event for readiness recovery: %s", event)
	default:
	}
}

func newStatusTestClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	require.NoError(t, v1beta1.AddToScheme(scheme))
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&v1alpha1.ClusterObservability{}).
		WithObjects(objects...).
		Build()
}
