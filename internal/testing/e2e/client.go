// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// fieldManager is the server-side-apply field owner used for every object this
// framework applies.
const fieldManager = "opentelemetry-operator-e2e"

// CRClient builds a controller-runtime client (typed + unstructured, with a dynamic
// RESTMapper for CRDs) from the test's REST config. Exported so e2e test packages can
// drive the typed/unstructured API directly while sharing one client construction path.
func CRClient(t *testing.T, cfg *envconf.Config) crclient.Client {
	t.Helper()
	c, err := crclient.New(cfg.Client().RESTConfig(), crclient.Options{Scheme: clientgoscheme.Scheme})
	require.NoError(t, err, "create controller-runtime client")
	return c
}

// ClientSet builds a client-go clientset, used for the API server service proxy and by
// e2e test packages that drive the typed Kubernetes API directly.
func ClientSet(t *testing.T, cfg *envconf.Config) *kubernetes.Clientset {
	t.Helper()
	cs, err := clientSet(cfg)
	require.NoError(t, err, "create clientset")
	return cs
}

// clientSet is the error-returning variant of ClientSet, for helpers that run inside
// polling loops or cleanup handlers where failing the test immediately is wrong.
func clientSet(cfg *envconf.Config) (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(cfg.Client().RESTConfig())
}
