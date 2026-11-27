// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	allocatorWatcher "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/watcher"
)

const testTimeout = 5 * time.Second

// fakePromWatcher is a promCRWatcher whose Watch blocks until Close is called, unless watchErr
// is sent first, in which case Watch returns that error on its own.
type fakePromWatcher struct {
	name string
	// watching is closed once Watch has been called.
	watching chan struct{}
	stop     chan struct{}
	// watchErr, when it receives, makes Watch return the received error without Close being called.
	watchErr chan error
	// resultAfterClose is what Watch returns after Close.
	resultAfterClose error
	closeErr         error
	available        func() (bool, error)

	closeCalls atomic.Int32
	checkCalls atomic.Int32
}

func newFakePromWatcher(name string) *fakePromWatcher {
	return &fakePromWatcher{
		name:      name,
		watching:  make(chan struct{}),
		stop:      make(chan struct{}),
		watchErr:  make(chan error, 1),
		available: func() (bool, error) { return false, nil },
	}
}

func (w *fakePromWatcher) Watch(_ chan allocatorWatcher.Event, _ chan error) error {
	close(w.watching)
	select {
	case <-w.stop:
		return w.resultAfterClose
	case err := <-w.watchErr:
		return err
	}
}

func (w *fakePromWatcher) Close() error {
	if w.closeCalls.Add(1) == 1 {
		close(w.stop)
	}
	return w.closeErr
}

func (w *fakePromWatcher) MissingCRDsAvailable() (bool, error) {
	w.checkCalls.Add(1)
	return w.available()
}

func (w *fakePromWatcher) waitWatching(t *testing.T) {
	t.Helper()
	select {
	case <-w.watching:
	case <-time.After(testTimeout):
		t.Fatalf("watcher %s was not started", w.name)
	}
}

// logRecorder captures log lines written through a logr.Logger.
type logRecorder struct {
	mu    sync.Mutex
	lines []string
}

func (r *logRecorder) logger() logr.Logger {
	return funcr.New(func(prefix, args string) {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.lines = append(r.lines, prefix+" "+args)
	}, funcr.Options{})
}

func (r *logRecorder) contains(substr string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, line := range r.lines {
		if strings.Contains(line, substr) {
			return true
		}
	}
	return false
}

func (r *logRecorder) empty() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.lines) == 0
}

// promWatcherHarness runs runPromWatcher in the background with a sequence of watchers (or errors)
// returned from newWatcher.
type promWatcherHarness struct {
	logs        *logRecorder
	recheck     chan time.Time
	stop        chan struct{}
	done        chan error
	createCalls atomic.Int32
}

// newWatcherResult is one result of the newWatcher function.
type newWatcherResult struct {
	watcher *fakePromWatcher
	err     error
}

func startHarness(t *testing.T, results ...newWatcherResult) *promWatcherHarness {
	t.Helper()
	h := &promWatcherHarness{
		logs:    &logRecorder{},
		recheck: make(chan time.Time),
		stop:    make(chan struct{}),
		done:    make(chan error, 1),
	}
	newWatcher := func() (promCRWatcher, error) {
		i := int(h.createCalls.Add(1)) - 1
		require.Less(t, i, len(results), "unexpected call to newWatcher")
		if results[i].err != nil {
			return nil, results[i].err
		}
		return results[i].watcher, nil
	}
	go func() {
		h.done <- runPromWatcher(h.logs.logger(), newWatcher, h.recheck, nil, nil, h.stop)
	}()
	return h
}

// tick sends two recheck ticks. The recheck channel is unbuffered, so the second send is received
// only once the first tick was fully handled. The second tick may still be in progress when tick
// returns; its effects are observable only after shutdown or wait.
func (h *promWatcherHarness) tick(t *testing.T) {
	t.Helper()
	for range 2 {
		select {
		case h.recheck <- time.Now():
		case <-time.After(testTimeout):
			t.Fatal("recheck tick was not received")
		}
	}
}

func (h *promWatcherHarness) wait(t *testing.T) error {
	t.Helper()
	select {
	case err := <-h.done:
		return err
	case <-time.After(testTimeout):
		t.Fatal("runPromWatcher did not return")
		return nil
	}
}

func (h *promWatcherHarness) shutdown(t *testing.T) error {
	t.Helper()
	close(h.stop)
	return h.wait(t)
}

