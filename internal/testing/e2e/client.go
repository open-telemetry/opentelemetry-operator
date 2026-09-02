// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"sync"
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

// fieldManager is the server-side-apply field owner used for every object this
// framework applies.
const fieldManager = "opentelemetry-operator-e2e"

// Scheme returns the scheme the framework talks to the cluster with: the built-in
// Kubernetes types plus the custom resources the e2e suites drive. Registering the
// CRDs is what lets a test express a custom resource as a typed Go struct (see
// ApplyObjects) instead of templated YAML.
var Scheme = sync.OnceValue(func() *runtime.Scheme {
	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(monitoringv1.AddToScheme(s))
	utilruntime.Must(v1beta1.AddToScheme(s))
	return s
})

// CRClient builds a controller-runtime client (typed + unstructured, with a dynamic
// RESTMapper for CRDs) from the test's REST config. Exported so e2e test packages can
// drive the typed/unstructured API directly while sharing one client construction path.
func CRClient(t *testing.T, cfg *envconf.Config) crclient.Client {
	t.Helper()
	c, err := crclient.New(cfg.Client().RESTConfig(), crclient.Options{Scheme: Scheme()})
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
