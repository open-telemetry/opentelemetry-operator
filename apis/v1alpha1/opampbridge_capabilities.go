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
