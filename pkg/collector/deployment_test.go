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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	. "github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

var testTolerationValues = []v1.Toleration{
	{
		Key:    "hii",
		Value:  "greeting",
		Effect: "NoSchedule",
	},
}

func TestDeploymentNewDefault(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Tolerations: testTolerationValues,
		},
	}
	cfg := config.New()

	// test
	d := Deployment(cfg, logger, otelcol)

	// verify
	assert.Equal(t, "my-instance-collector", d.Name)
	assert.Equal(t, "my-instance-collector", d.Labels["app.kubernetes.io/name"])
	assert.Equal(t, "true", d.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "8888", d.Annotations["prometheus.io/port"])
	assert.Equal(t, "/metrics", d.Annotations["prometheus.io/path"])
	assert.Equal(t, testTolerationValues, d.Spec.Template.Spec.Tolerations)

	assert.Len(t, d.Spec.Template.Spec.Containers, 1)

	// verify sha256 podAnnotation
	expectedAnnotations := map[string]string{
		"opentelemetry-operator-config/sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}
	assert.Equal(t, expectedAnnotations, d.Spec.Template.Annotations)

	// the pod selector should match the pod spec's labels
	assert.Equal(t, d.Spec.Template.Labels, d.Spec.Selector.MatchLabels)
}

func TestDeploymentPodAnnotations(t *testing.T) {
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
	d := Deployment(cfg, logger, otelcol)

	//Add sha256 podAnnotation
	testPodAnnotationValues["opentelemetry-operator-config/sha256"] = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	// verify
	assert.Equal(t, "my-instance-collector", d.Name)
	assert.Equal(t, testPodAnnotationValues, d.Spec.Template.Annotations)
}

func TestDeploymenttPodSecurityContext(t *testing.T) {
	runAsNonRoot := true
	runAsUser := int64(1337)
	runasGroup := int64(1338)

	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			PodSecurityContext: &v1.PodSecurityContext{
				RunAsNonRoot: &runAsNonRoot,
				RunAsUser:    &runAsUser,
				RunAsGroup:   &runasGroup,
			},
		},
	}

	cfg := config.New()

	d := Deployment(cfg, logger, otelcol)

	assert.Equal(t, &runAsNonRoot, d.Spec.Template.Spec.SecurityContext.RunAsNonRoot)
	assert.Equal(t, &runAsUser, d.Spec.Template.Spec.SecurityContext.RunAsUser)
	assert.Equal(t, &runasGroup, d.Spec.Template.Spec.SecurityContext.RunAsGroup)
}

func TestDeploymentHostNetwork(t *testing.T) {
	// Test default
	otelcol_1 := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	d1 := Deployment(cfg, logger, otelcol_1)

	assert.Equal(t, d1.Spec.Template.Spec.HostNetwork, false)
	assert.Equal(t, d1.Spec.Template.Spec.DNSPolicy, v1.DNSClusterFirst)

	// Test hostNetwork=true
	otelcol_2 := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-hostnetwork",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			HostNetwork: true,
		},
	}

	cfg = config.New()

	d2 := Deployment(cfg, logger, otelcol_2)
	assert.Equal(t, d2.Spec.Template.Spec.HostNetwork, true)
	assert.Equal(t, d2.Spec.Template.Spec.DNSPolicy, v1.DNSClusterFirstWithHostNet)
}

func TestDeploymentFilterLabels(t *testing.T) {
	excludedLabels := map[string]string{
		"foo":         "1",
		"app.foo.bar": "1",
	}

	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "my-instance",
			Labels: excludedLabels,
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{},
	}

	cfg := config.New(config.WithLabelFilters([]string{"foo*", "app.*.bar"}))

	d := Deployment(cfg, logger, otelcol)

	assert.Len(t, d.ObjectMeta.Labels, 5)
	for k := range excludedLabels {
		assert.NotContains(t, d.ObjectMeta.Labels, k)
	}
}
