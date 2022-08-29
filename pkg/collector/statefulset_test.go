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
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	. "github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

func TestStatefulSetNewDefault(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-namespace",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Mode:        "statefulset",
			Tolerations: testTolerationValues,
		},
	}
	cfg := config.New()

	// test
	ss := StatefulSet(cfg, logger, otelcol)

	// verify
	assert.Equal(t, "my-instance-collector", ss.Name)
	assert.Equal(t, "my-instance-collector", ss.Labels["app.kubernetes.io/name"])
	assert.Equal(t, "true", ss.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "8888", ss.Annotations["prometheus.io/port"])
	assert.Equal(t, "/metrics", ss.Annotations["prometheus.io/path"])
	assert.Equal(t, testTolerationValues, ss.Spec.Template.Spec.Tolerations)

	assert.Len(t, ss.Spec.Template.Spec.Containers, 1)

	// verify sha256 podAnnotation
	expectedAnnotations := map[string]string{
		"opentelemetry-operator-config/sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}
	assert.Equal(t, expectedAnnotations, ss.Spec.Template.Annotations)

	expectedLabels := map[string]string{
		"app.kubernetes.io/component":  "opentelemetry-collector",
		"app.kubernetes.io/instance":   "my-namespace.my-instance",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/name":       "my-instance-collector",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/version":    "latest",
	}
	assert.Equal(t, expectedLabels, ss.Spec.Template.Labels)

	expectedSelectorLabels := map[string]string{
		"app.kubernetes.io/component":  "opentelemetry-collector",
		"app.kubernetes.io/instance":   "my-namespace.my-instance",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/part-of":    "opentelemetry",
	}
	assert.Equal(t, expectedSelectorLabels, ss.Spec.Selector.MatchLabels)

	// the pod selector must be contained within pod spec's labels
	for k, v := range ss.Spec.Selector.MatchLabels {
		assert.Equal(t, v, ss.Spec.Template.Labels[k])
	}

	// assert correct service name
	assert.Equal(t, "my-instance-collector", ss.Spec.ServiceName)

	// assert correct pod management policy
	assert.Equal(t, appsv1.ParallelPodManagement, ss.Spec.PodManagementPolicy)
}

func TestStatefulSetReplicas(t *testing.T) {
	// prepare
	replicaInt := int32(3)
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Mode:     "statefulset",
			Replicas: &replicaInt,
		},
	}
	cfg := config.New()

	// test
	ss := StatefulSet(cfg, logger, otelcol)

	// assert correct number of replicas
	assert.Equal(t, int32(3), *ss.Spec.Replicas)
}

func TestStatefulSetVolumeClaimTemplates(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Mode: "statefulset",
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
				ObjectMeta: metav1.ObjectMeta{
					Name: "added-volume",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{"storage": resource.MustParse("1Gi")},
					},
				},
			}},
		},
	}
	cfg := config.New()

	// test
	ss := StatefulSet(cfg, logger, otelcol)

	// assert correct pvc name
	assert.Equal(t, "added-volume", ss.Spec.VolumeClaimTemplates[0].Name)

	// assert correct pvc access mode
	assert.Equal(t, corev1.PersistentVolumeAccessMode("ReadWriteOnce"), ss.Spec.VolumeClaimTemplates[0].Spec.AccessModes[0])

	// assert correct pvc storage
	assert.Equal(t, resource.MustParse("1Gi"), ss.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests["storage"])
}

func TestStatefulSetPodAnnotations(t *testing.T) {
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
	ss := StatefulSet(cfg, logger, otelcol)

	// Add sha256 podAnnotation
	testPodAnnotationValues["opentelemetry-operator-config/sha256"] = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	// verify
	assert.Equal(t, "my-instance-collector", ss.Name)
	assert.Equal(t, testPodAnnotationValues, ss.Spec.Template.Annotations)
}

func TestStatefulSetPodSecurityContext(t *testing.T) {
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

	d := StatefulSet(cfg, logger, otelcol)

	assert.Equal(t, &runAsNonRoot, d.Spec.Template.Spec.SecurityContext.RunAsNonRoot)
	assert.Equal(t, &runAsUser, d.Spec.Template.Spec.SecurityContext.RunAsUser)
	assert.Equal(t, &runasGroup, d.Spec.Template.Spec.SecurityContext.RunAsGroup)
}

func TestStatefulSetHostNetwork(t *testing.T) {
	// Test default
	otelcol_1 := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	d1 := StatefulSet(cfg, logger, otelcol_1)

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

	d2 := StatefulSet(cfg, logger, otelcol_2)
	assert.Equal(t, d2.Spec.Template.Spec.HostNetwork, true)
	assert.Equal(t, d2.Spec.Template.Spec.DNSPolicy, v1.DNSClusterFirstWithHostNet)
}

func TestStatefulSetFilterLabels(t *testing.T) {
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

	d := StatefulSet(cfg, logger, otelcol)

	assert.Len(t, d.ObjectMeta.Labels, 6)
	for k := range excludedLabels {
		assert.NotContains(t, d.ObjectMeta.Labels, k)
	}
}

func TestStatefulSetNodeSelector(t *testing.T) {
	// Test default
	otelcol_1 := v1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	d1 := StatefulSet(cfg, logger, otelcol_1)

	assert.Empty(t, d1.Spec.Template.Spec.NodeSelector)

	// Test nodeSelector
	otelcol_2 := v1alpha1.OpenTelemetryCollector{
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

	d2 := StatefulSet(cfg, logger, otelcol_2)
	assert.Equal(t, d2.Spec.Template.Spec.NodeSelector, map[string]string{"node-key": "node-value"})
}
