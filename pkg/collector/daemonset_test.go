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

package collector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/collector/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	. "github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

func TestDaemonSetNewDefault(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Tolerations: testTolerationValues,
			Config:      "test",
		},
	}
	cfg := config.New()

	// test
	d := DaemonSet(cfg, logger, otelcol)

	// verify
	assert.Equal(t, "my-instance-collector", d.Name)
	assert.Equal(t, "my-instance-collector", d.Labels["app.kubernetes.io/name"])
	assert.Equal(t, "true", d.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "8888", d.Annotations["prometheus.io/port"])
	assert.Equal(t, "/metrics", d.Annotations["prometheus.io/path"])
	assert.Equal(t, testTolerationValues, d.Spec.Template.Spec.Tolerations)

	assert.Len(t, d.Spec.Template.Spec.Containers, 1)

	// should have a pod annotation for config SHA
	assert.Len(t, d.Spec.Template.Annotations, 1)
	assert.Equal(t, "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", d.Spec.Template.Annotations["opentelemetry-operator-config/sha256"])

	// the pod selector should match the pod spec's labels
	assert.Equal(t, d.Spec.Selector.MatchLabels, d.Spec.Template.Labels)
}

func TestDaemonsetHostNetwork(t *testing.T) {
	// test
	d1 := DaemonSet(config.New(), logger, v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{},
	})
	assert.False(t, d1.Spec.Template.Spec.HostNetwork)

	// verify custom
	d2 := DaemonSet(config.New(), logger, v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			HostNetwork: true,
		},
	})
	assert.True(t, d2.Spec.Template.Spec.HostNetwork)
}

func TestDaemonsetPodAnnotations(t *testing.T) {
	// prepare
	testPodAnnotationValues := map[string]string{"annotation-key": "annotation-value"}
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			PodAnnotations: testPodAnnotationValues,
		},
	}
	cfg := config.New()

	// test
	ds := DaemonSet(cfg, logger, otelcol)
	expectedAnnotations := testPodAnnotationValues

	expectedAnnotations["opentelemetry-operator-config/sha256"] = ""

	// verify
	assert.Equal(t, "my-instance-collector", ds.Name)
	assert.Equal(t, expectedAnnotations, ds.Spec.Template.Annotations)
}
