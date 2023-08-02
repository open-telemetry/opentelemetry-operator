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

package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestDefaultAnnotations(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-ns",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Config: "test",
		},
	}

	// test
	annotations := Annotations(otelcol)
	podAnnotations := PodAnnotations(otelcol)

	//verify
	assert.Equal(t, "true", annotations["prometheus.io/scrape"])
	assert.Equal(t, "8888", annotations["prometheus.io/port"])
	assert.Equal(t, "/metrics", annotations["prometheus.io/path"])
	assert.Equal(t, "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", annotations["opentelemetry-operator-config/sha256"])
	//verify propagation from metadata.annotations to spec.template.spec.metadata.annotations
	assert.Equal(t, "true", podAnnotations["prometheus.io/scrape"])
	assert.Equal(t, "8888", podAnnotations["prometheus.io/port"])
	assert.Equal(t, "/metrics", podAnnotations["prometheus.io/path"])
	assert.Equal(t, "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", podAnnotations["opentelemetry-operator-config/sha256"])
}

func TestUserAnnotations(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-ns",
			Annotations: map[string]string{"prometheus.io/scrape": "false",
				"prometheus.io/port":                   "1234",
				"prometheus.io/path":                   "/test",
				"opentelemetry-operator-config/sha256": "shouldBeOverwritten",
			},
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Config: "test",
		},
	}

	// test
	annotations := Annotations(otelcol)
	podAnnotations := PodAnnotations(otelcol)

	//verify
	assert.Equal(t, "false", annotations["prometheus.io/scrape"])
	assert.Equal(t, "1234", annotations["prometheus.io/port"])
	assert.Equal(t, "/test", annotations["prometheus.io/path"])
	assert.Equal(t, "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", annotations["opentelemetry-operator-config/sha256"])
	assert.Equal(t, "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", podAnnotations["opentelemetry-operator-config/sha256"])
}

func TestAnnotationsPropagateDown(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"myapp": "mycomponent"},
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			PodAnnotations: map[string]string{"pod_annotation": "pod_annotation_value"},
		},
	}

	// test
	annotations := Annotations(otelcol)
	podAnnotations := PodAnnotations(otelcol)

	// verify
	assert.Len(t, annotations, 5)
	assert.Equal(t, "mycomponent", annotations["myapp"])
	assert.Equal(t, "mycomponent", podAnnotations["myapp"])
	assert.Equal(t, "pod_annotation_value", podAnnotations["pod_annotation"])
}
