// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

// HttpRouteConfig represents the HTTP route configuration for the Gateway API.
type HttpRouteConfig struct {
	// Enabled indicates whether the HTTP route configuration is enabled.
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Gateway specifies the name of the Gateway resource to associate with the HTTP route.
	Gateway string `json:"gateway" yaml:"gateway"`

	// GatewayNamespace specifies the namespace of the Gateway resource.
	// Default is the same namespace as the collector.
	GatewayNamespace string `json:"gatewayNamespace,omitempty" yaml:"gatewayNamespace,omitempty"`

	// Hostnames specifies the hostnames for the HTTP route.
	// Multiple hostnames can be specified to match requests with any of the given hostnames.
	// If empty, the route matches requests with any hostname.
	Hostnames []string `json:"hostnames,omitempty" yaml:"hostnames,omitempty"`
}
