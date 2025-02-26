// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	. "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
)

func TestStatefulSetNewDefault(t *testing.T) {
	// prepare
	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-namespace",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode: "statefulset",
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Tolerations: testTolerationValues,
			},
		},
	}
	cfg := config.New()

	params := manifests.Params{
		OtelCol: otelcol,
		Config:  cfg,
		Log:     logger,
	}

	// test
	ss, err := StatefulSet(params)
	require.NoError(t, err)

	// verify
	assert.Equal(t, "my-instance-collector", ss.Name)
	assert.Equal(t, "my-instance-collector", ss.Labels["app.kubernetes.io/name"])
	assert.Equal(t, testTolerationValues, ss.Spec.Template.Spec.Tolerations)

	assert.Len(t, ss.Spec.Template.Spec.Containers, 1)

	// verify sha256 podAnnotation
	expectedAnnotations := map[string]string{
		"opentelemetry-operator-config/sha256": "fbcdae6a02b2115cd5ca4f34298202ab041d1dfe62edebfaadb48b1ee178231d",
		"prometheus.io/path":                   "/metrics",
		"prometheus.io/port":                   "8888",
		"prometheus.io/scrape":                 "true",
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
	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode: "statefulset",
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Replicas: &replicaInt,
			},
		},
	}
	cfg := config.New()

	params := manifests.Params{
		OtelCol: otelcol,
		Config:  cfg,
		Log:     logger,
	}

	// test
	ss, err := StatefulSet(params)
	require.NoError(t, err)

	// assert correct number of replicas
	assert.Equal(t, int32(3), *ss.Spec.Replicas)
}

func TestStatefulSetVolumeClaimTemplates(t *testing.T) {
	// prepare
	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode: "statefulset",
			StatefulSetCommonFields: v1beta1.StatefulSetCommonFields{
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
					ObjectMeta: metav1.ObjectMeta{
						Name: "added-volume",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{"storage": resource.MustParse("1Gi")},
						},
					},
				}},
			},
		},
	}
	cfg := config.New()

	params := manifests.Params{
		OtelCol: otelcol,
		Config:  cfg,
		Log:     logger,
	}

	// test
	ss, err := StatefulSet(params)
	require.NoError(t, err)

	// assert correct pvc name
	assert.Equal(t, "added-volume", ss.Spec.VolumeClaimTemplates[0].Name)

	// assert correct pvc access mode
	assert.Equal(t, corev1.PersistentVolumeAccessMode("ReadWriteOnce"), ss.Spec.VolumeClaimTemplates[0].Spec.AccessModes[0])

	// assert correct pvc storage
	assert.Equal(t, resource.MustParse("1Gi"), ss.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests["storage"])
}

func TestStatefulSetPeristentVolumeRetentionPolicy(t *testing.T) {
	// prepare
	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode: "statefulset",
			StatefulSetCommonFields: v1beta1.StatefulSetCommonFields{
				PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
					WhenDeleted: appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
					WhenScaled:  appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
				},
			},
		},
	}
	cfg := config.New()

	params := manifests.Params{
		OtelCol: otelcol,
		Config:  cfg,
		Log:     logger,
	}

	// test
	ss, err := StatefulSet(params)
	require.NoError(t, err)

	// assert PersistentVolumeClaimRetentionPolicy added
	assert.NotNil(t, ss.Spec.PersistentVolumeClaimRetentionPolicy)

	// assert correct WhenDeleted value
	assert.Equal(t, ss.Spec.PersistentVolumeClaimRetentionPolicy.WhenDeleted, appsv1.RetainPersistentVolumeClaimRetentionPolicyType)

	// assert correct WhenScaled value
	assert.Equal(t, ss.Spec.PersistentVolumeClaimRetentionPolicy.WhenScaled, appsv1.DeletePersistentVolumeClaimRetentionPolicyType)

}

