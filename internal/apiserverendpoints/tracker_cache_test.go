// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apiserverendpoints

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlenvtest "sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/open-telemetry/opentelemetry-operator/internal/testenv"
)

// TestTrackerWithCache verifies that a cache configured with CacheByObject serves the API server EndpointSlices even
// when the default namespace isn't watched, doesn't serve any other EndpointSlices, and that the Tracker can wait
// for it to start.
func TestTrackerWithCache(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")

	env, err := testenv.Start(&ctrlenvtest.Environment{}, scheme.Scheme)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, env.Stop()) })

	watchedNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "watched"}}
	require.NoError(t, env.Client.Create(t.Context(), watchedNamespace))
	for _, slice := range []client.Object{
		endpointSlice("kubernetes", Namespace, ServiceName, discoveryv1.AddressTypeIPv4, 6443, "172.18.0.2"),
		endpointSlice("other", Namespace, "other", discoveryv1.AddressTypeIPv4, 6443, "172.18.0.9"),
		endpointSlice("kubernetes", watchedNamespace.Name, ServiceName, discoveryv1.AddressTypeIPv4, 6443, "172.18.0.9"),
	} {
		require.NoError(t, env.Client.Create(t.Context(), slice))
	}

	endpointSliceType, endpointSliceCache := CacheByObject()
	c, err := cache.New(env.Config, cache.Options{
		Scheme:            scheme.Scheme,
		DefaultNamespaces: map[string]cache.Config{watchedNamespace.Name: {}},
		ByObject:          map[client.Object]cache.ByObject{endpointSliceType: endpointSliceCache},
	})
	require.NoError(t, err)
	tracker := NewTracker(c, DefaultTTL)

	// reading before the cache is started fails
	_, _, err = tracker.Endpoints(t.Context())
	var notStarted *cache.ErrCacheNotStarted
	require.ErrorAs(t, err, &notStarted)

	// waiting for the endpoints blocks until the cache is started
	type result struct {
		endpoints []Endpoint
		err       error
	}
	waitResult := make(chan result, 1)
	go func() {
		endpoints, _, waitErr := tracker.WaitForEndpoints(t.Context())
		waitResult <- result{endpoints, waitErr}
	}()
	go func() {
		assert.NoError(t, c.Start(t.Context()))
	}()

	select {
	case r := <-waitResult:
		require.NoError(t, r.err)
		assert.Equal(t, []Endpoint{{IP: "172.18.0.2", Port: 6443}}, r.endpoints)
	case <-time.After(time.Minute):
		t.Fatal("timed out waiting for the API server endpoints")
	}

	// only the API server EndpointSlice is cached
	var slices discoveryv1.EndpointSliceList
	require.NoError(t, c.List(t.Context(), &slices))
	require.Len(t, slices.Items, 1)
	assert.Equal(t, client.ObjectKey{Namespace: Namespace, Name: "kubernetes"}, client.ObjectKeyFromObject(&slices.Items[0]))
}
