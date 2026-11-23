// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func clusterObservabilityOwner(name string, uid types.UID) []metav1.OwnerReference {
	return []metav1.OwnerReference{{
		APIVersion:         v1alpha1.GroupVersion.String(),
		Kind:               "ClusterObservability",
		Name:               name,
		UID:                uid,
		Controller:         new(true),
		BlockOwnerDeletion: new(true),
	}}
}

func TestReconcileOpenTelemetryResourceSyncsOwnerReferences(t *testing.T) {
	const (
		name      = "cluster-obs"
		namespace = "observability"
	)
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	require.NoError(t, v1beta1.AddToScheme(scheme))

	tests := []struct {
		name    string
		desired func(owners []metav1.OwnerReference) client.Object
		fetch   func() client.Object
	}{
		{
			name: "collector",
			desired: func(owners []metav1.OwnerReference) client.Object {
				return &v1beta1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{Name: name + "-agent", Namespace: namespace, OwnerReferences: owners},
				}
			},
			fetch: func() client.Object { return &v1beta1.OpenTelemetryCollector{} },
		},
		{
			name: "instrumentation",
			desired: func(owners []metav1.OwnerReference) client.Object {
				return &v1alpha1.Instrumentation{
					ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, OwnerReferences: owners},
				}
			},
			fetch: func() client.Object { return &v1alpha1.Instrumentation{} },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldOwners := clusterObservabilityOwner(name, "old-uid")
			newOwners := clusterObservabilityOwner(name, "new-uid")

			// The specs are identical, only the owner differs, as when a
			// ClusterObservability is recreated under the same name.
			existing := tt.desired(oldOwners)
			cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
			r := &ClusterObservabilityReconciler{Client: cli}

			desired := tt.desired(newOwners)
			require.NoError(t, r.reconcileOpenTelemetryResource(context.Background(), logr.Discard(), desired))

			got := tt.fetch()
			require.NoError(t, cli.Get(context.Background(), client.ObjectKeyFromObject(desired), got))
			assert.Equal(t, newOwners, got.GetOwnerReferences())
		})
	}
}
