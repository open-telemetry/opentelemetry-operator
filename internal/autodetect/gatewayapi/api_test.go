// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package gatewayapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApiAvailabilityString(t *testing.T) {
	tests := []struct {
		name         string
		availability ApiAvailability
		want         string
	}{
		{"not available", ApiNotAvailable, "NotAvailable"},
		{"available", ApiAvailable, "Available"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.availability.String())
		})
	}
}
