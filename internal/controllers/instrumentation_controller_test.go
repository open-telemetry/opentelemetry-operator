// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestHashSpec(t *testing.T) {
	spec1 := v1alpha1.InstrumentationSpec{
		Java: v1alpha1.Java{Image: "java:1"},
	}
	spec2 := v1alpha1.InstrumentationSpec{
		Java: v1alpha1.Java{Image: "java:2"},
	}
	specWithAutoUpdate := v1alpha1.InstrumentationSpec{
		AutoUpdate: ptr.To(true),
		Java:       v1alpha1.Java{Image: "java:1"},
	}

	h1, err := hashSpec(spec1)
	require.NoError(t, err)

	h2, err := hashSpec(spec2)
	require.NoError(t, err)

	hAutoUpdate, err := hashSpec(specWithAutoUpdate)
	require.NoError(t, err)

	assert.NotEqual(t, h1, h2, "different specs should produce different hashes")
	assert.Equal(t, h1, hAutoUpdate, "toggling AutoUpdate should not change the hash")
}

func TestReferencesInstrumentation(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		instName    string
		isOnlyInst  bool
		expected    bool
	}{
		{
			name:        "direct name match",
			annotations: map[string]string{"instrumentation.opentelemetry.io/inject-java": "my-inst"},
			instName:    "my-inst",
			isOnlyInst:  false,
			expected:    true,
		},
		{
			name:        "true annotation with single inst",
			annotations: map[string]string{"instrumentation.opentelemetry.io/inject-python": "true"},
			instName:    "my-inst",
			isOnlyInst:  true,
			expected:    true,
		},
		{
			name:        "true annotation with multiple insts",
			annotations: map[string]string{"instrumentation.opentelemetry.io/inject-python": "true"},
			instName:    "my-inst",
			isOnlyInst:  false,
			expected:    false,
		},
		{
			name:        "no matching annotation",
			annotations: map[string]string{"instrumentation.opentelemetry.io/inject-java": "other-inst"},
			instName:    "my-inst",
			isOnlyInst:  false,
			expected:    false,
		},
		{
			name:        "no annotations",
			annotations: nil,
			instName:    "my-inst",
			isOnlyInst:  true,
			expected:    false,
		},
		{
			name:        "false annotation",
			annotations: map[string]string{"instrumentation.opentelemetry.io/inject-java": "false"},
			instName:    "my-inst",
			isOnlyInst:  true,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podMeta := metav1.ObjectMeta{Annotations: tt.annotations}
			result := referencesInstrumentation(podMeta, tt.instName, tt.isOnlyInst)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInstrumentationReconciler_AutoUpdateDisabled(t *testing.T) {
	inst := &v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-inst",
			Namespace: "default",
		},
		Spec: v1alpha1.InstrumentationSpec{
			Java: v1alpha1.Java{Image: "java:1"},
		},
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"app": "test"},
					Annotations: map[string]string{"instrumentation.opentelemetry.io/inject-java": "my-inst"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app", Image: "app:1"}},
				},
			},
		},
	}

	k8s := fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(inst, deploy).
		WithStatusSubresource(inst).
		Build()

	r := NewInstrumentationReconciler(InstrumentationReconcilerParams{
		Client:   k8s,
		Scheme:   testScheme,
		Log:      ctrl.Log.WithName("test"),
		Recorder: record.NewFakeRecorder(10),
	})

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-inst", Namespace: "default"},
	})
	require.NoError(t, err)

	var updated appsv1.Deployment
	require.NoError(t, k8s.Get(context.Background(), types.NamespacedName{Name: "my-app", Namespace: "default"}, &updated))
	assert.Empty(t, updated.Spec.Template.Annotations[instrumentationSpecHashAnnotation])
}

func TestInstrumentationReconciler_AutoUpdateEnabled(t *testing.T) {
	inst := &v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-inst",
			Namespace: "default",
		},
		Spec: v1alpha1.InstrumentationSpec{
			AutoUpdate: ptr.To(true),
			Java:       v1alpha1.Java{Image: "java:1"},
		},
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"app": "test"},
					Annotations: map[string]string{"instrumentation.opentelemetry.io/inject-java": "my-inst"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app", Image: "app:1"}},
				},
			},
		},
	}

	k8s := fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(inst, deploy).
		WithStatusSubresource(inst).
		Build()

	r := NewInstrumentationReconciler(InstrumentationReconcilerParams{
		Client:   k8s,
		Scheme:   testScheme,
		Log:      ctrl.Log.WithName("test"),
		Recorder: record.NewFakeRecorder(10),
	})

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-inst", Namespace: "default"},
	})
	require.NoError(t, err)

	var updated appsv1.Deployment
	require.NoError(t, k8s.Get(context.Background(), types.NamespacedName{Name: "my-app", Namespace: "default"}, &updated))
	assert.NotEmpty(t, updated.Spec.Template.Annotations[instrumentationSpecHashAnnotation])
	assert.NotEmpty(t, updated.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"])
}

