// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

type (
	// IngressType represents how a collector should be exposed (ingress vs route).
	// +kubebuilder:validation:Enum=ingress;route
	IngressType string
)

const (
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
)

// IngressRuleType defines how the collector receivers will be exposed in the Ingress.
//
// +kubebuilder:validation:Enum=path;subdomain
type IngressRuleType string

const (
	// IngressRuleTypePath configures Ingress to use single host with multiple paths.
	// This configuration might require additional ingress setting to rewrite paths.
	IngressRuleTypePath IngressRuleType = "path"
)
