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
	// +kubebuilder:validation:Enum=ingress;route
	IngressType string
)

const (
	// IngressTypeNginx specifies that an ingress entry should be created.
	IngressTypeNginx IngressType = "ingress"
	// IngressTypeOpenshiftRoute specifies that an route entry should be created.
	IngressTypeRoute IngressType = "route"
)

type (
	// TLSRouteTerminationType is used to indicate which tls settings should be used.
	// +kubebuilder:validation:Enum=insecure;edge;passthrough;reencrypt
	TLSRouteTerminationType string
)

const (
	// TLSRouteTerminationTypeInsecure indicates that insecure connections are allowed.
	TLSRouteTerminationTypeInsecure TLSRouteTerminationType = "insecure"
	// TLSRouteTerminationTypeEdge indicates that encryption should be terminated
	// at the edge router.
	TLSRouteTerminationTypeEdge TLSRouteTerminationType = "edge"
	// TLSTerminationPassthrough indicates that the destination service is
	// responsible for decrypting traffic.
	TLSRouteTerminationTypePassthrough TLSRouteTerminationType = "passthrough"
	// TLSTerminationReencrypt indicates that traffic will be decrypted on the edge
	// and re-encrypt using a new certificate.
	TLSRouteTerminationTypeReencrypt TLSRouteTerminationType = "reencrypt"
)

// IngressRuleType defines how the collector receivers will be exposed in the Ingress.
//
// +kubebuilder:validation:Enum=path;subdomain
type IngressRuleType string

const (
	// IngressRuleTypePath configures Ingress to use single host with multiple paths.
	// This configuration might require additional ingress setting to rewrite paths.
	IngressRuleTypePath IngressRuleType = "path"

	// IngressRuleTypeSubdomain configures Ingress to use multiple hosts - one for each exposed
	// receiver port. The port name is used as a subdomain for the host defined in the Ingress e.g. otlp-http.example.com.
	IngressRuleTypeSubdomain IngressRuleType = "subdomain"
)
