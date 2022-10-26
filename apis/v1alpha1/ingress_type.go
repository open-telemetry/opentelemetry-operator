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

package v1alpha1

type (
	// IngressType represents how a collector should be exposed (ingress vs route).
	// +kubebuilder:validation:Enum=ingress;openshift-route-v1-insecure;openshift-route-v1-edge;openshift-route-v1-passthrough
	IngressType string
)

const (
	// IngressTypeNginx specifies that an ingress entry should be created.
	IngressTypeNginx IngressType = "ingress"
	// IngressTypeOpenshiftRoute specifies that an route entry should be created.
	IngressTypeRouteV1Insecure    IngressType = "openshift-route-v1-insecure"
	IngressTypeRouteV1Edge        IngressType = "openshift-route-v1-edge"
	IngressTypeRouteV1Passthrough IngressType = "openshift-route-v1-passthrough"
)
