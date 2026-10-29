// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	allocatorWatcher "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/watcher"
)

type fakePromWatcher struct {
	stop      chan struct{}
	available bool
	closed    bool
}

func newFakePromWatcher(available bool) *fakePromWatcher {
	return &fakePromWatcher{stop: make(chan struct{}), available: available}
}

func (w *fakePromWatcher) Watch(_ chan allocatorWatcher.Event, _ chan error) error {
	<-w.stop
	return nil
}

func (w *fakePromWatcher) Close() error {
	w.closed = true
	close(w.stop)
	return nil
}

func (w *fakePromWatcher) MissingCRDsAvailable() (bool, error) {
	return w.available, nil
}

func TestRunPromWatcher(t *testing.T) {
	tests := []struct {
		name            string
		available       bool
		newWatcherErr   error
		expectRecreated bool
	}{
		{
			name:            "missing CRDs available",
			available:       true,
			expectRecreated: true,
		},
		{
			name:      "missing CRDs not available",
			available: false,
		},
		{
			name:          "recreating fails",
			available:     true,
			newWatcherErr: errors.New("failed"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := newFakePromWatcher(tt.available)
			second := newFakePromWatcher(false)
			created := make(chan struct{}, 2)
			newWatcher := func() (promCRWatcher, error) {
				created <- struct{}{}
				if tt.newWatcherErr != nil {
					return nil, tt.newWatcherErr
				}
				return second, nil
			}
			recheck := make(chan time.Time)
			stop := make(chan struct{})
			done := make(chan error, 1)
			go func() {
				done <- runPromWatcher(first, newWatcher, recheck, nil, nil, stop)
			}()

			recheck <- time.Now()
			// A second tick is received only after the first one has been handled.
			recheck <- time.Now()

			close(stop)
			require.NoError(t, <-done)

			assert.True(t, first.closed)
			assert.Equal(t, tt.expectRecreated, second.closed)
			if tt.available {
				assert.NotEmpty(t, created)
			} else {
				assert.Empty(t, created)
			}
		})
	}
}
