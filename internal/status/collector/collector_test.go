// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
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

	err := UpdateCollectorStatus(ctx, cli, changed)
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

	err := UpdateCollectorStatus(ctx, cli, changed)
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

	err := UpdateCollectorStatus(ctx, cli, changed)
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

	err := UpdateCollectorStatus(ctx, cli, changed)
	assert.NoError(t, err)

	assert.Contains(t, changed.Status.Scale.Selector, "customLabel=customValue", "expected selector to contain customlabel=customValue")
	assert.Equal(t, "app:latest", changed.Status.Image, "expected image to be app:latest")
}
