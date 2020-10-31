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

// Package platform contains target platforms this operator might run on
package platform

// Platform holds the auto-detected platform type
type Platform int

const (
	// Unknown is used when the current platform can't be determined
	Unknown Platform = iota

	// OpenShift represents a platform of type OpenShift
	OpenShift Platform = iota

	// Kubernetes represents a platform of type Kubernetes
	Kubernetes
)

func (p Platform) String() string {
	return [...]string{"Unknown", "OpenShift", "Kubernetes"}[p]
}
