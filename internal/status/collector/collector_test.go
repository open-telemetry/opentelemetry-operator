// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func TestUpdateCollectorStatusUnsupported(t *testing.T) {
	ctx := context.TODO()
	cli := client.Client(fake.NewFakeClient())

	changed := &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sidecar",
			Namespace: "default",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode: v1beta1.ModeSidecar,
		},
	}

	err := updateCollectorStatus(ctx, cli, changed)
	assert.NoError(t, err)

	assert.Equal(t, int32(0), changed.Status.Scale.Replicas, "expected replicas to be 0")
	assert.Equal(t, "", changed.Status.Scale.Selector, "expected selector to be empty")
}

func createMockKubernetesClientDeployment() client.Client {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment-collector",
			Namespace: "default",
		},
		Status: appsv1.DeploymentStatus{
			Replicas:      1,
			ReadyReplicas: 1,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "app:latest",
						},
					},
				},
			},
		},
	}
	return fake.NewClientBuilder().WithObjects(deployment).Build()
}

func TestUpdateCollectorStatusDeploymentMode(t *testing.T) {
	ctx := context.TODO()
	cli := createMockKubernetesClientDeployment()

	changed := &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode: v1beta1.ModeDeployment,
		},
	}

	err := updateCollectorStatus(ctx, cli, changed)
	assert.NoError(t, err)

	assert.Equal(t, int32(1), changed.Status.Scale.Replicas, "expected replicas to be 1")
	assert.Equal(t, "1/1", changed.Status.Scale.StatusReplicas, "expected status replicas to be 1/1")
	assert.Equal(t, "app:latest", changed.Status.Image, "expected image to be app:latest")
}

func createMockKubernetesClientStatefulset() client.Client {
	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-statefulset-collector",
			Namespace: "default",
		},
		Status: appsv1.StatefulSetStatus{
			Replicas:      1,
			ReadyReplicas: 1,
		},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "app:latest",
						},
					},
				},
			},
		},
	}
	return fake.NewClientBuilder().WithObjects(statefulset).Build()
}

func TestUpdateCollectorStatusStatefulset(t *testing.T) {
	ctx := context.TODO()
	cli := createMockKubernetesClientStatefulset()

	changed := &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-statefulset",
			Namespace: "default",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode: v1beta1.ModeStatefulSet,
		},
	}

	err := updateCollectorStatus(ctx, cli, changed)
	assert.NoError(t, err)

	assert.Equal(t, int32(1), changed.Status.Scale.Replicas, "expected replicas to be 1")
	assert.Equal(t, "1/1", changed.Status.Scale.StatusReplicas, "expected status replicas to be 1/1")
	assert.Equal(t, "app:latest", changed.Status.Image, "expected image to be app:latest")
}

func createMockKubernetesClientDaemonset() client.Client {
	daemonset := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-daemonset-collector",
			Namespace: "default",
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "app:latest",
						},
					},
				},
			},
		},
		Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: 1,
			NumberReady:            1,
		},
	}
	return fake.NewClientBuilder().WithObjects(daemonset).Build()
}

func TestUpdateCollectorStatusDaemonsetMode(t *testing.T) {
	ctx := context.TODO()
	cli := createMockKubernetesClientDaemonset()

	changed := &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-daemonset",
			Namespace: "default",
			Labels: map[string]string{
				"customLabel": "customValue",
			},
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode: v1beta1.ModeDaemonSet,
		},
	}

	err := updateCollectorStatus(ctx, cli, changed)
	assert.NoError(t, err)

	assert.Equal(t, int32(1), changed.Status.Scale.Replicas, "expected replicas to be 1")
	assert.Equal(t, "1/1", changed.Status.Scale.StatusReplicas, "expected status replicas to be 1/1")
	assert.Contains(t, changed.Status.Scale.Selector, "customLabel=customValue", "expected selector to contain customlabel=customValue")
	assert.Equal(t, "app:latest", changed.Status.Image, "expected image to be app:latest")
}

func TestUpdateCollectorStatusWithSideCarStatus(t *testing.T) {
	ctx := context.TODO()
	cli := client.Client(fake.NewFakeClient())

	changed := &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-sidecar",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode: v1beta1.ModeSidecar,
		},
	}

	err := updateCollectorStatus(ctx, cli, changed)
	assert.NoError(t, err)

	assert.Equal(t, int32(0), changed.Status.Scale.Replicas, "expected replicas to be 0 for sidecar mode")
	assert.Equal(t, "", changed.Status.Scale.Selector, "expected selector to be empty for sidecar mode")

	assert.Equal(t, int32(0), changed.Status.PodsInjected, "expected no pods injected initially")
	assert.False(t, changed.Status.SidecarInjected, "expected SidecarInjected to be false when no pods")
	assert.Equal(t, "Pending", changed.Status.InjectionStatus, "expected InjectionStatus to be Pending")
	assert.Equal(t, int64(1), changed.Status.ObservedGeneration, "expected ObservedGeneration to match Generation")
	assert.Empty(t, changed.Status.LastInjectionError, "expected no injection error")

	assert.NotEmpty(t, changed.Status.Conditions, "expected conditions to be set")
	assert.Equal(t, "SidecarInjected", changed.Status.Conditions[0].Type, "expected SidecarInjected condition")
	assert.Equal(t, metav1.ConditionFalse, changed.Status.Conditions[0].Status, "expected condition status to be False")
	assert.Equal(t, "NoPods", changed.Status.Conditions[0].Reason, "expected reason to be NoPods")

	statusBytes, err := json.MarshalIndent(changed.Status, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal status to JSON: %v", err)
	}
	fmt.Println(string(statusBytes))
}