func TestInstrumentationReconciler_NoRestartWhenHashUnchanged(t *testing.T) {
	inst := &v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-inst",
			Namespace: "default",
		},
		Spec: v1alpha1.InstrumentationSpec{
			AutoUpdate: ptr.To(true),
			Java:       v1alpha1.Java{Image: "java:1"},
		},
	}

	specHash, err := hashSpec(inst.Spec)
	require.NoError(t, err)

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "test"},
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/inject-java": "my-inst",
						instrumentationSpecHashAnnotation:              specHash,
						"kubectl.kubernetes.io/restartedAt":            "2024-01-01T00:00:00Z",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app", Image: "app:1"}},
				},
			},
		},
	}

	k8s := fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(inst, deploy).
		WithStatusSubresource(inst).
		Build()

	r := NewInstrumentationReconciler(InstrumentationReconcilerParams{
		Client:   k8s,
		Scheme:   testScheme,
		Log:      ctrl.Log.WithName("test"),
		Recorder: record.NewFakeRecorder(10),
	})

	_, err = r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-inst", Namespace: "default"},
	})
	require.NoError(t, err)

	var updated appsv1.Deployment
	require.NoError(t, k8s.Get(context.Background(), types.NamespacedName{Name: "my-app", Namespace: "default"}, &updated))
	assert.Equal(t, "2024-01-01T00:00:00Z", updated.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"])
}

func TestInstrumentationReconciler_TrueAnnotationSingleInst(t *testing.T) {
	inst := &v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-inst",
			Namespace: "default",
		},
		Spec: v1alpha1.InstrumentationSpec{
			AutoUpdate: ptr.To(true),
			Java:       v1alpha1.Java{Image: "java:1"},
		},
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"app": "test"},
					Annotations: map[string]string{"instrumentation.opentelemetry.io/inject-java": "true"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app", Image: "app:1"}},
				},
			},
		},
	}

	k8s := fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(inst, deploy).
		WithStatusSubresource(inst).
		Build()

	r := NewInstrumentationReconciler(InstrumentationReconcilerParams{
		Client:   k8s,
		Scheme:   testScheme,
		Log:      ctrl.Log.WithName("test"),
		Recorder: record.NewFakeRecorder(10),
	})

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-inst", Namespace: "default"},
	})
	require.NoError(t, err)

	var updated appsv1.Deployment
	require.NoError(t, k8s.Get(context.Background(), types.NamespacedName{Name: "my-app", Namespace: "default"}, &updated))
	assert.NotEmpty(t, updated.Spec.Template.Annotations[instrumentationSpecHashAnnotation])
}

func TestInstrumentationReconciler_StatefulSetAndDaemonSet(t *testing.T) {
	inst := &v1alpha1.Instrumentation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-inst",
			Namespace: "default",
		},
		Spec: v1alpha1.InstrumentationSpec{
			AutoUpdate: ptr.To(true),
			Python:     v1alpha1.Python{Image: "python:1"},
		},
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-sts",
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "sts"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"app": "sts"},
					Annotations: map[string]string{"instrumentation.opentelemetry.io/inject-python": "my-inst"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app", Image: "app:1"}},
				},
			},
		},
	}

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-ds",
			Namespace: "default",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "ds"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"app": "ds"},
					Annotations: map[string]string{"instrumentation.opentelemetry.io/inject-python": "my-inst"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app", Image: "app:1"}},
				},
			},
		},
	}

	k8s := fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(inst, sts, ds).
		WithStatusSubresource(inst).
		Build()

	r := NewInstrumentationReconciler(InstrumentationReconcilerParams{
		Client:   k8s,
		Scheme:   testScheme,
		Log:      ctrl.Log.WithName("test"),
		Recorder: record.NewFakeRecorder(10),
	})

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-inst", Namespace: "default"},
	})
	require.NoError(t, err)

	var updatedSts appsv1.StatefulSet
	require.NoError(t, k8s.Get(context.Background(), types.NamespacedName{Name: "my-sts", Namespace: "default"}, &updatedSts))
	assert.NotEmpty(t, updatedSts.Spec.Template.Annotations[instrumentationSpecHashAnnotation])

	var updatedDs appsv1.DaemonSet
	require.NoError(t, k8s.Get(context.Background(), types.NamespacedName{Name: "my-ds", Namespace: "default"}, &updatedDs))
	assert.NotEmpty(t, updatedDs.Spec.Template.Annotations[instrumentationSpecHashAnnotation])
}
