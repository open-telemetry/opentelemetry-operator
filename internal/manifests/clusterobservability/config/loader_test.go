// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestLoadAgentCollectorConfigOpenShiftTLSOverride(t *testing.T) {
	loader := NewConfigLoader()

	base, err := loader.LoadCollectorConfig(AgentCollectorType, DistroProvider(""), v1alpha1.ClusterObservabilitySpec{})
	require.NoError(t, err)
	openshift, err := loader.LoadCollectorConfig(AgentCollectorType, OpenShift, v1alpha1.ClusterObservabilitySpec{})
	require.NoError(t, err)

	assert.Equal(t, true, requireMap(t, base.Receivers.Object, "kubelet_stats")["insecure_skip_verify"])

	kubeletstats := requireMap(t, openshift.Receivers.Object, "kubelet_stats")
	assert.Equal(t, "/etc/kubelet-serving-ca/ca-bundle.crt", kubeletstats["ca_file"])
	assert.Equal(t, false, kubeletstats["insecure_skip_verify"])
	assert.Contains(t, openshift.Receivers.Object, "host_metrics")
	assert.Contains(t, openshift.Receivers.Object, "file_log")
}

func requireMap(t *testing.T, parent map[string]any, key string) map[string]any {
	t.Helper()

	value, ok := parent[key]
	require.Truef(t, ok, "key %q not found", key)
	result, ok := value.(map[string]any)
	require.Truef(t, ok, "key %q has type %T", key, value)
	return result
}
