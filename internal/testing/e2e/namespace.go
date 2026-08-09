// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// nsContextKey is the context key under which SetupNamespace stores the namespace.
type nsContextKey struct{}

// SetupNamespace creates a namespace named after the running test (see
// NamespaceFromT), registers its deletion as test cleanup, and returns a context
// carrying the namespace name — retrieve it with Namespace. Every test gets its own
// namespace, which is what lets the manifests use fixed object names.
func SetupNamespace(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	t.Helper()
	ns := NamespaceFromT(t)
	CreateNamespace(ctx, t, cfg, ns)
	t.Cleanup(func() {
		// The feature's context is already cancelled by the time cleanup runs.
		DeleteNamespace(context.WithoutCancel(ctx), t, cfg, ns)
	})
	return context.WithValue(ctx, nsContextKey{}, ns)
}

// Namespace returns the namespace SetupNamespace stored in ctx, failing t when the
// context does not carry one (i.e. the feature's Setup did not call SetupNamespace).
func Namespace(t *testing.T, ctx context.Context) string {
	t.Helper()
	ns, ok := ctx.Value(nsContextKey{}).(string)
	require.True(t, ok, "no namespace in context: call e2e.SetupNamespace in the feature's Setup")
	return ns
}

// CreateNamespace creates ns.
func CreateNamespace(ctx context.Context, t *testing.T, cfg *envconf.Config, ns string) {
	t.Helper()
	obj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
	require.NoError(t, CRClient(t, cfg).Create(ctx, obj), "create namespace %s", ns)
}

// DeleteNamespace deletes ns (ignoring not-found), used for test cleanup.
func DeleteNamespace(ctx context.Context, t *testing.T, cfg *envconf.Config, ns string) {
	t.Helper()
	obj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
	err := CRClient(t, cfg).Delete(ctx, obj)
	if !apierrors.IsNotFound(err) {
		require.NoError(t, err, "delete namespace %s", ns)
	}
}
