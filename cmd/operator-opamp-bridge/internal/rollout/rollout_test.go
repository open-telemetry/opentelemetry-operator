// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package rollout_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/rollout"
)

const (
	testNamespace = "test-ns"
	testName      = "test-workload"
)

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(s))
	return s
}

func TestTriggerRollout_Deployment(t *testing.T) {
	deploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace}}
	k8s := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(deploy).Build()

	before := time.Now().Truncate(time.Second)
	require.NoError(t, rollout.TriggerRollout(context.Background(), k8s, testNamespace, "deployment", testName))

	result := &appsv1.Deployment{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: testName, Namespace: testNamespace}, result))
	val := result.Spec.Template.Annotations[rollout.RestartAnnotation]
	assert.NotEmpty(t, val)
	parsed, err := time.Parse(time.RFC3339, val)
	require.NoError(t, err, "annotation value must be RFC3339")
	assert.False(t, parsed.Before(before), "restart timestamp must not be before test start")
}

func TestTriggerRollout_DaemonSet(t *testing.T) {
	ds := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace}}
	k8s := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(ds).Build()

	require.NoError(t, rollout.TriggerRollout(context.Background(), k8s, testNamespace, "daemonset", testName))

	result := &appsv1.DaemonSet{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: testName, Namespace: testNamespace}, result))
	assert.NotEmpty(t, result.Spec.Template.Annotations[rollout.RestartAnnotation])
}

func TestTriggerRollout_StatefulSet(t *testing.T) {
	sts := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace}}
	k8s := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(sts).Build()

	require.NoError(t, rollout.TriggerRollout(context.Background(), k8s, testNamespace, "statefulset", testName))

	result := &appsv1.StatefulSet{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: testName, Namespace: testNamespace}, result))
	assert.NotEmpty(t, result.Spec.Template.Annotations[rollout.RestartAnnotation])
}

func TestTriggerRollout_CaseInsensitive(t *testing.T) {
	deploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace}}
	k8s := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(deploy).Build()

	require.NoError(t, rollout.TriggerRollout(context.Background(), k8s, testNamespace, "Deployment", testName))

	result := &appsv1.Deployment{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: testName, Namespace: testNamespace}, result))
	assert.NotEmpty(t, result.Spec.Template.Annotations[rollout.RestartAnnotation])
}

func TestTriggerRollout_PreservesExistingAnnotations(t *testing.T) {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"existing-key": "existing-value"},
				},
			},
		},
	}
	k8s := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(deploy).Build()

	require.NoError(t, rollout.TriggerRollout(context.Background(), k8s, testNamespace, "deployment", testName))

	result := &appsv1.Deployment{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: testName, Namespace: testNamespace}, result))
	assert.Equal(t, "existing-value", result.Spec.Template.Annotations["existing-key"], "pre-existing annotations must be preserved")
	assert.NotEmpty(t, result.Spec.Template.Annotations[rollout.RestartAnnotation])
}

func TestTriggerRollout_UnknownType(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	err := rollout.TriggerRollout(context.Background(), k8s, testNamespace, "sidecar", testName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported workload type")
}

func TestTriggerRollout_NotFound(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	err := rollout.TriggerRollout(context.Background(), k8s, testNamespace, "deployment", testName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Deployment")
}
