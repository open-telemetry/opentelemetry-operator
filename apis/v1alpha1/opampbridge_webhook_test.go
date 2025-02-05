// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

var testScheme = scheme.Scheme

func TestOpAMPBridgeDefaultingWebhook(t *testing.T) {
	one := int32(1)
	five := int32(5)

	if err := AddToScheme(testScheme); err != nil {
		fmt.Printf("failed to register scheme: %v", err)
		os.Exit(1)
	}

	tests := []struct {
		name        string
		opampBridge OpAMPBridge
		expected    OpAMPBridge
	}{
		{
			name:        "all fields default",
			opampBridge: OpAMPBridge{},
			expected: OpAMPBridge{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: OpAMPBridgeSpec{
					Replicas:        &one,
					UpgradeStrategy: UpgradeStrategyAutomatic,
					Capabilities:    map[OpAMPBridgeCapability]bool{OpAMPBridgeCapabilityReportsStatus: true},
				},
			},
		},
		{
			name: "provided values in spec",
			opampBridge: OpAMPBridge{
				Spec: OpAMPBridgeSpec{
					Replicas:        &five,
					UpgradeStrategy: "adhoc",
				},
			},
			expected: OpAMPBridge{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: OpAMPBridgeSpec{
					Replicas:        &five,
					UpgradeStrategy: "adhoc",
					Capabilities:    map[OpAMPBridgeCapability]bool{OpAMPBridgeCapabilityReportsStatus: true},
				},
			},
		},
		{
			name: "enable ReportsStatus capability if not enabled already",
			opampBridge: OpAMPBridge{
				Spec: OpAMPBridgeSpec{
					Capabilities: map[OpAMPBridgeCapability]bool{
						OpAMPBridgeCapabilityReportsStatus: false,
					},
				},
			},
			expected: OpAMPBridge{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: OpAMPBridgeSpec{
					Replicas:        &one,
					UpgradeStrategy: UpgradeStrategyAutomatic,
					Capabilities: map[OpAMPBridgeCapability]bool{
						OpAMPBridgeCapabilityReportsStatus: true,
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			webhook := &OpAMPBridgeWebhook{
				logger: logr.Discard(),
				scheme: testScheme,
				cfg: config.New(
					config.WithCollectorImage("collector:v0.0.0"),
					config.WithTargetAllocatorImage("ta:v0.0.0"),
					config.WithOperatorOpAMPBridgeImage("opampbridge:v0.0.0"),
				),
			}
			ctx := context.Background()
			err := webhook.Default(ctx, &test.opampBridge)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, test.opampBridge)
		})
	}
}

func TestOpAMPBridgeValidatingWebhook(t *testing.T) {

	two := int32(2)

	tests := []struct { //nolint:govet
		name             string
		opampBridge      OpAMPBridge
		expectedErr      string
		expectedWarnings []string
	}{
		{
			name: "specify all required fields, should not return error",
			opampBridge: OpAMPBridge{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: OpAMPBridgeSpec{
					Endpoint: "ws://opamp-server:4320/v1/opamp",
					Capabilities: map[OpAMPBridgeCapability]bool{
						OpAMPBridgeCapabilityReportsStatus:                  true,
						OpAMPBridgeCapabilityAcceptsRemoteConfig:            true,
						OpAMPBridgeCapabilityReportsEffectiveConfig:         true,
						OpAMPBridgeCapabilityReportsOwnTraces:               true,
						OpAMPBridgeCapabilityReportsOwnMetrics:              true,
						OpAMPBridgeCapabilityReportsOwnLogs:                 true,
						OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings: true,
						OpAMPBridgeCapabilityAcceptsOtherConnectionSettings: true,
						OpAMPBridgeCapabilityAcceptsRestartCommand:          true,
						OpAMPBridgeCapabilityReportsHealth:                  true,
						OpAMPBridgeCapabilityReportsRemoteConfig:            true,
					},
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
			name: "empty capabilities",
			opampBridge: OpAMPBridge{
				Spec: OpAMPBridgeSpec{
					Endpoint: "ws://opamp-server:4320/v1/opamp",
				},
			},
			expectedErr: "the capabilities supported by OpAMP Bridge are not specified",
		},
		{
			name: "replica count greater than 1 should return error",
			opampBridge: OpAMPBridge{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: OpAMPBridgeSpec{
					Replicas: &two,
					Endpoint: "ws://opamp-server:4320/v1/opamp",
					Capabilities: map[OpAMPBridgeCapability]bool{
						OpAMPBridgeCapabilityReportsStatus:                  true,
						OpAMPBridgeCapabilityAcceptsRemoteConfig:            true,
						OpAMPBridgeCapabilityReportsEffectiveConfig:         true,
						OpAMPBridgeCapabilityReportsOwnTraces:               true,
						OpAMPBridgeCapabilityReportsOwnMetrics:              true,
						OpAMPBridgeCapabilityReportsOwnLogs:                 true,
						OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings: true,
						OpAMPBridgeCapabilityAcceptsOtherConnectionSettings: true,
						OpAMPBridgeCapabilityAcceptsRestartCommand:          true,
						OpAMPBridgeCapabilityReportsHealth:                  true,
						OpAMPBridgeCapabilityReportsRemoteConfig:            true,
					},
				},
			},
			expectedErr: "replica count must not be greater than 1",
		},
		{
			name: "invalid port name",
			opampBridge: OpAMPBridge{
				Spec: OpAMPBridgeSpec{
					Endpoint: "ws://opamp-server:4320/v1/opamp",
					Capabilities: map[OpAMPBridgeCapability]bool{
						OpAMPBridgeCapabilityReportsStatus:                  true,
						OpAMPBridgeCapabilityAcceptsRemoteConfig:            true,
						OpAMPBridgeCapabilityReportsEffectiveConfig:         true,
						OpAMPBridgeCapabilityReportsOwnTraces:               true,
						OpAMPBridgeCapabilityReportsOwnMetrics:              true,
						OpAMPBridgeCapabilityReportsOwnLogs:                 true,
						OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings: true,
						OpAMPBridgeCapabilityAcceptsOtherConnectionSettings: true,
						OpAMPBridgeCapabilityAcceptsRestartCommand:          true,
						OpAMPBridgeCapabilityReportsHealth:                  true,
						OpAMPBridgeCapabilityReportsRemoteConfig:            true,
					},
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
					Endpoint: "ws://opamp-server:4320/v1/opamp",
					Capabilities: map[OpAMPBridgeCapability]bool{
						OpAMPBridgeCapabilityReportsStatus:                  true,
						OpAMPBridgeCapabilityAcceptsRemoteConfig:            true,
						OpAMPBridgeCapabilityReportsEffectiveConfig:         true,
						OpAMPBridgeCapabilityReportsOwnTraces:               true,
						OpAMPBridgeCapabilityReportsOwnMetrics:              true,
						OpAMPBridgeCapabilityReportsOwnLogs:                 true,
						OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings: true,
						OpAMPBridgeCapabilityAcceptsOtherConnectionSettings: true,
						OpAMPBridgeCapabilityAcceptsRestartCommand:          true,
						OpAMPBridgeCapabilityReportsHealth:                  true,
						OpAMPBridgeCapabilityReportsRemoteConfig:            true,
					}, Ports: []v1.ServicePort{
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
					Endpoint: "ws://opamp-server:4320/v1/opamp",
					Capabilities: map[OpAMPBridgeCapability]bool{
						OpAMPBridgeCapabilityReportsStatus:                  true,
						OpAMPBridgeCapabilityAcceptsRemoteConfig:            true,
						OpAMPBridgeCapabilityReportsEffectiveConfig:         true,
						OpAMPBridgeCapabilityReportsOwnTraces:               true,
						OpAMPBridgeCapabilityReportsOwnMetrics:              true,
						OpAMPBridgeCapabilityReportsOwnLogs:                 true,
						OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings: true,
						OpAMPBridgeCapabilityAcceptsOtherConnectionSettings: true,
						OpAMPBridgeCapabilityAcceptsRestartCommand:          true,
						OpAMPBridgeCapabilityReportsHealth:                  true,
						OpAMPBridgeCapabilityReportsRemoteConfig:            true,
					},
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
		test := test
		t.Run(test.name, func(t *testing.T) {
			webhook := &OpAMPBridgeWebhook{
				logger: logr.Discard(),
				scheme: testScheme,
				cfg: config.New(
					config.WithCollectorImage("collector:v0.0.0"),
					config.WithTargetAllocatorImage("ta:v0.0.0"),
					config.WithOperatorOpAMPBridgeImage("opampbridge:v0.0.0"),
				),
			}
			ctx := context.Background()
			warnings, err := webhook.ValidateCreate(ctx, &test.opampBridge)
			if test.expectedErr == "" {
				assert.NoError(t, err)
				return
			}
			if len(test.expectedWarnings) == 0 {
				assert.Empty(t, warnings, test.expectedWarnings)
			} else {
				assert.ElementsMatch(t, warnings, test.expectedWarnings)
			}
			assert.ErrorContains(t, err, test.expectedErr)
		})
	}
}
