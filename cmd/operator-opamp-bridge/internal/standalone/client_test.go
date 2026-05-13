// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/config"
)

const validCollectorConfig = `receivers:
  otlp:
    protocols:
      grpc:
exporters:
  debug:
service:
  pipelines:
    traces:
      receivers:
        - otlp
      exporters:
        - debug
`

func getFakeK8sClient(t *testing.T, objs ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	require.NoError(t, v1.AddToScheme(scheme))
	builder := fake.NewClientBuilder().WithScheme(scheme)
	for _, obj := range objs {
		builder = builder.WithObjects(obj)
	}
	return builder.Build()
}

func testConfigMap() *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "collector-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"collector.yaml": validCollectorConfig,
			"extra.yaml":     "extra: true",
		},
	}
}

func testAgentConfig() config.StandaloneAgentConfig {
	return config.StandaloneAgentConfig{
		Name:      "standalone-collector",
		Namespace: "default",
		Type:      "otel-collector",
		Config: map[string]config.StandaloneConfigEntry{
			"collector": {
				Kind:      "configmap",
				Namespace: "default",
				Name:      "collector-config",
				Key:       "collector.yaml",
			},
		},
	}
}

func newTestClient(k8s client.Client) *Client {
	return NewClient("test-bridge", logr.Discard(), k8s, nil, nil)
}

func TestScopedApplierListInstancesReturnsConfiguredConfigMapKey(t *testing.T) {
	cm := testConfigMap()
	c := newTestClient(getFakeK8sClient(t, cm))

	instances, err := c.scopedApplier(testAgentConfig()).ListInstances()
	require.NoError(t, err)
	require.Len(t, instances, 1)

	assert.Equal(t, "collector", instances[0].GetName())
	assert.Equal(t, "collector", instances[0].GetConfigMapKey().String())
	assert.Equal(t, validCollectorConfig, string(instances[0].GetEffectiveConfig()))
}

func TestScopedApplierApplyUpdatesConfiguredConfigMapKey(t *testing.T) {
	cm := testConfigMap()
	k8s := getFakeK8sClient(t, cm)
	c := newTestClient(k8s)

	updatedConfig := `receivers:
  otlp:
    protocols:
      http:
exporters:
  debug:
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [debug]
`
	err := c.scopedApplier(testAgentConfig()).Apply("collector", "", &protobufs.AgentConfigFile{Body: []byte(updatedConfig)})
	require.NoError(t, err)

	updated := &v1.ConfigMap{}
	require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Name: "collector-config", Namespace: "default"}, updated))
	assert.Equal(t, updatedConfig, updated.Data["collector.yaml"])
	assert.Equal(t, "extra: true", updated.Data["extra.yaml"])
}

func TestScopedApplierApplyRejectsUnknownRemoteName(t *testing.T) {
	c := newTestClient(getFakeK8sClient(t, testConfigMap()))

	err := c.scopedApplier(testAgentConfig()).Apply("unknown", "", &protobufs.AgentConfigFile{Body: []byte(validCollectorConfig)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not manage config")
}

func TestScopedApplierApplyDoesNotCreateConfigMap(t *testing.T) {
	c := newTestClient(getFakeK8sClient(t))

	err := c.scopedApplier(testAgentConfig()).Apply("collector", "", &protobufs.AgentConfigFile{Body: []byte(validCollectorConfig)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support creating ConfigMap")
}

func TestScopedApplierDeleteUnsupported(t *testing.T) {
	c := newTestClient(getFakeK8sClient(t, testConfigMap()))

	err := c.scopedApplier(testAgentConfig()).Delete("collector", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support deleting")
}
