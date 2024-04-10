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
