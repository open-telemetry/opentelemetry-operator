// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package gatewayapi

// ApiAvailability holds the auto-detected API availability state.
type ApiAvailability int

const (
	// ApiNotAvailable represents the API is not available.
	ApiNotAvailable ApiAvailability = iota

	// ApiAvailable represents the API is available.
	ApiAvailable
)

func (p ApiAvailability) String() string {
	return [...]string{"Available", "NotAvailable"}[p]
}
