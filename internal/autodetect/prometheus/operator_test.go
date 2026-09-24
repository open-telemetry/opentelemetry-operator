// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prometheus_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
)

func TestAvailableCRDsMethods(t *testing.T) {
	for _, tt := range []struct {
		name                    string
		crds                    prometheus.AvailableCRDs
		available               bool
		availableServiceMonitor bool
		availablePodMonitor     bool
	}{
		{
			name:                    "nil",
			crds:                    nil,
			available:               false,
			availableServiceMonitor: false,
			availablePodMonitor:     false,
		},
		{
			name:                    "empty",
			crds:                    prometheus.AvailableCRDs{},
			available:               false,
			availableServiceMonitor: false,
			availablePodMonitor:     false,
		},
		{
			name:                    "servicemonitors only",
			crds:                    prometheus.AvailableCRDs{"servicemonitors"},
			available:               true,
			availableServiceMonitor: true,
			availablePodMonitor:     false,
		},
		{
			name:                    "podmonitors only",
			crds:                    prometheus.AvailableCRDs{"podmonitors"},
			available:               true,
			availableServiceMonitor: false,
			availablePodMonitor:     true,
		},
		{
			name:                    "all CRDs",
			crds:                    prometheus.AvailableCRDs{"servicemonitors", "podmonitors", "probes", "scrapeconfigs"},
			available:               true,
			availableServiceMonitor: true,
			availablePodMonitor:     true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.available, tt.crds.Available())
			assert.Equal(t, tt.availableServiceMonitor, tt.crds.AvailableServiceMonitor())
			assert.Equal(t, tt.availablePodMonitor, tt.crds.AvailablePodMonitor())
		})
	}
}
