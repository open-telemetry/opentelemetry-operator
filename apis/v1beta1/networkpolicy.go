// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

// NetworkPolicy defines the configuration for NetworkPolicy.
type NetworkPolicy struct {
	// Enable enables the NetworkPolicy.
	// The default value is taken from the operator feature-gate `--feature-gates=+operand.networkpolicy`.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable NetworkPolicy"
	Enabled *bool `json:"enabled,omitempty"`

	// APIServerCIDRs are the CIDRs through which the pods reach the Kubernetes API server.
	// When set, the NetworkPolicy allows egress to these CIDRs on all ports instead of to the API server
	// addresses detected from the `default/kubernetes` EndpointSlices. Use it when the pods reach the API server
	// through other addresses, for example in managed or NAT setups.
	// Only the TargetAllocator NetworkPolicy restricts egress. For an OpenTelemetryCollector, the value is passed
	// to the TargetAllocator it creates.
	// +optional
	// +listType=atomic
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="API Server CIDRs"
	APIServerCIDRs []string `json:"apiServerCIDRs,omitempty"`
}
