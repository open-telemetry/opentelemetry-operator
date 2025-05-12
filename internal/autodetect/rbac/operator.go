// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package rbac

// Availability represents that the opeerator service account has permissions to create RBAC resources.
type Availability int

const (
	// NotAvailable RBAC permissions are not available.
	NotAvailable Availability = iota

	// Available NotAvailable RBAC permissions are available.
	Available
)

func (p Availability) String() string {
	return [...]string{"NotAvailable", "Available"}[p]
}
