package reconcile

import (
	"context"
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("unit-tests")

func TestDesiredConfigMap(t *testing.T) {
	t.Run("should return expected config map", func(t *testing.T) {
		expected := configMap()
		actual := desiredConfigMap(context.Background(), params())
		assert.Equal(t, expected, actual)
	})

}

func TestExpectedConfigMap(t *testing.T) {
	t.Run("should create config map", func(t *testing.T) {
		err := expectedConfigMaps(context.Background(), params(), []v1.ConfigMap{configMap()}, true)
		assert.NoError(t, err)

		actual := v1.ConfigMap{}
		err = k8sClient.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "test-collector"}, &actual)

		assert.NoError(t, err)
		assert.NotNil(t, actual)
	})

	t.Run("should update config map", func(t *testing.T) {
		err := k8sClient.Create(context.Background(),
			&v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				}})
		assert.NoError(t, err)

		//err = expectedConfigMaps(context.Background(), params(), []v1.ConfigMap{configMap()}, true)
		//assert.NoError(t, err)
		//
		//actual := v1.ConfigMap{}
		//err = k8sClient.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: "test-collector"}, &actual)
		//assert.NoError(t, err)
		//assert.Equal(t, actual.Data, configMap().Data)
	})
}

func configMap() v1.ConfigMap {
	return v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-collector",
			Namespace: "default",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
				"app.kubernetes.io/instance":   "default.test",
				"app.kubernetes.io/part-of":    "opentelemetry",
				"app.kubernetes.io/component":  "opentelemetry-collector",
				"app.kubernetes.io/name":       "test-collector",
			},
		},
		Data: map[string]string{
			"collector.yaml": params().Instance.Spec.Config,
		},
	}
}

func params() Params {
	return Params{
		Config: config.New(),
		Client: k8sClient,
		Instance: v1alpha1.OpenTelemetryCollector{
			TypeMeta: metav1.TypeMeta{
				Kind:       "opentelemetry.io",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
				UID:       "testuid1234",
			},
			Spec: v1alpha1.OpenTelemetryCollectorSpec{
				Config: `
    receivers:
      jaeger:
        protocols:
          grpc:
    processors:

    exporters:
      logging:

    service:
      pipelines:
        traces:
          receivers: [jaeger]
          processors: []
          exporters: [logging]

`,
			},
		},
		Scheme:   testScheme,
		Log:      logger,
		Recorder: record.NewFakeRecorder(10),
	}
}
