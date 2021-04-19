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

package reconcile

import (
	"context"
	"fmt"
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("unit-tests")

func TestDesiredConfigMap(t *testing.T) {
	t.Run("should return expected config map", func(t *testing.T) {
		expected := configMap("test-collector")
		actual := desiredConfigMap(context.Background(), params())
		assert.Equal(t, expected, actual)
	})

}

func TestExpectedConfigMap(t *testing.T) {
	t.Run("should create config map", func(t *testing.T) {
		err := expectedConfigMaps(context.Background(), params(), []v1.ConfigMap{configMap("test-collector")}, true)
		assert.NoError(t, err)

		actual, err := getCM("test-collector")

		assert.NoError(t, err)
		assert.NotNil(t, actual)
	})

	t.Run("should update config map", func(t *testing.T) {
		createCMIfNotExists(t, "test-collector")

		_ = expectedConfigMaps(context.Background(), params(), []v1.ConfigMap{configMap("test-collector")}, true)
		//assert.NoError(t, err)
		//
		//actual, err := getCM(t)
		//
		//assert.NoError(t, err)
		//assert.Equal(t, actual.Data, configMap().Data)
	})

	t.Run("should delete config map", func(t *testing.T) {
		createCMIfNotExists(t, "test")

		err := deleteConfigMaps(context.Background(), params(), []v1.ConfigMap{configMap("test-collector")})
		assert.NoError(t, err)

		_, err = getCM("test")

		assert.True(t, errors.IsNotFound(err))
	})
}

func configMap(name string) v1.ConfigMap {
	return v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
				"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params().Instance.Namespace, params().Instance.Name),
				"app.kubernetes.io/part-of":    "opentelemetry",
				"app.kubernetes.io/component":  "opentelemetry-collector",
				"app.kubernetes.io/name":       name,
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

func createCMIfNotExists(tb testing.TB, name string) {
	tb.Helper()
	actual := v1.ConfigMap{}
	err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: name}, &actual)
	if errors.IsNotFound(err) {
		cm := configMap(name)
		err := k8sClient.Create(context.Background(),
			&cm)
		assert.NoError(tb, err)
	}
}

func getCM(name string) (cm v1.ConfigMap, err error) {
	err = k8sClient.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: name}, &cm)
	return
}
