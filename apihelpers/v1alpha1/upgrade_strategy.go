// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

type (
	// UpgradeStrategy represents how the operator will handle upgrades to the CR when a newer version of the operator is deployed
	// +kubebuilder:validation:Enum=automatic;none
	UpgradeStrategy string
)

const (
	// UpgradeStrategyAutomatic specifies that the operator will automatically apply upgrades to the CR.
	UpgradeStrategyAutomatic UpgradeStrategy = "automatic"

	// UpgradeStrategyNone specifies that the operator will not apply any upgrades to the CR.
	UpgradeStrategyNone UpgradeStrategy = "none"
)
