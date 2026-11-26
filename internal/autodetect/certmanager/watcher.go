// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package certmanager

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
)

const availabilityCheckTimeout = 30 * time.Second

type availabilityChecker interface {
	CertManagerAvailability(context.Context) (Availability, error)
}

// AvailabilityWatcher checks cert-manager availability on every replica.
type AvailabilityWatcher struct {
	checker  availabilityChecker
	current  atomic.Int64
	onChange func(Availability)
	interval time.Duration
}

// NewAvailabilityWatcher creates a watcher with the startup availability.
func NewAvailabilityWatcher(checker availabilityChecker, initial Availability, onChange func(Availability)) *AvailabilityWatcher {
	w := &AvailabilityWatcher{checker: checker, onChange: onChange, interval: time.Minute}
	w.current.Store(int64(initial))
	return w
}

// Availability returns the last confirmed availability and is safe for concurrent use.
func (w *AvailabilityWatcher) Availability() Availability {
	return Availability(w.current.Load())
}

// NeedLeaderElection reports whether the watcher requires leader election.
func (*AvailabilityWatcher) NeedLeaderElection() bool {
	return false
}

// Start checks availability until the manager stops.
func (w *AvailabilityWatcher) Start(ctx context.Context) error {
	wait.JitterUntilWithContext(ctx, w.check, w.interval, 0.2, true)
	return nil
}

func (w *AvailabilityWatcher) check(ctx context.Context) {
	checkCtx, cancel := context.WithTimeout(ctx, availabilityCheckTimeout)
	defer cancel()

	availability, err := w.checker.CertManagerAvailability(checkCtx)
	logger := ctrl.LoggerFrom(ctx).WithName("cert-manager")
	if err != nil && !errors.Is(err, ErrMissingPermissions) {
		if ctx.Err() == nil {
			logger.Error(err, "failed to check cert-manager availability")
		}
		return
	}
	if ctx.Err() != nil || availability == w.Availability() {
		return
	}
	w.current.Store(int64(availability))
	logger.Info("cert-manager availability changed", "availability", availability)
	if w.onChange != nil {
		w.onChange(availability)
	}
}