func TestStatefulSetPodAnnotations(t *testing.T) {
	// prepare
	testPodAnnotationValues := map[string]string{"annotation-key": "annotation-value"}
	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				PodAnnotations: testPodAnnotationValues,
			},
		},
	}
	cfg := config.New()

	params := manifests.Params{
		OtelCol: otelcol,
		Config:  cfg,
		Log:     logger,
	}

	// test
	ss, err := StatefulSet(params)
	require.NoError(t, err)

	// Add sha256 podAnnotation
	testPodAnnotationValues["opentelemetry-operator-config/sha256"] = "fbcdae6a02b2115cd5ca4f34298202ab041d1dfe62edebfaadb48b1ee178231d"

	expectedAnnotations := map[string]string{
		"annotation-key":                       "annotation-value",
		"opentelemetry-operator-config/sha256": "fbcdae6a02b2115cd5ca4f34298202ab041d1dfe62edebfaadb48b1ee178231d",
		"prometheus.io/path":                   "/metrics",
		"prometheus.io/port":                   "8888",
		"prometheus.io/scrape":                 "true",
	}
	// verify
	assert.Equal(t, "my-instance-collector", ss.Name)
	assert.Equal(t, expectedAnnotations, ss.Spec.Template.Annotations)
}

func TestStatefulSetPodSecurityContext(t *testing.T) {
	runAsNonRoot := true
	runAsUser := int64(1337)
	runasGroup := int64(1338)

	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				PodSecurityContext: &v1.PodSecurityContext{
					RunAsNonRoot: &runAsNonRoot,
					RunAsUser:    &runAsUser,
					RunAsGroup:   &runasGroup,
				},
			},
		},
	}

	cfg := config.New()

	params := manifests.Params{
		OtelCol: otelcol,
		Config:  cfg,
		Log:     logger,
	}

	d, err := StatefulSet(params)
	require.NoError(t, err)

	assert.Equal(t, &runAsNonRoot, d.Spec.Template.Spec.SecurityContext.RunAsNonRoot)
	assert.Equal(t, &runAsUser, d.Spec.Template.Spec.SecurityContext.RunAsUser)
	assert.Equal(t, &runasGroup, d.Spec.Template.Spec.SecurityContext.RunAsGroup)
}

func TestStatefulSetHostNetwork(t *testing.T) {
	// Test default
	otelcol1 := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := manifests.Params{
		OtelCol: otelcol1,
		Config:  cfg,
		Log:     logger,
	}

	d1, err := StatefulSet(params1)
	require.NoError(t, err)

	assert.Equal(t, d1.Spec.Template.Spec.HostNetwork, false)
	assert.Equal(t, d1.Spec.Template.Spec.DNSPolicy, v1.DNSClusterFirst)

	// Test hostNetwork=true
	otelcol2 := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-hostnetwork",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				HostNetwork: true,
			},
		},
	}

	cfg = config.New()

	params2 := manifests.Params{
		OtelCol: otelcol2,
		Config:  cfg,
		Log:     logger,
	}

	d2, err := StatefulSet(params2)
	require.NoError(t, err)
	assert.Equal(t, d2.Spec.Template.Spec.HostNetwork, true)
	assert.Equal(t, d2.Spec.Template.Spec.DNSPolicy, v1.DNSClusterFirstWithHostNet)
}

func TestStatefulSetFilterLabels(t *testing.T) {
	excludedLabels := map[string]string{
		"foo":         "1",
		"app.foo.bar": "1",
	}

	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "my-instance",
			Labels: excludedLabels,
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{},
	}

	cfg := config.New(config.WithLabelFilters([]string{"foo*", "app.*.bar"}))

	params := manifests.Params{
		OtelCol: otelcol,
		Config:  cfg,
		Log:     logger,
	}

	d, err := StatefulSet(params)
	require.NoError(t, err)

	assert.Len(t, d.ObjectMeta.Labels, 6)
	for k := range excludedLabels {
		assert.NotContains(t, d.ObjectMeta.Labels, k)
	}
}

