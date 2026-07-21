// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package e2e

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// Apply server-side-applies multi-document YAML into ns (every object is namespaced
// into ns). Objects are decoded as unstructured, so no scheme registration is needed
// for CRDs like OpenTelemetryCollector, Prometheus or ServiceMonitor.
func Apply(ctx context.Context, t *testing.T, cfg *envconf.Config, ns, manifests string) {
	t.Helper()
	applyManifests(ctx, t, CRClient(t, cfg), strings.NewReader(manifests), ns)
}

// applyManifests SSA-applies each document from r. When forceNS is non-empty it is set
// as the namespace on every object (callers pass it only for namespaced manifests);
// when empty, each object's own namespace (if any) is respected.
func applyManifests(ctx context.Context, t *testing.T, c crclient.Client, r io.Reader, forceNS string) {
	t.Helper()
	dec := utilyaml.NewYAMLOrJSONDecoder(r, 4096)
	for {
		raw := map[string]any{}
		if err := dec.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			t.Fatalf("decode manifest: %v", err)
		}
		if len(raw) == 0 {
			continue
		}
		u := &unstructured.Unstructured{Object: raw}
		if forceNS != "" {
			u.SetNamespace(forceNS)
		}
		if err := c.Apply(ctx, crclient.ApplyConfigurationFromUnstructured(u), crclient.FieldOwner(fieldManager), crclient.ForceOwnership); err != nil {
			t.Fatalf("apply %s %q: %v", u.GetKind(), u.GetName(), err)
		}
	}
}

// CreateNamespace creates ns.
func CreateNamespace(ctx context.Context, t *testing.T, cfg *envconf.Config, ns string) {
	t.Helper()
	obj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
	if err := CRClient(t, cfg).Create(ctx, obj); err != nil {
		t.Fatalf("create namespace %s: %v", ns, err)
	}
}

// DeleteNamespace deletes ns (ignoring not-found), used for test cleanup.
func DeleteNamespace(ctx context.Context, t *testing.T, cfg *envconf.Config, ns string) {
	t.Helper()
	obj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
	if err := CRClient(t, cfg).Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		t.Fatalf("delete namespace %s: %v", ns, err)
	}
}
