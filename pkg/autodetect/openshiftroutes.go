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

// Package autodetect is for auto-detecting traits from the environment (APIs, ...).
package autodetect

// OpenShiftRoutesAvailability holds the auto-detected OpenShift Routes availability API.
type OpenShiftRoutesAvailability int

const (
	// OpenShiftRoutesAvailable represents the route.openshift.io API is available.
	OpenShiftRoutesAvailable OpenShiftRoutesAvailability = iota

	// OpenShiftRoutesNotAvailable represents the route.openshift.io API is not available.
	OpenShiftRoutesNotAvailable
)

func (p OpenShiftRoutesAvailability) String() string {
	return [...]string{"Available", "NotAvailable"}[p]
}
