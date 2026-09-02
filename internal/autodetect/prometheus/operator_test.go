// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAvailabilityString(t *testing.T) {
	tests := []struct {
		name         string
		availability Availability
		want         string
	}{
		{"not available", NotAvailable, "NotAvailable"},
		{"available", Available, "Available"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.availability.String())
		})
	}
}
