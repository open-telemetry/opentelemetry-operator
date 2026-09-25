// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestOwnerIndexKey(t *testing.T) {
	tests := []struct {
		ownerKind string
		want      string
	}{
		{"OpenTelemetryCollector", ".metadata.owner.opentelemetrycollector"},
		{"TargetAllocator", ".metadata.owner.targetallocator"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, ownerIndexKey(tt.ownerKind), "ownerKind=%s", tt.ownerKind)
	}
}

func TestOwnerIndexKey_DifferentKindsProduceDifferentKeys(t *testing.T) {
	// Two owner kinds must map to different index keys so their field indexes can
	// coexist on the same resource type without conflicting.
	assert.NotEqual(t,
		ownerIndexKey("OpenTelemetryCollector"),
		ownerIndexKey("TargetAllocator"),
	)
}

// TestFindOwnedObjects verifies that findOwnedObjects correctly partitions owned objects by owner
// kind, so that querying for one kind does not return objects whose controller owner is a different
// kind, even when the owner name is the same.
func TestFindOwnedObjects(t *testing.T) {
	trueVal := true
	const ns = "default"

	cmA := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cm-a",
			Namespace: ns,
			UID:       "uid-a",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "opentelemetry.io/v1alpha1",
				Kind:       "KindA",
				Name:       "owner",
				UID:        "uid-owner-a",
				Controller: &trueVal,
			}},
		},
	}
	cmB := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cm-b",
			Namespace: ns,
			UID:       "uid-b",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "opentelemetry.io/v1alpha1",
				Kind:       "KindB",
				Name:       "owner",
				UID:        "uid-owner-b",
				Controller: &trueVal,
			}},
		},
	}
	cmNoOwner := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "cm-no-owner", Namespace: ns, UID: "uid-no-owner"},
	}

	ownedTypes := []client.Object{&corev1.ConfigMap{}}
	cl := fake.NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(cmA, cmB, cmNoOwner).
		WithIndex(&corev1.ConfigMap{}, ownerIndexKey("KindA"), controllerOwnerIndexFunc("KindA")).
		WithIndex(&corev1.ConfigMap{}, ownerIndexKey("KindB"), controllerOwnerIndexFunc("KindB")).
		Build()

	t.Run("finds only objects owned by KindA", func(t *testing.T) {
		found, err := findOwnedObjects(t.Context(), cl, "KindA", ownedTypes, ns, "owner")
		require.NoError(t, err)
		require.Len(t, found, 1)
		for _, obj := range found {
			assert.Equal(t, cmA.Name, obj.GetName())
		}
	})

	t.Run("finds only objects owned by KindB", func(t *testing.T) {
		found, err := findOwnedObjects(t.Context(), cl, "KindB", ownedTypes, ns, "owner")
		require.NoError(t, err)
		require.Len(t, found, 1)
		for _, obj := range found {
			assert.Equal(t, cmB.Name, obj.GetName())
		}
	})

	t.Run("finds nothing for an unknown owner name", func(t *testing.T) {
		found, err := findOwnedObjects(t.Context(), cl, "KindA", ownedTypes, ns, "other-owner")
		require.NoError(t, err)
		assert.Empty(t, found)
	})

	t.Run("objects without an owner reference are not returned", func(t *testing.T) {
		found, err := findOwnedObjects(t.Context(), cl, "KindA", ownedTypes, ns, "owner")
		require.NoError(t, err)
		for _, obj := range found {
			assert.NotEqual(t, cmNoOwner.Name, obj.GetName())
		}
	})

	t.Run("returns error when the index for the owner kind is not registered", func(t *testing.T) {
		unindexedClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(cmA).Build()
		_, err := findOwnedObjects(t.Context(), unindexedClient, "KindA", ownedTypes, ns, "owner")
		assert.Error(t, err)
	})
}
