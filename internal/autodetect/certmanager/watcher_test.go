// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package certmanager

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type availabilityCheckFunc func(context.Context) (Availability, error)

func (f availabilityCheckFunc) CertManagerAvailability(ctx context.Context) (Availability, error) {
	return f(ctx)
}

func TestAvailabilityWatcher(t *testing.T) {
	tests := []struct {
		name    string
		initial Availability
		result  Availability
		err     error
		want    Availability
		changed bool
	}{
		{name: "unchanged", initial: Available, result: Available, want: Available},
		{name: "installed or granted", result: Available, want: Available, changed: true},
		{name: "removed", initial: Available, want: NotAvailable, changed: true},
		{name: "revoked", initial: Available, err: ErrMissingPermissions, want: NotAvailable, changed: true},
		{name: "discovery failure", initial: Available, err: errors.New("discovery failed"), want: Available},
		{name: "authorization failure", initial: Available, err: errors.New("authorization failed"), want: Available},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var changes []Availability
			w := NewAvailabilityWatcher(availabilityCheckFunc(func(ctx context.Context) (Availability, error) {
				_, ok := ctx.Deadline()
				assert.True(t, ok)
				return tt.result, tt.err
			}), tt.initial, func(a Availability) { changes = append(changes, a) })
			assert.False(t, w.NeedLeaderElection())
			w.check(t.Context())
			w.check(t.Context())
			assert.Equal(t, tt.want, w.Availability())
			if tt.changed {
				assert.Equal(t, []Availability{tt.want}, changes)
			} else {
				assert.Empty(t, changes)
			}
		})
	}
}

func TestAvailabilityWatcherPolling(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	results := []Availability{NotAvailable, Available, NotAvailable}
	var calls int
	changes := make(chan Availability, 2)
	w := NewAvailabilityWatcher(availabilityCheckFunc(func(context.Context) (Availability, error) {
		result := results[calls]
		calls++
		return result, nil
	}), NotAvailable, func(a Availability) {
		changes <- a
		if a == NotAvailable {
			cancel()
		}
	})
	w.interval = time.Millisecond
	done := make(chan error, 1)
	go func() { done <- w.Start(ctx) }()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("watcher did not stop")
	}
	assert.Equal(t, Available, <-changes)
	assert.Equal(t, NotAvailable, <-changes)
	assert.Equal(t, NotAvailable, w.Availability())
	assert.Equal(t, 3, calls)
}

func TestAvailabilityWatcherControllerRestart(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	w := NewAvailabilityWatcher(availabilityCheckFunc(func(context.Context) (Availability, error) {
		return Available, nil
	}), NotAvailable, func(Availability) { cancel() })
	require.NoError(t, w.Start(ctx))
	assert.ErrorIs(t, ctx.Err(), context.Canceled)
}

func TestAvailabilityWatcherCanceledCheck(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	w := NewAvailabilityWatcher(availabilityCheckFunc(func(context.Context) (Availability, error) {
		cancel()
		return Available, nil
	}), NotAvailable, func(Availability) { t.Error("unexpected availability change") })
	w.check(ctx)
	assert.Equal(t, NotAvailable, w.Availability())
}