func TestRunPromWatcher(t *testing.T) {
	t.Run("initial creation fails", func(t *testing.T) {
		h := startHarness(t, newWatcherResult{err: errors.New("creation failed")})

		require.EqualError(t, h.wait(t), "creation failed")
		assert.Equal(t, int32(1), h.createCalls.Load())
		assert.True(t, h.logs.empty())
	})

	t.Run("stop closes the watcher", func(t *testing.T) {
		w := newFakePromWatcher("initial")
		h := startHarness(t, newWatcherResult{watcher: w})
		w.waitWatching(t)

		require.NoError(t, h.shutdown(t))
		assert.Equal(t, int32(1), w.closeCalls.Load())
		assert.True(t, h.logs.empty())
	})

	t.Run("stop logs close and watch errors", func(t *testing.T) {
		w := newFakePromWatcher("initial")
		w.closeErr = errors.New("close failed")
		w.resultAfterClose = errors.New("sync failed")
		h := startHarness(t, newWatcherResult{watcher: w})
		w.waitWatching(t)

		require.NoError(t, h.shutdown(t))
		assert.True(t, h.logs.contains("Prometheus watcher failed to close"))
		assert.True(t, h.logs.contains("close failed"))
		assert.True(t, h.logs.contains("Prometheus watcher exited with an error"))
		assert.True(t, h.logs.contains("sync failed"))
	})

	t.Run("watch error is returned and the watcher closed", func(t *testing.T) {
		w := newFakePromWatcher("initial")
		w.watchErr <- errors.New("watch failed")
		w.closeErr = errors.New("close failed")
		h := startHarness(t, newWatcherResult{watcher: w})

		require.EqualError(t, h.wait(t), "watch failed")
		assert.Equal(t, int32(1), w.closeCalls.Load())
		assert.True(t, h.logs.contains("Prometheus watcher failed to close"))
		assert.True(t, h.logs.contains("close failed"))
	})

	t.Run("watch returning nil is returned", func(t *testing.T) {
		w := newFakePromWatcher("initial")
		w.watchErr <- nil
		h := startHarness(t, newWatcherResult{watcher: w})

		require.NoError(t, h.wait(t))
		assert.Equal(t, int32(1), w.closeCalls.Load())
	})

	t.Run("missing CRDs still absent keeps the watcher", func(t *testing.T) {
		w := newFakePromWatcher("initial")
		h := startHarness(t, newWatcherResult{watcher: w})
		w.waitWatching(t)

		h.tick(t)
		require.NoError(t, h.shutdown(t))

		assert.Equal(t, int32(2), w.checkCalls.Load())
		assert.Equal(t, int32(1), h.createCalls.Load())
		// Closed only by the shutdown.
		assert.Equal(t, int32(1), w.closeCalls.Load())
		assert.True(t, h.logs.empty())
	})

	t.Run("availability check error keeps the watcher", func(t *testing.T) {
		w := newFakePromWatcher("initial")
		w.available = func() (bool, error) { return false, errors.New("discovery failed") }
		h := startHarness(t, newWatcherResult{watcher: w})
		w.waitWatching(t)

		h.tick(t)
		require.NoError(t, h.shutdown(t))

		assert.Equal(t, int32(2), w.checkCalls.Load())
		assert.Equal(t, int32(1), h.createCalls.Load())
		// Closed only by the shutdown.
		assert.Equal(t, int32(1), w.closeCalls.Load())
		assert.True(t, h.logs.contains("Unable to check availability of Prometheus CRDs"))
		assert.True(t, h.logs.contains("discovery failed"))
	})

	t.Run("missing CRDs available replaces the watcher", func(t *testing.T) {
		initial := newFakePromWatcher("initial")
		initial.available = func() (bool, error) { return true, nil }
		replacement := newFakePromWatcher("replacement")
		h := startHarness(t,
			newWatcherResult{watcher: initial},
			newWatcherResult{watcher: replacement},
		)
		initial.waitWatching(t)

		h.tick(t)
		replacement.waitWatching(t)
		// The initial watcher is closed before the replacement starts watching.
		assert.Equal(t, int32(1), initial.closeCalls.Load())

		require.NoError(t, h.shutdown(t))
		assert.Equal(t, int32(2), h.createCalls.Load())
		assert.Equal(t, int32(1), initial.closeCalls.Load())
		assert.Equal(t, int32(1), replacement.closeCalls.Load())
		// The second tick is checked against the replacement, not the closed watcher.
		assert.Equal(t, int32(1), initial.checkCalls.Load())
		assert.Equal(t, int32(1), replacement.checkCalls.Load())
		assert.True(t, h.logs.contains("recreating the Prometheus watcher"))
	})

	t.Run("replacement watch error is returned", func(t *testing.T) {
		initial := newFakePromWatcher("initial")
		initial.available = func() (bool, error) { return true, nil }
		replacement := newFakePromWatcher("replacement")
		h := startHarness(t,
			newWatcherResult{watcher: initial},
			newWatcherResult{watcher: replacement},
		)
		initial.waitWatching(t)
		h.tick(t)
		replacement.waitWatching(t)

		replacement.watchErr <- errors.New("replacement failed")

		require.EqualError(t, h.wait(t), "replacement failed")
		assert.Equal(t, int32(1), initial.closeCalls.Load())
		assert.Equal(t, int32(1), replacement.closeCalls.Load())
	})

	t.Run("failed replacement keeps the watcher and is retried", func(t *testing.T) {
		initial := newFakePromWatcher("initial")
		initial.available = func() (bool, error) { return true, nil }
		replacement := newFakePromWatcher("replacement")
		h := startHarness(t,
			newWatcherResult{watcher: initial},
			newWatcherResult{err: errors.New("recreation failed")},
			newWatcherResult{watcher: replacement},
		)
		initial.waitWatching(t)

		// First tick: recreation fails, second tick: recreation succeeds.
		h.tick(t)
		replacement.waitWatching(t)
		require.NoError(t, h.shutdown(t))

		assert.Equal(t, int32(3), h.createCalls.Load())
		assert.Equal(t, int32(2), initial.checkCalls.Load())
		assert.Equal(t, int32(1), initial.closeCalls.Load())
		assert.Equal(t, int32(1), replacement.closeCalls.Load())
		assert.True(t, h.logs.contains("Unable to recreate the Prometheus watcher, keeping the current one"))
		assert.True(t, h.logs.contains("recreation failed"))
	})

	t.Run("nil recheck channel disables the recheck", func(t *testing.T) {
		w := newFakePromWatcher("initial")
		w.available = func() (bool, error) { return true, nil }
		done := make(chan error, 1)
		stop := make(chan struct{})
		go func() {
			done <- runPromWatcher(logr.Discard(), func() (promCRWatcher, error) { return w, nil }, nil, nil, nil, stop)
		}()
		w.waitWatching(t)

		close(stop)
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(testTimeout):
			t.Fatal("runPromWatcher did not return")
		}
		assert.Equal(t, int32(0), w.checkCalls.Load())
	})
}
