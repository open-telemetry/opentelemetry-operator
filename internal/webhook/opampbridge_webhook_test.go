// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package webhook

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

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

var testScheme = scheme.Scheme

func TestOpAMPBridgeDefaultingWebhook(t *testing.T) {
	one := int32(1)
	five := int32(5)

	if err := v1alpha1.AddToScheme(testScheme); err != nil {
		fmt.Printf("failed to register scheme: %v", err)
		os.Exit(1)
	}

	tests := []struct {
		name        string
		opampBridge v1alpha1.OpAMPBridge
		expected    v1alpha1.OpAMPBridge
	}{
		{
			name:        "all fields default",
			opampBridge: v1alpha1.OpAMPBridge{},
			expected: v1alpha1.OpAMPBridge{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: v1alpha1.OpAMPBridgeSpec{
					Replicas:        &one,
					UpgradeStrategy: v1alpha1.UpgradeStrategyAutomatic,
					Capabilities:    map[v1alpha1.OpAMPBridgeCapability]bool{v1alpha1.OpAMPBridgeCapabilityReportsStatus: true},
				},
			},
		},
		{
			name: "provided values in spec",
			opampBridge: v1alpha1.OpAMPBridge{
				Spec: v1alpha1.OpAMPBridgeSpec{
					Replicas:        &five,
					UpgradeStrategy: "adhoc",
				},
			},
			expected: v1alpha1.OpAMPBridge{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: v1alpha1.OpAMPBridgeSpec{
					Replicas:        &five,
					UpgradeStrategy: "adhoc",
					Capabilities:    map[v1alpha1.OpAMPBridgeCapability]bool{v1alpha1.OpAMPBridgeCapabilityReportsStatus: true},
				},
			},
		},
		{
			name: "enable ReportsStatus capability if not enabled already",
			opampBridge: v1alpha1.OpAMPBridge{
				Spec: v1alpha1.OpAMPBridgeSpec{
					Capabilities: map[v1alpha1.OpAMPBridgeCapability]bool{
						v1alpha1.OpAMPBridgeCapabilityReportsStatus: false,
					},
				},
			},
			expected: v1alpha1.OpAMPBridge{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: v1alpha1.OpAMPBridgeSpec{
					Replicas:        &one,
					UpgradeStrategy: v1alpha1.UpgradeStrategyAutomatic,
					Capabilities: map[v1alpha1.OpAMPBridgeCapability]bool{
						v1alpha1.OpAMPBridgeCapabilityReportsStatus: true,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := config.Config{
				CollectorImage:           "collector:v0.0.0",
				TargetAllocatorImage:     "ta:v0.0.0",
				OperatorOpAMPBridgeImage: "opampbridge:v0.0.0",
			}
			webhook := &OpAMPBridgeWebhook{
				logger: logr.Discard(),
				scheme: testScheme,
				cfg:    cfg,
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

	tests := []struct {
		name             string
		opampBridge      v1alpha1.OpAMPBridge
		expectedErr      string
		expectedWarnings []string
	}{
		{
			name: "specify all required fields, should not return error",
			opampBridge: v1alpha1.OpAMPBridge{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: v1alpha1.OpAMPBridgeSpec{
					Endpoint: "ws://opamp-server:4320/v1/opamp",
					Capabilities: map[v1alpha1.OpAMPBridgeCapability]bool{
						v1alpha1.OpAMPBridgeCapabilityReportsStatus:                  true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsRemoteConfig:            true,
						v1alpha1.OpAMPBridgeCapabilityReportsEffectiveConfig:         true,
						v1alpha1.OpAMPBridgeCapabilityReportsOwnTraces:               true,
						v1alpha1.OpAMPBridgeCapabilityReportsOwnMetrics:              true,
						v1alpha1.OpAMPBridgeCapabilityReportsOwnLogs:                 true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings: true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsOtherConnectionSettings: true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsRestartCommand:          true,
						v1alpha1.OpAMPBridgeCapabilityReportsHealth:                  true,
						v1alpha1.OpAMPBridgeCapabilityReportsRemoteConfig:            true,
					},
				},
			},
		},
		{
			name: "empty OpAMP Server endpoint",
			opampBridge: v1alpha1.OpAMPBridge{
				Spec: v1alpha1.OpAMPBridgeSpec{
					Endpoint: "",
				},
			},
			expectedErr: "the OpAMP server endpoint is not specified",
		},
		{
			name: "empty capabilities",
			opampBridge: v1alpha1.OpAMPBridge{
				Spec: v1alpha1.OpAMPBridgeSpec{
					Endpoint: "ws://opamp-server:4320/v1/opamp",
				},
			},
			expectedErr: "the capabilities supported by OpAMP Bridge are not specified",
		},
		{
			name: "replica count greater than 1 should return error",
			opampBridge: v1alpha1.OpAMPBridge{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: v1alpha1.OpAMPBridgeSpec{
					Replicas: &two,
					Endpoint: "ws://opamp-server:4320/v1/opamp",
					Capabilities: map[v1alpha1.OpAMPBridgeCapability]bool{
						v1alpha1.OpAMPBridgeCapabilityReportsStatus:                  true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsRemoteConfig:            true,
						v1alpha1.OpAMPBridgeCapabilityReportsEffectiveConfig:         true,
						v1alpha1.OpAMPBridgeCapabilityReportsOwnTraces:               true,
						v1alpha1.OpAMPBridgeCapabilityReportsOwnMetrics:              true,
						v1alpha1.OpAMPBridgeCapabilityReportsOwnLogs:                 true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings: true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsOtherConnectionSettings: true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsRestartCommand:          true,
						v1alpha1.OpAMPBridgeCapabilityReportsHealth:                  true,
						v1alpha1.OpAMPBridgeCapabilityReportsRemoteConfig:            true,
					},
				},
			},
			expectedErr: "replica count must not be greater than 1",
		},
		{
			name: "invalid port name",
			opampBridge: v1alpha1.OpAMPBridge{
				Spec: v1alpha1.OpAMPBridgeSpec{
					Endpoint: "ws://opamp-server:4320/v1/opamp",
					Capabilities: map[v1alpha1.OpAMPBridgeCapability]bool{
						v1alpha1.OpAMPBridgeCapabilityReportsStatus:                  true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsRemoteConfig:            true,
						v1alpha1.OpAMPBridgeCapabilityReportsEffectiveConfig:         true,
						v1alpha1.OpAMPBridgeCapabilityReportsOwnTraces:               true,
						v1alpha1.OpAMPBridgeCapabilityReportsOwnMetrics:              true,
						v1alpha1.OpAMPBridgeCapabilityReportsOwnLogs:                 true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings: true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsOtherConnectionSettings: true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsRestartCommand:          true,
						v1alpha1.OpAMPBridgeCapabilityReportsHealth:                  true,
						v1alpha1.OpAMPBridgeCapabilityReportsRemoteConfig:            true,
					},
					Ports: []v1.ServicePort{
						{
							// this port name contains a non-alphanumeric character, which is invalid.
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
			opampBridge: v1alpha1.OpAMPBridge{
				Spec: v1alpha1.OpAMPBridgeSpec{
					Endpoint: "ws://opamp-server:4320/v1/opamp",
					Capabilities: map[v1alpha1.OpAMPBridgeCapability]bool{
						v1alpha1.OpAMPBridgeCapabilityReportsStatus:                  true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsRemoteConfig:            true,
						v1alpha1.OpAMPBridgeCapabilityReportsEffectiveConfig:         true,
						v1alpha1.OpAMPBridgeCapabilityReportsOwnTraces:               true,
						v1alpha1.OpAMPBridgeCapabilityReportsOwnMetrics:              true,
						v1alpha1.OpAMPBridgeCapabilityReportsOwnLogs:                 true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings: true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsOtherConnectionSettings: true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsRestartCommand:          true,
						v1alpha1.OpAMPBridgeCapabilityReportsHealth:                  true,
						v1alpha1.OpAMPBridgeCapabilityReportsRemoteConfig:            true,
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
			opampBridge: v1alpha1.OpAMPBridge{
				Spec: v1alpha1.OpAMPBridgeSpec{
					Endpoint: "ws://opamp-server:4320/v1/opamp",
					Capabilities: map[v1alpha1.OpAMPBridgeCapability]bool{
						v1alpha1.OpAMPBridgeCapabilityReportsStatus:                  true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsRemoteConfig:            true,
						v1alpha1.OpAMPBridgeCapabilityReportsEffectiveConfig:         true,
						v1alpha1.OpAMPBridgeCapabilityReportsOwnTraces:               true,
						v1alpha1.OpAMPBridgeCapabilityReportsOwnMetrics:              true,
						v1alpha1.OpAMPBridgeCapabilityReportsOwnLogs:                 true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsOpAMPConnectionSettings: true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsOtherConnectionSettings: true,
						v1alpha1.OpAMPBridgeCapabilityAcceptsRestartCommand:          true,
						v1alpha1.OpAMPBridgeCapabilityReportsHealth:                  true,
						v1alpha1.OpAMPBridgeCapabilityReportsRemoteConfig:            true,
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
		t.Run(test.name, func(t *testing.T) {
			cfg := config.Config{
				CollectorImage:           "collector:v0.0.0",
				TargetAllocatorImage:     "ta:v0.0.0",
				OperatorOpAMPBridgeImage: "opampbridge:v0.0.0",
			}
			webhook := &OpAMPBridgeWebhook{
				logger: logr.Discard(),
				scheme: testScheme,
				cfg:    cfg,
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
