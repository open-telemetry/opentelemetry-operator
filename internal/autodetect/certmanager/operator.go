// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package certmanager

// Availability represents that the Cert Manager CRDs are installed and the operator's service account has permissions to manage cert-manager resources.
type Availability int

const (
	// NotAvailable Cert Manager CRDs or RBAC permissions to manage cert-manager certificates are not available.
	NotAvailable Availability = iota

	// Available Cert Manager CRDs and RBAC permissions to manage cert-manager certificates are available.
	Available
)

func (p Availability) String() string {
	return [...]string{"NotAvailable", "Available"}[p]
}
