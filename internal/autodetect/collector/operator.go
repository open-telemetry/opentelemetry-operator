// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

// Availability represents that the OpenTelemetryCollector CR is available in the cluster.
type Availability int

const (
	// NotAvailable OpenTelemetryCollector CR is not available in the cluster.
	NotAvailable Availability = iota

	// Available OpenTelemetryCollector CR is available in the cluster.
	Available
)

func (p Availability) String() string {
	return [...]string{"NotAvailable", "Available"}[p]
}
