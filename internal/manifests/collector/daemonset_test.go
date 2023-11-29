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
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	. "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
)

func TestDaemonSetNewDefault(t *testing.T) {
	// prepare
	params := manifests.Params{
		Config: config.New(),
		OtelCol: v1alpha1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-instance",
				Namespace: "my-namespace",
			},
			Spec: v1alpha1.OpenTelemetryCollectorSpec{
				Tolerations: testTolerationValues,
			},
		},
		Log: logger,
	}

	// test
	d := DaemonSet(params)

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
		"prometheus.io/path":                   "/metrics",
		"prometheus.io/port":                   "8888",
		"prometheus.io/scrape":                 "true",
	}
	assert.Equal(t, expectedAnnotations, d.Spec.Template.Annotations)

	expectedLabels := map[string]string{
		"app.kubernetes.io/component":  "opentelemetry-collector",
		"app.kubernetes.io/instance":   "my-namespace.my-instance",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/name":       "my-instance-collector",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/version":    "latest",
	}
	assert.Equal(t, expectedLabels, d.Spec.Template.Labels)

	expectedSelectorLabels := map[string]string{
		"app.kubernetes.io/component":  "opentelemetry-collector",
		"app.kubernetes.io/instance":   "my-namespace.my-instance",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/part-of":    "opentelemetry",
	}
	assert.Equal(t, expectedSelectorLabels, d.Spec.Selector.MatchLabels)

	// the pod selector must be contained within pod spec's labels
	for k, v := range d.Spec.Selector.MatchLabels {
		assert.Equal(t, v, d.Spec.Template.Labels[k])
	}
}

func TestDaemonsetHostNetwork(t *testing.T) {
	params1 := manifests.Params{
		Config: config.New(),
		OtelCol: v1alpha1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-instance",
				Namespace: "my-namespace",
			},
			Spec: v1alpha1.OpenTelemetryCollectorSpec{},
		},
		Log: logger,
	}
	// test
	d1 := DaemonSet(params1)
	assert.False(t, d1.Spec.Template.Spec.HostNetwork)
	assert.Equal(t, d1.Spec.Template.Spec.DNSPolicy, v1.DNSClusterFirst)

	// verify custom
	params2 := manifests.Params{
		Config: config.New(),
		OtelCol: v1alpha1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-instance",
				Namespace: "my-namespace",
			},
			Spec: v1alpha1.OpenTelemetryCollectorSpec{
				HostNetwork: true,
			},
		},
		Log: logger,
	}
	d2 := DaemonSet(params2)
	assert.True(t, d2.Spec.Template.Spec.HostNetwork)
	assert.Equal(t, d2.Spec.Template.Spec.DNSPolicy, v1.DNSClusterFirstWithHostNet)
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

	params := manifests.Params{
		Config:  cfg,
		OtelCol: otelcol,
		Log:     logger,
	}

	// test
	ds := DaemonSet(params)

	// Add sha256 podAnnotation
	testPodAnnotationValues["opentelemetry-operator-config/sha256"] = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	expectedAnnotations := map[string]string{
		"annotation-key":                       "annotation-value",
		"opentelemetry-operator-config/sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		"prometheus.io/path":                   "/metrics",
		"prometheus.io/port":                   "8888",
		"prometheus.io/scrape":                 "true",
	}

	// verify
	assert.Equal(t, "my-instance-collector", ds.Name)
	assert.Len(t, ds.Spec.Template.Annotations, 5)
	assert.Equal(t, expectedAnnotations, ds.Spec.Template.Annotations)
}

func TestDaemonstPodSecurityContext(t *testing.T) {
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

	params := manifests.Params{
		Config:  cfg,
		OtelCol: otelcol,
		Log:     logger,
	}

	d := DaemonSet(params)

	assert.Equal(t, &runAsNonRoot, d.Spec.Template.Spec.SecurityContext.RunAsNonRoot)
	assert.Equal(t, &runAsUser, d.Spec.Template.Spec.SecurityContext.RunAsUser)
	assert.Equal(t, &runasGroup, d.Spec.Template.Spec.SecurityContext.RunAsGroup)
}

func TestDaemonsetFilterLabels(t *testing.T) {
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

	params := manifests.Params{
		Config:  cfg,
		OtelCol: otelcol,
		Log:     logger,
	}

	d := DaemonSet(params)

	assert.Len(t, d.ObjectMeta.Labels, 6)
	for k := range excludedLabels {
		assert.NotContains(t, d.ObjectMeta.Labels, k)
	}
}

func TestDaemonSetNodeSelector(t *testing.T) {
	// Test default
	otelcol1 := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := manifests.Params{
		Config:  cfg,
		OtelCol: otelcol1,
		Log:     logger,
	}

	d1 := DaemonSet(params1)

	assert.Empty(t, d1.Spec.Template.Spec.NodeSelector)

	// Test nodeSelector
	otelcol2 := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-nodeselector",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			HostNetwork: true,
			NodeSelector: map[string]string{
				"node-key": "node-value",
			},
		},
	}

	cfg = config.New()

	params2 := manifests.Params{
		Config:  cfg,
		OtelCol: otelcol2,
		Log:     logger,
	}

	d2 := DaemonSet(params2)
	assert.Equal(t, d2.Spec.Template.Spec.NodeSelector, map[string]string{"node-key": "node-value"})
}

