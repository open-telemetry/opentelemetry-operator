// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package openshift

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoutesAvailabilityString(t *testing.T) {
	tests := []struct {
		name         string
		availability RoutesAvailability
		want         string
	}{
		{"available", RoutesAvailable, "Available"},
		{"not available", RoutesNotAvailable, "NotAvailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.availability.String())
		})
	}
}
