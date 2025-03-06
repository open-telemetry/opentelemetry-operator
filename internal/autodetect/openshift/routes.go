// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package openshift

// RoutesAvailability holds the auto-detected OpenShift Routes availability API.
type RoutesAvailability int

const (
	// RoutesAvailable represents the route.openshift.io API is available.
	RoutesAvailable RoutesAvailability = iota

	// RoutesNotAvailable represents the route.openshift.io API is not available.
	RoutesNotAvailable
)

func (p RoutesAvailability) String() string {
	return [...]string{"Available", "NotAvailable"}[p]
}