func TestStatefulSetFilterAnnotations(t *testing.T) {
	excludedAnnotations := map[string]string{
		"foo":         "1",
		"app.foo.bar": "1",
	}

	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-instance",
			Annotations: excludedAnnotations,
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{},
	}

	cfg := config.New(config.WithAnnotationFilters([]string{"foo*", "app.*.bar"}))

	params := manifests.Params{
		OtelCol: otelcol,
		Config:  cfg,
		Log:     logger,
	}

	d, err := StatefulSet(params)
	require.NoError(t, err)

	assert.Len(t, d.ObjectMeta.Annotations, 0)
	for k := range excludedAnnotations {
		assert.NotContains(t, d.ObjectMeta.Annotations, k)
	}
}

func TestStatefulSetNodeSelector(t *testing.T) {
	// Test default
	otelcol1 := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := manifests.Params{
		OtelCol: otelcol1,
		Config:  cfg,
		Log:     logger,
	}

	d1, err := StatefulSet(params1)
	require.NoError(t, err)

	assert.Empty(t, d1.Spec.Template.Spec.NodeSelector)

	// Test nodeSelector
	otelcol2 := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-nodeselector",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				HostNetwork: true,
				NodeSelector: map[string]string{
					"node-key": "node-value",
				},
			},
		},
	}

	cfg = config.New()

	params2 := manifests.Params{
		OtelCol: otelcol2,
		Config:  cfg,
		Log:     logger,
	}

	d2, err := StatefulSet(params2)
	require.NoError(t, err)
	assert.Equal(t, d2.Spec.Template.Spec.NodeSelector, map[string]string{"node-key": "node-value"})
}

func TestStatefulSetPriorityClassName(t *testing.T) {
	otelcol1 := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := manifests.Params{
		OtelCol: otelcol1,
		Config:  cfg,
		Log:     logger,
	}

	sts1, err := StatefulSet(params1)
	require.NoError(t, err)
	assert.Empty(t, sts1.Spec.Template.Spec.PriorityClassName)

	priorityClassName := "test-class"

	otelcol2 := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-priortyClassName",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				PriorityClassName: priorityClassName,
			},
		},
	}

	cfg = config.New()

	params2 := manifests.Params{
		OtelCol: otelcol2,
		Config:  cfg,
		Log:     logger,
	}

	sts2, err := StatefulSet(params2)
	require.NoError(t, err)
	assert.Equal(t, priorityClassName, sts2.Spec.Template.Spec.PriorityClassName)
}

func TestStatefulSetAffinity(t *testing.T) {
	otelcol1 := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := manifests.Params{
		OtelCol: otelcol1,
		Config:  cfg,
		Log:     logger,
	}

	sts1, err := Deployment(params1)
	require.NoError(t, err)
	assert.Nil(t, sts1.Spec.Template.Spec.Affinity)

	otelcol2 := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-priortyClassName",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Affinity: testAffinityValue,
			},
		},
	}

	cfg = config.New()

	params2 := manifests.Params{
		OtelCol: otelcol2,
		Config:  cfg,
		Log:     logger,
	}

	sts2, err := StatefulSet(params2)
	require.NoError(t, err)
	assert.NotNil(t, sts2.Spec.Template.Spec.Affinity)
	assert.Equal(t, *testAffinityValue, *sts2.Spec.Template.Spec.Affinity)
}

func TestStatefulSetInitContainer(t *testing.T) {
	// prepare
	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-namespace",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				InitContainers: []v1.Container{
					{
						Name: "test",
					},
				},
			},
		},
	}
	cfg := config.New()

	params := manifests.Params{
		OtelCol: otelcol,
		Config:  cfg,
		Log:     logger,
	}

	// test
	s, err := StatefulSet(params)
	require.NoError(t, err)
	assert.Equal(t, "my-instance-collector", s.Name)
	assert.Equal(t, "my-instance-collector", s.Labels["app.kubernetes.io/name"])
	assert.Equal(t, "true", s.Spec.Template.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "8888", s.Spec.Template.Annotations["prometheus.io/port"])
	assert.Equal(t, "/metrics", s.Spec.Template.Annotations["prometheus.io/path"])
	assert.Len(t, s.Spec.Template.Spec.InitContainers, 1)
}

