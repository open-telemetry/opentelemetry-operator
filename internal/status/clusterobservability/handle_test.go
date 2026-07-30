// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clusterobservability

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
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
			ObjectMeta: metav1.ObjectMeta{Name: name + "-agent", Namespace: namespace},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-agent-collector", Namespace: namespace},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 2,
				NumberReady:            2,
			},
		},
		&v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-cluster", Namespace: namespace},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-cluster-collector", Namespace: namespace},
			Status: appsv1.StatefulSetStatus{
				Replicas:      1,
				ReadyReplicas: 1,
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

	updateClusterObservabilityStatus(context.Background(), logr.Discard(), cli, co, false)

	assert.Equal(t, "Ready", co.Status.Phase)
	assert.Equal(t, co.Generation, co.Status.ObservedGeneration)
	for _, component := range []string{componentAgentCollector, componentClusterCollector, componentInstrumentation} {
		assert.Truef(t, co.Status.ComponentsStatus[component].Ready, "component %q should be ready", component)
	}

	ready := findCondition(co.Status.Conditions, v1alpha1.ClusterObservabilityConditionReady)
	require.NotNil(t, ready)
	assert.Equal(t, metav1.ConditionTrue, ready.Status)
	assert.Equal(t, reasonReady, ready.Reason)
}

func TestCheckClusterCollectorStatusRequiresReadyStatefulSet(t *testing.T) {
	const (
		name      = "cluster-obs"
		namespace = "observability"
	)
	co := &v1alpha1.ClusterObservability{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	cli := newStatusTestClient(t,
		&v1beta1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-cluster", Namespace: namespace},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: name + "-cluster-collector", Namespace: namespace},
			Status: appsv1.StatefulSetStatus{
				Replicas:      1,
				ReadyReplicas: 0,
			},
		},
	)

	status := checkClusterCollectorStatus(context.Background(), cli, co)
	assert.False(t, status.ready)
	assert.Equal(t, "Cluster collector StatefulSet not ready: 0/1 replicas ready", status.message)
}

func newStatusTestClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	require.NoError(t, v1beta1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}
