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

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOpAMPBridgeDefaultingWebhook(t *testing.T) {
	one := int32(1)
	five := int32(5)

	tests := []struct {
		name        string
		opampBridge OpAMPBridge
		expected    OpAMPBridge
	}{
		{
			name:        "provide only required values in spec",
			opampBridge: OpAMPBridge{},
			expected: OpAMPBridge{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: OpAMPBridgeSpec{
					Replicas:        &one,
					UpgradeStrategy: UpgradeStrategyAutomatic,
				},
			},
		},
		{
			name: "provided optional values in spec",
			opampBridge: OpAMPBridge{
				Spec: OpAMPBridgeSpec{
					Replicas:        &five,
					UpgradeStrategy: "adhoc",
				},
			},
			expected: OpAMPBridge{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": "opentelemetry-operator",
					},
				},
				Spec: OpAMPBridgeSpec{
					Replicas:        &five,
					UpgradeStrategy: "adhoc",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.opampBridge.Default()
			assert.Equal(t, test.expected, test.opampBridge)
		})
	}
}

func TestOpAMPBridgeValidatingWebhook(t *testing.T) {

	two := int32(2)

	tests := []struct { //nolint:govet
		name        string
		opampBridge OpAMPBridge
		expectedErr string
	}{
		{
			name: "specify all required fields, should not return error",
			opampBridge: OpAMPBridge{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: OpAMPBridgeSpec{
					Endpoint:     "ws://opamp-server:4320/v1/opamp",
					Protocol:     "wss",
					Capabilities: []OpAMPBridgeCapability{OpAMPBridgeCapabilityAcceptsRemoteConfig, OpAMPBridgeCapabilityReportsEffectiveConfig, OpAMPBridgeCapabilityReportsOwnTraces, OpAMPBridgeCapabilityReportsOwnMetrics, OpAMPBridgeCapabilityReportsOwnLogs, OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings, OpAMPBridgeCapabilityAcceptsOtherConnectionSettings, OpAMPBridgeCapabilityAcceptsRestartCommand, OpAMPBridgeCapabilityReportsHealth, OpAMPBridgeCapabilityReportsRemoteConfig},
				},
			},
		},
		{
			name: "empty OpAMP Server endpoint",
			opampBridge: OpAMPBridge{
				Spec: OpAMPBridgeSpec{
					Endpoint: "",
				},
			},
			expectedErr: "the OpAMP server endpoint is not specified",
		},
		{
			name: "empty transport for OpAMP protocol",
			opampBridge: OpAMPBridge{
				Spec: OpAMPBridgeSpec{
					Endpoint: "ws://opamp-server:4320/v1/opamp",
					Protocol: "",
				},
			},
			expectedErr: "the transport for OpAMP server protocol is not specified",
		},
		{
			name: "empty capabilities",
			opampBridge: OpAMPBridge{
				Spec: OpAMPBridgeSpec{
					Endpoint: "ws://opamp-server:4320/v1/opamp",
					Protocol: "wss",
				},
			},
			expectedErr: "the capabilities supported by OpAMP Bridge are not specified",
		},
		{
			name: "required capabilities not enabled",
			opampBridge: OpAMPBridge{
				Spec: OpAMPBridgeSpec{
					Endpoint:     "ws://opamp-server:4320/v1/opamp",
					Protocol:     "wss",
					Capabilities: []OpAMPBridgeCapability{OpAMPBridgeCapabilityReportsEffectiveConfig, OpAMPBridgeCapabilityReportsOwnTraces},
				},
			},
			expectedErr: "required capabilities must be enabled",
		},
		{
			name: "replica count greater than 1 should return error",
			opampBridge: OpAMPBridge{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: OpAMPBridgeSpec{
					Replicas:     &two,
					Endpoint:     "ws://opamp-server:4320/v1/opamp",
					Protocol:     "wss",
					Capabilities: []OpAMPBridgeCapability{OpAMPBridgeCapabilityAcceptsRemoteConfig, OpAMPBridgeCapabilityReportsEffectiveConfig, OpAMPBridgeCapabilityReportsOwnTraces, OpAMPBridgeCapabilityReportsOwnMetrics, OpAMPBridgeCapabilityReportsOwnLogs, OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings, OpAMPBridgeCapabilityAcceptsOtherConnectionSettings, OpAMPBridgeCapabilityAcceptsRestartCommand, OpAMPBridgeCapabilityReportsHealth, OpAMPBridgeCapabilityReportsRemoteConfig},
				},
			},
			expectedErr: "replica count must not be greater than 1",
		},
		{
			name: "invalid port name",
			opampBridge: OpAMPBridge{
				Spec: OpAMPBridgeSpec{
					Endpoint:     "ws://opamp-server:4320/v1/opamp",
					Protocol:     "wss",
					Capabilities: []OpAMPBridgeCapability{OpAMPBridgeCapabilityAcceptsRemoteConfig, OpAMPBridgeCapabilityReportsEffectiveConfig, OpAMPBridgeCapabilityReportsOwnTraces, OpAMPBridgeCapabilityReportsOwnMetrics, OpAMPBridgeCapabilityReportsOwnLogs, OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings, OpAMPBridgeCapabilityAcceptsOtherConnectionSettings, OpAMPBridgeCapabilityAcceptsRestartCommand, OpAMPBridgeCapabilityReportsHealth, OpAMPBridgeCapabilityReportsRemoteConfig},
					Ports: []v1.ServicePort{
						{
							// this port name contains a non alphanumeric character, which is invalid.
							Name:     "-test🦄port",
							Port:     12345,
							Protocol: v1.ProtocolTCP,
						},
					},
				},
			},
			expectedErr: "the OpAMPBridge Spec Ports configuration is incorrect",
		},
		{
			name: "invalid port name, too long",
			opampBridge: OpAMPBridge{
				Spec: OpAMPBridgeSpec{
					Endpoint:     "ws://opamp-server:4320/v1/opamp",
					Protocol:     "wss",
					Capabilities: []OpAMPBridgeCapability{OpAMPBridgeCapabilityAcceptsRemoteConfig, OpAMPBridgeCapabilityReportsEffectiveConfig, OpAMPBridgeCapabilityReportsOwnTraces, OpAMPBridgeCapabilityReportsOwnMetrics, OpAMPBridgeCapabilityReportsOwnLogs, OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings, OpAMPBridgeCapabilityAcceptsOtherConnectionSettings, OpAMPBridgeCapabilityAcceptsRestartCommand, OpAMPBridgeCapabilityReportsHealth, OpAMPBridgeCapabilityReportsRemoteConfig},
					Ports: []v1.ServicePort{
						{
							Name: "aaaabbbbccccdddd", // len: 16, too long
							Port: 5555,
						},
					},
				},
			},
			expectedErr: "the OpAMPBridge Spec Ports configuration is incorrect",
		},
		{
			name: "invalid port num",
			opampBridge: OpAMPBridge{
				Spec: OpAMPBridgeSpec{
					Endpoint:     "ws://opamp-server:4320/v1/opamp",
					Protocol:     "wss",
					Capabilities: []OpAMPBridgeCapability{OpAMPBridgeCapabilityAcceptsRemoteConfig, OpAMPBridgeCapabilityReportsEffectiveConfig, OpAMPBridgeCapabilityReportsOwnTraces, OpAMPBridgeCapabilityReportsOwnMetrics, OpAMPBridgeCapabilityReportsOwnLogs, OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings, OpAMPBridgeCapabilityAcceptsOtherConnectionSettings, OpAMPBridgeCapabilityAcceptsRestartCommand, OpAMPBridgeCapabilityReportsHealth, OpAMPBridgeCapabilityReportsRemoteConfig},
					Ports: []v1.ServicePort{
						{
							Name: "aaaabbbbccccddd", // len: 15
							// no port set means it's 0, which is invalid
						},
					},
				},
			},
			expectedErr: "the OpAMPBridge Spec Ports configuration is incorrect",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.opampBridge.validateCRDSpec()
			if test.expectedErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, test.expectedErr)
		})
	}
}