func TestUpdateCollectorStatusWithSideCarStatusWithPods(t *testing.T) {
	ctx := context.TODO()

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-1",
			Namespace: "default",
			Annotations: map[string]string{
				"sidecar.opentelemetry.io/inject": "test-sidecar",
			},
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "app:latest",
				},
				{
					Name:  "otc-container",
					Image: "otel/opentelemetry-collector:latest",
				},
			},
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-2",
			Namespace: "default",
			Annotations: map[string]string{
				"sidecar.opentelemetry.io/inject": "true",
			},
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "app:latest",
				},
				{
					Name:  "otc-container",
					Image: "otel/opentelemetry-collector:latest",
				},
			},
		},
	}

	pod3 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-3",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "app:latest",
				},
			},
		},
	}

	cli := fake.NewClientBuilder().WithObjects(pod1, pod2, pod3).Build()

	changed := &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-sidecar",
			Namespace:  "default",
			Generation: 2,
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode: v1beta1.ModeSidecar,
		},
	}

	err := updateCollectorStatus(ctx, cli, changed)
	assert.NoError(t, err)

	assert.Equal(t, int32(0), changed.Status.Scale.Replicas, "expected replicas to be 0 for sidecar mode")
	assert.Equal(t, "", changed.Status.Scale.Selector, "expected selector to be empty for sidecar mode")

	assert.Equal(t, int32(2), changed.Status.PodsInjected, "expected 2 pods injected")
	assert.True(t, changed.Status.SidecarInjected, "expected SidecarInjected to be true")
	assert.Equal(t, "Injected", changed.Status.InjectionStatus, "expected InjectionStatus to be Injected")
	assert.Equal(t, int64(2), changed.Status.ObservedGeneration, "expected ObservedGeneration to match Generation")
	assert.NotEmpty(t, changed.Status.LastInjectionTime, "expected LastInjectionTime to be set")
	assert.Empty(t, changed.Status.LastInjectionError, "expected no injection error")

	assert.NotEmpty(t, changed.Status.Conditions, "expected conditions to be set")
	assert.Equal(t, "SidecarInjected", changed.Status.Conditions[0].Type, "expected SidecarInjected condition")
	assert.Equal(t, metav1.ConditionTrue, changed.Status.Conditions[0].Status, "expected condition status to be True")
	assert.Equal(t, "SidecarActive", changed.Status.Conditions[0].Reason, "expected reason to be SidecarActive")
	assert.Contains(t, changed.Status.Conditions[0].Message, "2 pod(s)", "expected message to contain pod count")

	statusBytes, err := json.MarshalIndent(changed.Status, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal status to JSON: %v", err)
	}
	fmt.Println(string(statusBytes))
}

func TestUpdateCollectorStatusWithSideCarStatusWithInjectionError(t *testing.T) {
	ctx := context.TODO()

	podWithFailedInjection := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-failed",
			Namespace: "default",
			Annotations: map[string]string{
				"sidecar.opentelemetry.io/inject": "test-sidecar",
			},
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "app:latest",
				},
				// NO otc-container here so injection failed!
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{
				{
					Type:    corev1.PodScheduled,
					Status:  corev1.ConditionFalse,
					Reason:  "Unschedulable",
					Message: "Failed to inject sidecar: webhook error",
				},
			},
		},
	}

	cli := fake.NewClientBuilder().WithObjects(podWithFailedInjection).Build()

	changed := &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-sidecar",
			Namespace:  "default",
			Generation: 3,
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode: v1beta1.ModeSidecar,
		},
	}

	err := updateCollectorStatus(ctx, cli, changed)
	assert.NoError(t, err)

	assert.Equal(t, int32(0), changed.Status.PodsInjected, "expected no pods injected when sidecar container is missing")
	assert.False(t, changed.Status.SidecarInjected, "expected SidecarInjected to be false")
	assert.Equal(t, "Failed", changed.Status.InjectionStatus, "expected InjectionStatus to be Failed")
	assert.Equal(t, int64(3), changed.Status.ObservedGeneration, "expected ObservedGeneration to match Generation")

	assert.NotEmpty(t, changed.Status.LastInjectionError, "expected LastInjectionError to be populated")
	assert.Contains(t, changed.Status.LastInjectionError, "test-pod-failed", "expected error to mention the pod name")
	assert.Contains(t, changed.Status.LastInjectionError, "Failed to inject sidecar", "expected error to contain the failure message")

	assert.NotEmpty(t, changed.Status.Conditions, "expected conditions to be set")
	assert.Equal(t, "SidecarInjected", changed.Status.Conditions[0].Type, "expected SidecarInjected condition")
	assert.Equal(t, metav1.ConditionFalse, changed.Status.Conditions[0].Status, "expected condition status to be False")
	assert.Equal(t, "NoPods", changed.Status.Conditions[0].Reason, "expected reason to be NoPods")

	statusBytes, err := json.MarshalIndent(changed.Status, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal status to JSON: %v", err)
	}
	fmt.Println(string(statusBytes))
}
