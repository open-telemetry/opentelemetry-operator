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

// Package platform contains the availability of the OpenShift Routes API.
package openshift_routes

// Platform holds the auto-detected platform type.
type OpenShiftRoutesAvailability int

const (
	// Is not clear if OpenShift Routes are available.
	Unknown OpenShiftRoutesAvailability = iota

	// OpenShift Routes are available.
	Available OpenShiftRoutesAvailability = iota

	// OpenShift Routes are not available.
	NotAvailable OpenShiftRoutesAvailability = iota
)
