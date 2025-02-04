// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

// Availability represents that the TargetAllocator CR is available in the cluster.
type Availability int

const (
	// NotAvailable TargetAllocator CR is available in the cluster.
	NotAvailable Availability = iota

	// Available TargetAllocator CR is available in the cluster.
	Available
)

func (p Availability) String() string {
	return [...]string{"NotAvailable", "Available"}[p]
}
