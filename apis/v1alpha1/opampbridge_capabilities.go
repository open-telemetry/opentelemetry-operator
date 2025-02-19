// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

type (
	// OpAMPBridgeCapability represents capability supported by OpAMP Bridge.
	// +kubebuilder:validation:Enum=AcceptsRemoteConfig;ReportsEffectiveConfig;ReportsOwnTraces;ReportsOwnMetrics;ReportsOwnLogs;AcceptsOpAMPConnectionSettings;AcceptsOtherConnectionSettings;AcceptsRestartCommand;ReportsHealth;ReportsRemoteConfig
	OpAMPBridgeCapability string
)

const (
	OpAMPBridgeCapabilityReportsStatus                  OpAMPBridgeCapability = "ReportsStatus"
	OpAMPBridgeCapabilityAcceptsRemoteConfig            OpAMPBridgeCapability = "AcceptsRemoteConfig"
	OpAMPBridgeCapabilityReportsEffectiveConfig         OpAMPBridgeCapability = "ReportsEffectiveConfig"
	OpAMPBridgeCapabilityReportsOwnTraces               OpAMPBridgeCapability = "ReportsOwnTraces"
	OpAMPBridgeCapabilityReportsOwnMetrics              OpAMPBridgeCapability = "ReportsOwnMetrics"
	OpAMPBridgeCapabilityReportsOwnLogs                 OpAMPBridgeCapability = "ReportsOwnLogs"
	OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings OpAMPBridgeCapability = "AcceptsOpAMPConnectionSettings"
	OpAMPBridgeCapabilityAcceptsOtherConnectionSettings OpAMPBridgeCapability = "AcceptsOtherConnectionSettings"
	OpAMPBridgeCapabilityAcceptsRestartCommand          OpAMPBridgeCapability = "AcceptsRestartCommand"
	OpAMPBridgeCapabilityReportsHealth                  OpAMPBridgeCapability = "ReportsHealth"
	OpAMPBridgeCapabilityReportsRemoteConfig            OpAMPBridgeCapability = "ReportsRemoteConfig"
)