func TestStatefulSetTopologySpreadConstraints(t *testing.T) {
	// Test default
	otelcol1 := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := manifests.Params{
		OtelCol: otelcol1,
		Config:  cfg,
		Log:     logger,
	}
	s1, err := StatefulSet(params1)
	require.NoError(t, err)
	assert.Equal(t, "my-instance-collector", s1.Name)
	assert.Empty(t, s1.Spec.Template.Spec.TopologySpreadConstraints)

	// Test TopologySpreadConstraints
	otelcol2 := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-topologyspreadconstraint",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				TopologySpreadConstraints: testTopologySpreadConstraintValue,
			},
		},
	}

	cfg = config.New()

	params2 := manifests.Params{
		OtelCol: otelcol2,
		Config:  cfg,
		Log:     logger,
	}

	s2, err := StatefulSet(params2)
	require.NoError(t, err)
	assert.Equal(t, "my-instance-topologyspreadconstraint-collector", s2.Name)
	assert.NotNil(t, s2.Spec.Template.Spec.TopologySpreadConstraints)
	assert.NotEmpty(t, s2.Spec.Template.Spec.TopologySpreadConstraints)
	assert.Equal(t, testTopologySpreadConstraintValue, s2.Spec.Template.Spec.TopologySpreadConstraints)
}

func TestStatefulSetAdditionalContainers(t *testing.T) {
	// prepare
	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-namespace",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				AdditionalContainers: []v1.Container{
					{
						Name: "test",
					},
				},
			},
		},
	}
	cfg := config.New()

	params := manifests.Params{
		OtelCol: otelcol,
		Config:  cfg,
		Log:     logger,
	}

	// test
	s, err := StatefulSet(params)
	require.NoError(t, err)
	assert.Equal(t, "my-instance-collector", s.Name)
	assert.Equal(t, "my-instance-collector", s.Labels["app.kubernetes.io/name"])
	assert.Equal(t, "true", s.Spec.Template.Annotations["prometheus.io/scrape"])
	assert.Equal(t, "8888", s.Spec.Template.Annotations["prometheus.io/port"])
	assert.Equal(t, "/metrics", s.Spec.Template.Annotations["prometheus.io/path"])
	assert.Len(t, s.Spec.Template.Spec.Containers, 2)
	assert.Equal(t, v1.Container{Name: "test"}, s.Spec.Template.Spec.Containers[0])
}

func TestStatefulSetShareProcessNamespace(t *testing.T) {
	// Test default
	otelcol1 := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := manifests.Params{
		OtelCol: otelcol1,
		Config:  cfg,
		Log:     logger,
	}

	d1, err := StatefulSet(params1)
	require.NoError(t, err)
	assert.False(t, *d1.Spec.Template.Spec.ShareProcessNamespace)

	// Test shareProcessNamespace=true
	otelcol2 := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-with-shareprocessnamespace",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				ShareProcessNamespace: true,
			},
		},
	}

	cfg = config.New()

	params2 := manifests.Params{
		OtelCol: otelcol2,
		Config:  cfg,
		Log:     logger,
	}

	d2, err := StatefulSet(params2)
	require.NoError(t, err)
	assert.True(t, *d2.Spec.Template.Spec.ShareProcessNamespace)
}

func TestStatefulSetDNSConfig(t *testing.T) {
	// prepare
	otelcol := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-namespace",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				PodDNSConfig: v1.PodDNSConfig{
					Nameservers: []string{"8.8.8.8"},
					Searches:    []string{"my.dns.search.suffix"},
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
	d, err := StatefulSet(params)
	require.NoError(t, err)
	assert.Equal(t, "my-instance-collector", d.Name)
	assert.Equal(t, v1.DNSPolicy("None"), d.Spec.Template.Spec.DNSPolicy)
	assert.Equal(t, d.Spec.Template.Spec.DNSConfig.Nameservers, []string{"8.8.8.8"})
}
