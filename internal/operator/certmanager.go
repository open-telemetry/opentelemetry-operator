// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

// SetupCertManagerAvailabilityWatcher updates standalone webhooks when cancel is nil, otherwise it stops the controller on changes.
func SetupCertManagerAvailabilityWatcher(result *SetupResult, cancel context.CancelFunc) error {
	if !featuregate.EnableCertManagerAvailabilityWatcher.IsEnabled() {
		return nil
	}
	var onChange func(certmanager.Availability)
	if cancel != nil {
		onChange = func(certmanager.Availability) {
			setupLog.Info("cert-manager availability changed, restarting controller")
			cancel()
		}
	}
	watcher := certmanager.NewAvailabilityWatcher(result.Autodetector, result.Config.CertManagerAvailability, onChange)
	if cancel == nil {
		result.Config.Internal.CertManagerAvailability = watcher.Availability
	}
	if err := result.Manager.Add(watcher); err != nil {
		return fmt.Errorf("unable to setup cert-manager availability watcher: %w", err)
	}
	return nil
}
