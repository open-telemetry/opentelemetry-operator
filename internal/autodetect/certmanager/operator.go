// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
