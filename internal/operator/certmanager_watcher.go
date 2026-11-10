// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"errors"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
)

const (
	certManagerAvailabilityCheckInterval = time.Minute
	certManagerAvailabilityCheckTimeout  = 30 * time.Second
)

type certManagerAvailabilityChecker interface {
	CertManagerAvailability(context.Context) (certmanager.Availability, error)
}

type certManagerRolloutRequester interface {
	requestRollout(context.Context) error
}

type certManagerAvailabilityWatcher struct {
	checker             certManagerAvailabilityChecker
	initialAvailability certmanager.Availability
	rollout             certManagerRolloutRequester
	logger              logr.Logger
	interval            time.Duration
}

// StartCertManagerAvailabilityWatcher starts a watcher for cert-manager availability changes.
func StartCertManagerAvailabilityWatcher(ctx context.Context, checker certManagerAvailabilityChecker, initialAvailability certmanager.Availability, clientset kubernetes.Interface) <-chan struct{} {
	watcher := &certManagerAvailabilityWatcher{
		checker:             checker,
		initialAvailability: initialAvailability,
		rollout:             &certManagerRollout{clientset: clientset},
		logger:              ctrl.Log.WithName("config").WithName("cert-manager"),
		interval:            certManagerAvailabilityCheckInterval,
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		watcher.run(ctx)
	}()
	return done
}

func (w *certManagerAvailabilityWatcher) run(ctx context.Context) {
	w.check(ctx)
	for ctx.Err() == nil {
		timer := time.NewTimer(wait.Jitter(w.interval, 0.2))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			w.check(ctx)
		}
	}
}

func (w *certManagerAvailabilityWatcher) check(ctx context.Context) {
	checkCtx, cancel := context.WithTimeout(ctx, certManagerAvailabilityCheckTimeout)
	defer cancel()

	availability, err := w.checker.CertManagerAvailability(checkCtx)
	if err != nil && !errors.Is(err, certmanager.ErrMissingPermissions) {
		if ctx.Err() == nil {
			w.logger.Error(err, "failed to check cert-manager availability")
		}
		return
	}
	if availability == w.initialAvailability || ctx.Err() != nil {
		return
	}

	if err := w.rollout.requestRollout(checkCtx); err != nil {
		if ctx.Err() == nil {
			w.logger.Error(err, "failed to roll out deployment after cert-manager availability changed")
		}
		return
	}
	w.logger.Info("cert-manager availability changed, deployment rollout requested", "previous", w.initialAvailability, "current", availability)
}
