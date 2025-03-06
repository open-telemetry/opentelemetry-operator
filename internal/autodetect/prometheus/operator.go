// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package prometheus

// Availability represents what CRDs are available from the prometheus operator.
type Availability int

const (
	// NotAvailable represents the monitoring.coreos.com is not available.
	NotAvailable Availability = iota

	// Available represents the monitoring.coreos.com is available.
	Available
)

func (p Availability) String() string {
	return [...]string{"NotAvailable", "Available"}[p]
}
