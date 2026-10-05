// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apiserverendpoints

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	discoveryv1 "k8s.io/api/discovery/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// syncedReader is a CacheReader which is always synced.
type syncedReader struct {
	client.Reader
}

func (syncedReader) WaitForCacheSync(context.Context) bool {
	return true
}

func newTestTracker(t *testing.T, service *Endpoint, objects ...client.Object) (*Tracker, client.Client, *time.Time) {
	t.Helper()
	c := fake.NewClientBuilder().WithObjects(objects...).Build()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tracker := NewTracker(syncedReader{c}, time.Minute)
	tracker.now = func() time.Time { return now }
	tracker.service = service
	return tracker, c, &now
}

func setAddresses(t *testing.T, c client.Client, slice *discoveryv1.EndpointSlice, addresses ...string) {
	t.Helper()
	slice.Endpoints = nil
	for _, address := range addresses {
		slice.Endpoints = append(slice.Endpoints, discoveryv1.Endpoint{Addresses: []string{address}})
	}
	require.NoError(t, c.Update(t.Context(), slice))
}

func TestTrackerEndpoints(t *testing.T) {
	slice := endpointSlice("kubernetes", Namespace, ServiceName, discoveryv1.AddressTypeIPv4, 6443, "172.18.0.3", "172.18.0.2")
	service := &Endpoint{IP: "10.96.0.1", Port: 443}
	tracker, _, _ := newTestTracker(t, service, slice)

	endpoints, expiresIn, err := tracker.Endpoints(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []Endpoint{
		{IP: "10.96.0.1", Port: 443},
		{IP: "172.18.0.2", Port: 6443},
		{IP: "172.18.0.3", Port: 6443},
	}, endpoints)
	assert.Zero(t, expiresIn)
}

func TestTrackerRemovedEndpointsExpire(t *testing.T) {
	slice := endpointSlice("kubernetes", Namespace, ServiceName, discoveryv1.AddressTypeIPv4, 6443, "172.18.0.2")
	tracker, c, now := newTestTracker(t, nil, slice)

	endpoints, _, err := tracker.Endpoints(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []Endpoint{{IP: "172.18.0.2", Port: 6443}}, endpoints)

	// the API server is replaced by one with a different IP
	setAddresses(t, c, slice, "172.18.0.3")
	*now = now.Add(10 * time.Second)
	endpoints, expiresIn, err := tracker.Endpoints(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []Endpoint{{IP: "172.18.0.2", Port: 6443}, {IP: "172.18.0.3", Port: 6443}}, endpoints)
	assert.Equal(t, time.Minute, expiresIn)

	// the TTL is counted from when the removal was first noticed
	*now = now.Add(40 * time.Second)
	endpoints, expiresIn, err = tracker.Endpoints(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []Endpoint{{IP: "172.18.0.2", Port: 6443}, {IP: "172.18.0.3", Port: 6443}}, endpoints)
	assert.Equal(t, 20*time.Second, expiresIn)

	*now = now.Add(20 * time.Second)
	endpoints, expiresIn, err = tracker.Endpoints(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []Endpoint{{IP: "172.18.0.3", Port: 6443}}, endpoints)
	assert.Zero(t, expiresIn)
}

func TestTrackerReturningEndpointIsNoLongerRemoved(t *testing.T) {
	slice := endpointSlice("kubernetes", Namespace, ServiceName, discoveryv1.AddressTypeIPv4, 6443, "172.18.0.2")
	tracker, c, now := newTestTracker(t, nil, slice)

	_, _, err := tracker.Endpoints(t.Context())
	require.NoError(t, err)

	// the API server briefly removes itself from the endpoints while restarting
	setAddresses(t, c, slice)
	*now = now.Add(30 * time.Second)
	endpoints, expiresIn, err := tracker.Endpoints(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []Endpoint{{IP: "172.18.0.2", Port: 6443}}, endpoints)
	assert.Equal(t, time.Minute, expiresIn)

	setAddresses(t, c, slice, "172.18.0.2")
	*now = now.Add(30 * time.Second)
	_, expiresIn, err = tracker.Endpoints(t.Context())
	require.NoError(t, err)
	assert.Zero(t, expiresIn)

	// a later removal gets a full TTL again
	setAddresses(t, c, slice)
	*now = now.Add(30 * time.Second)
	endpoints, expiresIn, err = tracker.Endpoints(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []Endpoint{{IP: "172.18.0.2", Port: 6443}}, endpoints)
	assert.Equal(t, time.Minute, expiresIn)
}

func TestTrackerNoEndpoints(t *testing.T) {
	tracker, _, _ := newTestTracker(t, nil)
	_, _, err := tracker.Endpoints(t.Context())
	require.ErrorContains(t, err, "no endpoints found for the Kubernetes API server")

	// the Service ClusterIP alone is enough
	tracker.service = &Endpoint{IP: "10.96.0.1", Port: 443}
	endpoints, _, err := tracker.Endpoints(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []Endpoint{{IP: "10.96.0.1", Port: 443}}, endpoints)
}
