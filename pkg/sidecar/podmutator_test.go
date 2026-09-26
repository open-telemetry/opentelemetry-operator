// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sidecar

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// flakyGetInterceptor returns a NotFound error for the given number of Get calls
// against a matching object name before delegating to the real client, simulating
// a cache-backed client that briefly lags behind a just-created object.
func flakyGetInterceptor(name string, failures int) interceptor.Funcs {
	calls := 0
	return interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if key.Name == name && calls < failures {
				calls++
				return apierrors.NewNotFound(schema.GroupResource{}, key.Name)
			}
			return c.Get(ctx, key, obj, opts...)
		},
	}
}

func TestGetReplicaSetReferenceRetriesOnNotFound(t *testing.T) {
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "my-ns"}}
	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-replicaset",
			Namespace: "my-ns",
			UID:       "uuid-replicaset",
		},
	}
	fakeClient := fake.NewClientBuilder().
		WithObjects(replicaSet).
		WithInterceptorFuncs(flakyGetInterceptor("my-replicaset", 3)).
		Build()

	mutator := &sidecarPodMutator{client: fakeClient, logger: logr.Discard()}

	ownerRefs := []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "my-replicaset"}}
	got := mutator.getReplicaSetReference(context.Background(), ownerRefs, ns)

	assert.NotNil(t, got, "expected the replicaset to be found once the transient NotFound errors are retried")
	assert.Equal(t, "my-replicaset", got.Name)
}

func TestGetReplicaSetReferenceReturnsNilOnPersistentNotFound(t *testing.T) {
	// Use a short-lived backoff so this "always fails" case doesn't pay the full
	// production retry budget (defined for a webhook call, not a unit test).
	original := ownerLookupBackoff
	ownerLookupBackoff = wait.Backoff{Duration: time.Millisecond, Factor: 1, Steps: 3}
	t.Cleanup(func() { ownerLookupBackoff = original })

	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "my-ns"}}
	fakeClient := fake.NewClientBuilder().Build()
	mutator := &sidecarPodMutator{client: fakeClient, logger: logr.Discard()}

	ownerRefs := []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "missing-replicaset"}}
	got := mutator.getReplicaSetReference(context.Background(), ownerRefs, ns)

	assert.Nil(t, got)
}

func TestGetDeploymentReferenceRetriesOnNotFound(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-deployment",
			Namespace: "my-ns",
			UID:       "uuid-deployment",
		},
	}
	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "my-replicaset",
			Namespace:       "my-ns",
			OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: "my-deployment"}},
		},
	}
	fakeClient := fake.NewClientBuilder().
		WithObjects(deployment).
		WithInterceptorFuncs(flakyGetInterceptor("my-deployment", 3)).
		Build()

	mutator := &sidecarPodMutator{client: fakeClient, logger: logr.Discard()}

	got := mutator.getDeploymentReference(context.Background(), replicaSet)

	assert.NotNil(t, got, "expected the deployment to be found once the transient NotFound errors are retried")
	assert.Equal(t, "my-deployment", got.Name)
}
