// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opampbridge

// Availability represents that the OpAmpBridge CR is available in the cluster.
type Availability int

const (
	// NotAvailable OpAmpBridge CR is not available in the cluster.
	NotAvailable Availability = iota

	// Available OpAmpBridge CR is available in the cluster.
	Available
)

func (p Availability) String() string {
	return [...]string{"NotAvailable", "Available"}[p]
}
