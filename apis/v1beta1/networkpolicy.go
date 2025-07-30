// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

// NetworkPolicy defines the configuration for NetworkPolicy.
type NetworkPolicy struct {
	// Enable enables the NetworkPolicy.
	// +optional
	// +kubebuilder:default:=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable NetworkPolicy"
	Enabled bool `json:"enabled,omitempty"`
}
