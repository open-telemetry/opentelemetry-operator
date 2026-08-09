// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// EnsurePrometheusOperator installs prometheus-operator into the cluster if its
// controller is not already running, so tests can use a Prometheus CR as a live
// oracle. The version matches the prometheus-operator module this repo depends on
// (see prometheusOperatorVersion). Idempotent: a no-op once installed. The release
// bundle is fetched over the network and server-side-applied.
//
// Availability is waited for on every path, not just after a fresh install: an
// install left mid-rollout (or crash-looping) by a previous or concurrent run must
// not let tests proceed against an operator that is not reconciling yet.
func EnsurePrometheusOperator(ctx context.Context, t *testing.T, cfg *envconf.Config) {
	t.Helper()
	c := CRClient(t, cfg)
	dep := &appsv1.Deployment{}
	err := c.Get(ctx, crclient.ObjectKey{Namespace: "default", Name: "prometheus-operator"}, dep)
	if apierrors.IsNotFound(err) {
		installPrometheusOperator(ctx, t, c)
	} else {
		require.NoError(t, err, "get prometheus-operator deployment")
	}
	// Returns almost immediately when the operator is already available.
	WaitForDeployment(ctx, t, cfg, "default", "prometheus-operator", 3*time.Minute)
}

// installPrometheusOperator fetches the release bundle matching the module version
// and server-side-applies it.
func installPrometheusOperator(ctx context.Context, t *testing.T, c crclient.Client) {
	t.Helper()
	url := "https://github.com/prometheus-operator/prometheus-operator/releases/download/" + prometheusOperatorVersion(t) + "/bundle.yaml"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	require.NoError(t, err, "request bundle")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "fetch bundle")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fetch bundle: status %d", resp.StatusCode)
	}
	// The bundle mixes cluster-scoped (CRDs, RBAC) and namespaced (operator in
	// default) objects, so its own namespaces are respected.
	applyManifests(ctx, t, c, resp.Body, "")
}

// prometheusOperatorVersion returns the prometheus-operator version this repo depends
// on, read from go.mod, so the installed operator matches the API types and CRDs the
// code is written against.
func prometheusOperatorVersion(t *testing.T) string {
	t.Helper()
	const module = "github.com/prometheus-operator/prometheus-operator"
	path := filepath.Join(RepoRoot(t), "go.mod")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "read go.mod")
	f, err := modfile.Parse(path, data, nil)
	require.NoError(t, err, "parse go.mod")
	for _, r := range f.Require {
		// Match the module's own require, not its /pkg/... submodules.
		if r.Mod.Path == module {
			return r.Mod.Version
		}
	}
	t.Fatalf("module %s not found in go.mod", module)
	return ""
}
