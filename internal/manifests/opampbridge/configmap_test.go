// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opampbridge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
)

func expectedLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/instance":   "my-namespace.my-instance",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/version":    "0.69.0",
		"app.kubernetes.io/component":  "opentelemetry-opamp-bridge",
		"app.kubernetes.io/name":       "my-instance-opamp-bridge",
	}
}

func TestDesiredConfigMap(t *testing.T) {
	data := map[string]string{
		"remoteconfiguration.yaml": `capabilities:
  AcceptsOpAMPConnectionSettings: true
  AcceptsOtherConnectionSettings: true
  AcceptsRemoteConfig: true
  AcceptsRestartCommand: true
  ReportsEffectiveConfig: true
  ReportsHealth: true
  ReportsOwnLogs: true
  ReportsOwnMetrics: true
  ReportsOwnTraces: true
  ReportsRemoteConfig: true
  ReportsStatus: true
componentsAllowed:
  exporters:
  - debug
  processors:
  - memory_limiter
  receivers:
  - otlp
endpoint: ws://opamp-server:4320/v1/opamp
headers:
  authorization: access-12345-token
`}
	tests := []struct {
		description    string
		image          string
		expectedLabels func() map[string]string
		expectedData   map[string]string
	}{
		{
			description:    "should return expected opamp-bridge config map",
			image:          "ghcr.io/open-telemetry/opentelemetry-operator/operator-opamp-bridge:0.69.0",
			expectedLabels: expectedLabels,
			expectedData:   data,
		},
		{
			description: "should return expected opamp-bridge config map, sha256 image",
			image:       "ghcr.io/open-telemetry/opentelemetry-operator/operator-opamp-bridge:main@sha256:00738c3a6bca8f143995c9c89fd0c1976784d9785ea394fcdfe580fb18754e1e",
			expectedLabels: func() map[string]string {
				ls := expectedLabels()
				ls["app.kubernetes.io/version"] = "main"
				return ls
			},
			expectedData: data,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			opampBridge := v1alpha1.OpAMPBridge{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-instance",
					Namespace: "my-namespace",
				},
				Spec: v1alpha1.OpAMPBridgeSpec{
					Image:    tc.image,
					Endpoint: "ws://opamp-server:4320/v1/opamp",
					Headers:  map[string]string{"authorization": "access-12345-token"},
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
					ComponentsAllowed: map[string][]string{"receivers": {"otlp"}, "processors": {"memory_limiter"}, "exporters": {"debug"}},
				},
			}

			cfg := config.New()

			params := manifests.Params{
				Config:      cfg,
				OpAMPBridge: opampBridge,
				Log:         logger,
			}

			actual, err := ConfigMap(params)
			assert.NoError(t, err)

			assert.Equal(t, "my-instance-opamp-bridge", actual.Name)
			assert.Equal(t, tc.expectedLabels(), actual.Labels)
			assert.Equal(t, tc.expectedData, actual.Data)
		})
	}
}
