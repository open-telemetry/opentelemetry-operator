// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apiserverendpoints

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// DefaultTTL is how long an endpoint remains live after it disappears from the API server's EndpointSlices.
const DefaultTTL = 5 * time.Minute

// CacheReader is a cache-backed reader, like the controller-runtime manager's cache.
type CacheReader interface {
	client.Reader
	// WaitForCacheSync waits until the cache is started and synced. It returns false if the context is done first.
	WaitForCacheSync(ctx context.Context) bool
}

// Tracker decides which Kubernetes API server endpoints are considered live at any given moment.
//
// An endpoint is live while it's listed in the API server's EndpointSlices, and for a TTL after it disappears from
// them. Keeping removed endpoints for a while protects consumers restricting egress to the API server from
// transient changes, like an API server briefly removing itself from the endpoints while it restarts.
// The removal time is when the Tracker first notices the endpoint is gone, so the TTL can only be extended by
// infrequent reads, never shortened. This state is kept in memory, so it's lost when the operator restarts.
//
// The Tracker reads EndpointSlices from the cache and doesn't watch them itself. Consumers should watch
// EndpointSlices, filtered with Predicate, to be notified of changes, and requeue after the expiry returned by
// Endpoints to be notified when a removed endpoint stops being live.
type Tracker struct {
	reader  CacheReader
	ttl     time.Duration
	now     func() time.Time
	service *Endpoint

	mu sync.Mutex
	// present are the endpoints listed in the EndpointSlices when the Tracker last read them.
	present map[Endpoint]struct{}
	// removed are endpoints which disappeared from the EndpointSlices, with the time the Tracker first noticed.
	removed map[Endpoint]time.Time
}

// NewTracker creates a Tracker reading EndpointSlices from the given cache. The cache should be configured with
// CacheByObject.
func NewTracker(reader CacheReader, ttl time.Duration) *Tracker {
	t := &Tracker{
		reader:  reader,
		ttl:     ttl,
		now:     time.Now,
		present: map[Endpoint]struct{}{},
		removed: map[Endpoint]time.Time{},
	}
	if service, ok := serviceFromEnv(); ok {
		t.service = &service
	}
	return t
}

// Predicate matches only the EndpointSlices backing the API server Service. Use it to filter watches on
// EndpointSlices.
func (*Tracker) Predicate() predicate.Predicate {
	return predicate.NewPredicateFuncs(isAPIServerEndpointSlice)
}

// Endpoints returns the live API server endpoints, sorted by port and IP. It also returns the time until the
// earliest removed endpoint stops being live, or zero if there are none, so the caller can requeue to update
// anything based on the endpoints. It reads from the cache and returns an error if the cache isn't started, or if
// there are no live endpoints.
//
// The endpoints include the API server Service's ClusterIP, taken from the KUBERNETES_SERVICE_HOST and
// KUBERNETES_SERVICE_PORT environment variables, as CNIs differ in whether they evaluate NetworkPolicies
// before or after translating the ClusterIP. This endpoint is always live.
func (t *Tracker) Endpoints(ctx context.Context) ([]Endpoint, time.Duration, error) {
	// Hold the lock while reading, so that the state is updated in the same order as the cache.
	t.mu.Lock()
	defer t.mu.Unlock()
	current, err := endpointSliceEndpoints(ctx, t.reader)
	if err != nil {
		return nil, 0, err
	}
	now := t.now()

	present := make(map[Endpoint]struct{}, len(current))
	for _, endpoint := range current {
		present[endpoint] = struct{}{}
		delete(t.removed, endpoint)
	}
	for endpoint := range t.present {
		if _, ok := present[endpoint]; !ok {
			t.removed[endpoint] = now
		}
	}
	t.present = present

	live := current
	var expiresIn time.Duration
	for endpoint, removedAt := range t.removed {
		remaining := removedAt.Add(t.ttl).Sub(now)
		if remaining <= 0 {
			delete(t.removed, endpoint)
			continue
		}
		live = append(live, endpoint)
		if expiresIn == 0 || remaining < expiresIn {
			expiresIn = remaining
		}
	}
	if t.service != nil {
		live = append(live, *t.service)
	}

	if len(live) == 0 {
		return nil, 0, fmt.Errorf("no endpoints found for the Kubernetes API server: no usable %s/%s EndpointSlices, and KUBERNETES_SERVICE_HOST is not set", Namespace, ServiceName)
	}
	return sortEndpoints(live), expiresIn, nil
}

// WaitForEndpoints is like Endpoints, but first waits for the cache to start and sync. Use it outside of
// reconcilers, which only run after the cache is synced.
func (t *Tracker) WaitForEndpoints(ctx context.Context) ([]Endpoint, time.Duration, error) {
	if !t.reader.WaitForCacheSync(ctx) {
		return nil, 0, errors.Join(errors.New("failed waiting for the cache to sync"), ctx.Err())
	}
	return t.Endpoints(ctx)
}
