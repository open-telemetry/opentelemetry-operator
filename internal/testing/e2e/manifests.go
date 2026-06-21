// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// Apply server-side-applies multi-document YAML into ns (every object is namespaced
// into ns). Objects are decoded as unstructured, so no scheme registration is needed
// for CRDs like OpenTelemetryCollector, Prometheus or ServiceMonitor.
func Apply(ctx context.Context, t *testing.T, cfg *envconf.Config, ns, manifests string) {
	t.Helper()
	applyManifests(ctx, t, CRClient(t, cfg), strings.NewReader(manifests), ns)
}

// ApplyObjects server-side-applies typed objects into ns. It is the alternative to
// Apply for resources a test needs to vary: rather than templating YAML, build the
// object as a Go struct and set the fields that differ. Any type registered in Scheme
// works, including custom resources such as ServiceMonitor or OpenTelemetryCollector.
func ApplyObjects(ctx context.Context, t *testing.T, cfg *envconf.Config, ns string, objs ...crclient.Object) {
	t.Helper()
	c := CRClient(t, cfg)
	for _, obj := range objs {
		obj.SetNamespace(ns)
		u := toUnstructured(t, obj)
		err := c.Apply(ctx, crclient.ApplyConfigurationFromUnstructured(u), crclient.FieldOwner(fieldManager), crclient.ForceOwnership)
		require.NoError(t, err, "apply %s %q", u.GetKind(), u.GetName())
	}
}

// toUnstructured converts a typed object into an unstructured one carrying its
// apiVersion/kind (typed structs leave TypeMeta empty), so it can go through the same
// server-side-apply path as YAML manifests.
func toUnstructured(t *testing.T, obj crclient.Object) *unstructured.Unstructured {
	t.Helper()
	gvk, err := apiutil.GVKForObject(obj, Scheme())
	require.NoError(t, err, "look up GroupVersionKind for %T", obj)
	raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	require.NoError(t, err, "convert %T to unstructured", obj)
	u := &unstructured.Unstructured{Object: raw}
	u.SetGroupVersionKind(gvk)
	// Typed structs always render a status; it is a subresource on every type these
	// tests apply, so sending it would be at best ignored and at worst rejected.
	unstructured.RemoveNestedField(u.Object, "status")
	return u
}

// applyManifests SSA-applies each document from r. When forceNS is non-empty it is set
// as the namespace on every object (callers pass it only for namespaced manifests);
// when empty, each object's own namespace (if any) is respected.
func applyManifests(ctx context.Context, t *testing.T, c crclient.Client, r io.Reader, forceNS string) {
	t.Helper()
	dec := utilyaml.NewYAMLOrJSONDecoder(r, 4096)
	for {
		raw := map[string]any{}
		err := dec.Decode(&raw)
		if errors.Is(err, io.EOF) {
			return
		}
		require.NoError(t, err, "decode manifest")
		if len(raw) == 0 {
			continue
		}
		u := &unstructured.Unstructured{Object: raw}
		if forceNS != "" {
			u.SetNamespace(forceNS)
		}
		err = c.Apply(ctx, crclient.ApplyConfigurationFromUnstructured(u), crclient.FieldOwner(fieldManager), crclient.ForceOwnership)
		require.NoError(t, err, "apply %s %q", u.GetKind(), u.GetName())
	}
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