func TestDaemonSetPriorityClassName(t *testing.T) {
	otelcol1 := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := manifests.Params{
		Config:  cfg,
		OtelCol: otelcol1,
		Log:     logger,
	}

	d1 := DaemonSet(params1)
	assert.Empty(t, d1.Spec.Template.Spec.PriorityClassName)

	priorityClassName := "test-class"

	otelcol2 := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-priortyClassName",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			PriorityClassName: priorityClassName,
		},
	}

	cfg = config.New()

	params2 := manifests.Params{
		Config:  cfg,
		OtelCol: otelcol2,
		Log:     logger,
	}

	d2 := DaemonSet(params2)
	assert.Equal(t, priorityClassName, d2.Spec.Template.Spec.PriorityClassName)
}

func TestDaemonSetAffinity(t *testing.T) {
	otelcol1 := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := manifests.Params{
		Config:  cfg,
		OtelCol: otelcol1,
		Log:     logger,
	}

	d1 := DaemonSet(params1)
	assert.Nil(t, d1.Spec.Template.Spec.Affinity)

	otelcol2 := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-priortyClassName",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Affinity: testAffinityValue,
		},
	}

	cfg = config.New()

	params2 := manifests.Params{
		Config:  cfg,
		OtelCol: otelcol2,
		Log:     logger,
	}

	d2 := DaemonSet(params2)
	assert.NotNil(t, d2.Spec.Template.Spec.Affinity)
	assert.Equal(t, *testAffinityValue, *d2.Spec.Template.Spec.Affinity)
}

func TestDaemonSetInitContainer(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-namespace",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			InitContainers: []v1.Container{
				{
					Name: "test",
				},
			},
		},
	}
	cfg := config.New()

	params := manifests.Params{
		Config:  cfg,
		OtelCol: otelcol,
		Log:     logger,
	}

	// test
	d := DaemonSet(params)
	assert.Equal(t, "my-instance-collector", d.Name)
	assert.Equal(t, "my-instance-collector", d.Labels["app.kubernetes.io/name"])
	assert.Equal(t, "true", d.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "8888", d.Annotations["prometheus.io/port"])
	assert.Equal(t, "/metrics", d.Annotations["prometheus.io/path"])
	assert.Len(t, d.Spec.Template.Spec.InitContainers, 1)
}

func TestDaemonSetAdditionalContainer(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-namespace",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			AdditionalContainers: []v1.Container{
				{
					Name: "test",
				},
			},
		},
	}
	cfg := config.New()

	params := manifests.Params{
		Config:  cfg,
		OtelCol: otelcol,
		Log:     logger,
	}

	// test
	d := DaemonSet(params)
	assert.Equal(t, "my-instance-collector", d.Name)
	assert.Equal(t, "my-instance-collector", d.Labels["app.kubernetes.io/name"])
	assert.Equal(t, "true", d.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "8888", d.Annotations["prometheus.io/port"])
	assert.Equal(t, "/metrics", d.Annotations["prometheus.io/path"])
	assert.Len(t, d.Spec.Template.Spec.Containers, 2)
	assert.Equal(t, v1.Container{Name: "test"}, d.Spec.Template.Spec.Containers[0])
}

func TestDaemonSetDefaultUpdateStrategy(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-namespace",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: "RollingUpdate",
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: int32(1)},
					MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: int32(1)},
				},
			},
		},
	}
	cfg := config.New()

	params := manifests.Params{
		Config:  cfg,
		OtelCol: otelcol,
		Log:     logger,
	}

	// test
	d := DaemonSet(params)
	assert.Equal(t, "my-instance-collector", d.Name)
	assert.Equal(t, "my-instance-collector", d.Labels["app.kubernetes.io/name"])
	assert.Equal(t, appsv1.DaemonSetUpdateStrategyType("RollingUpdate"), d.Spec.UpdateStrategy.Type)
	assert.Equal(t, &intstr.IntOrString{Type: intstr.Int, IntVal: int32(1)}, d.Spec.UpdateStrategy.RollingUpdate.MaxSurge)
	assert.Equal(t, &intstr.IntOrString{Type: intstr.Int, IntVal: int32(1)}, d.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable)
}

func TestDaemonSetOnDeleteUpdateStrategy(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-namespace",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: "OnDelete",
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: int32(1)},
					MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: int32(1)},
				},
			},
		},
	}
	cfg := config.New()

	params := manifests.Params{
		Config:  cfg,
		OtelCol: otelcol,
		Log:     logger,
	}

	// test
	d := DaemonSet(params)
	assert.Equal(t, "my-instance-collector", d.Name)
	assert.Equal(t, "my-instance-collector", d.Labels["app.kubernetes.io/name"])
	assert.Equal(t, appsv1.DaemonSetUpdateStrategyType("OnDelete"), d.Spec.UpdateStrategy.Type)
	assert.Equal(t, &intstr.IntOrString{Type: intstr.Int, IntVal: int32(1)}, d.Spec.UpdateStrategy.RollingUpdate.MaxSurge)
	assert.Equal(t, &intstr.IntOrString{Type: intstr.Int, IntVal: int32(1)}, d.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable)
}
