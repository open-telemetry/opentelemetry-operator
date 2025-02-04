// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	go_yaml "gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
)

var testTolerationValues = []v1.Toleration{
	{
		Key:    "hii",
		Value:  "greeting",
		Effect: "NoSchedule",
	},
}

var testTopologySpreadConstraintValue = []v1.TopologySpreadConstraint{
	{
		MaxSkew:           1,
		TopologyKey:       "kubernetes.io/hostname",
		WhenUnsatisfiable: "DoNotSchedule",
		LabelSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"foo": "bar",
			},
		},
	},
}

var testAffinityValue = &v1.Affinity{
	NodeAffinity: &v1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
			NodeSelectorTerms: []v1.NodeSelectorTerm{
				{
					MatchExpressions: []v1.NodeSelectorRequirement{
						{
							Key:      "node",
							Operator: v1.NodeSelectorOpIn,
							Values:   []string{"test-node"},
						},
					},
				},
			},
		},
	},
}

var runAsUser int64 = 1000
var runAsGroup int64 = 1000

var testSecurityContextValue = &v1.PodSecurityContext{
	RunAsUser:  &runAsUser,
	RunAsGroup: &runAsGroup,
}

func TestDeploymentSecurityContext(t *testing.T) {
	// Test default
	targetallocator11 := v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := Params{
		TargetAllocator: targetallocator11,
		Config:          cfg,
		Log:             logger,
	}
	d1, err := Deployment(params1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Empty(t, d1.Spec.Template.Spec.SecurityContext)

	// Test SecurityContext
	targetAllocator2 := v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-securitycontext",
		},
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				PodSecurityContext: testSecurityContextValue,
			},
		},
	}

	cfg = config.New()

	params2 := Params{
		TargetAllocator: targetAllocator2,
		Config:          cfg,
		Log:             logger,
	}

	d2, err := Deployment(params2)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, *testSecurityContextValue, *d2.Spec.Template.Spec.SecurityContext)
}

func TestDeploymentNewDefault(t *testing.T) {
	// prepare
	otelcol := collectorInstance()
	targetAllocator := targetAllocatorInstance()
	cfg := config.New()

	params := Params{
		Collector:       otelcol,
		TargetAllocator: targetAllocator,
		Config:          cfg,
		Log:             logger,
	}

	// test
	d, err := Deployment(params)

	assert.NoError(t, err)

	// verify
	assert.Equal(t, "my-instance-targetallocator", d.GetName())
	assert.Equal(t, "my-instance-targetallocator", d.GetLabels()["app.kubernetes.io/name"])

	assert.Len(t, d.Spec.Template.Spec.Containers, 1)

	// should only have the ConfigMap hash annotation
	assert.Contains(t, d.Spec.Template.Annotations, configMapHashAnnotationKey)
	assert.Len(t, d.Spec.Template.Annotations, 1)

	// the pod selector should match the pod spec's labels
	assert.Subset(t, d.Spec.Template.Labels, d.Spec.Selector.MatchLabels)
}

func TestDeploymentPodAnnotations(t *testing.T) {
	// prepare
	testPodAnnotationValues := map[string]string{"annotation-key": "annotation-value"}
	otelcol := collectorInstance()
	targetAllocator := targetAllocatorInstance()
	targetAllocator.Spec.PodAnnotations = testPodAnnotationValues
	cfg := config.New()

	params := Params{
		Collector:       otelcol,
		TargetAllocator: targetAllocator,
		Config:          cfg,
		Log:             logger,
	}

	// test
	ds, err := Deployment(params)
	assert.NoError(t, err)
	// verify
	assert.Equal(t, "my-instance-targetallocator", ds.Name)
	assert.Subset(t, ds.Spec.Template.Annotations, testPodAnnotationValues)
}

func collectorInstance() *v1beta1.OpenTelemetryCollector {
	configYAML, err := os.ReadFile("testdata/test.yaml")
	if err != nil {
		fmt.Printf("Error getting yaml file: %v", err)
	}
	cfg := v1beta1.Config{}
	err = go_yaml.Unmarshal(configYAML, &cfg)
	if err != nil {
		fmt.Printf("Error unmarshalling YAML: %v", err)
	}
	return &v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "default",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Image: "ghcr.io/open-telemetry/opentelemetry-operator/opentelemetry-operator:0.47.0",
			},
			Config: cfg,
			TargetAllocator: v1beta1.TargetAllocatorEmbedded{
				Image:          "ghcr.io/open-telemetry/opentelemetry-operator/opentelemetry-targetallocator:0.47.0",
				FilterStrategy: "relabel-config",
			},
		},
	}
}

func targetAllocatorInstance() v1alpha1.TargetAllocator {
	collectorInstance := collectorInstance()
	collectorInstance.Spec.TargetAllocator.Enabled = true
	params := manifests.Params{OtelCol: *collectorInstance}
	targetAllocator, _ := collector.TargetAllocator(params)
	targetAllocator.Spec.Image = "ghcr.io/open-telemetry/opentelemetry-operator/opentelemetry-targetallocator:0.47.0"
	return *targetAllocator
}

func TestDeploymentNodeSelector(t *testing.T) {
	// Test default
	targetAllocator1 := v1alpha1.TargetAllocator{}

	cfg := config.New()

	params1 := Params{
		TargetAllocator: targetAllocator1,
		Config:          cfg,
		Log:             logger,
	}
	d1, err := Deployment(params1)
	assert.NoError(t, err)
	assert.Empty(t, d1.Spec.Template.Spec.NodeSelector)

	// Test nodeSelector
	targetAllocator2 := v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-nodeselector",
		},
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				NodeSelector: map[string]string{
					"node-key": "node-value",
				},
			},
		},
	}

	cfg = config.New()

	params2 := Params{
		TargetAllocator: targetAllocator2,
		Config:          cfg,
		Log:             logger,
	}

	d2, err := Deployment(params2)
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"node-key": "node-value"}, d2.Spec.Template.Spec.NodeSelector)
}

func TestDeploymentAffinity(t *testing.T) {
	// Test default
	targetAllocator1 := v1alpha1.TargetAllocator{}

	cfg := config.New()

	params1 := Params{
		TargetAllocator: targetAllocator1,
		Config:          cfg,
		Log:             logger,
	}
	d1, err := Deployment(params1)
	assert.NoError(t, err)
	assert.Empty(t, d1.Spec.Template.Spec.Affinity)

	// Test affinity
	targetAllocator2 := v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-affinity",
		},
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Affinity: testAffinityValue,
			},
		},
	}

	cfg = config.New()

	params2 := Params{
		TargetAllocator: targetAllocator2,
		Config:          cfg,
		Log:             logger,
	}

	d2, err := Deployment(params2)
	assert.NoError(t, err)
	assert.Equal(t, *testAffinityValue, *d2.Spec.Template.Spec.Affinity)
}

func TestDeploymentTolerations(t *testing.T) {
	// Test default
	targetAllocator1 := v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()
	params1 := Params{
		TargetAllocator: targetAllocator1,
		Config:          cfg,
		Log:             logger,
	}
	d1, err := Deployment(params1)
	assert.NoError(t, err)
	assert.Equal(t, "my-instance-targetallocator", d1.Name)
	assert.Empty(t, d1.Spec.Template.Spec.Tolerations)

	// Test Tolerations
	targetAllocator2 := v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-toleration",
		},
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Tolerations: testTolerationValues,
			},
		},
	}

	params2 := Params{
		TargetAllocator: targetAllocator2,
		Config:          cfg,
		Log:             logger,
	}
	d2, err := Deployment(params2)
	assert.NoError(t, err)
	assert.Equal(t, "my-instance-toleration-targetallocator", d2.Name)
	assert.NotNil(t, d2.Spec.Template.Spec.Tolerations)
	assert.NotEmpty(t, d2.Spec.Template.Spec.Tolerations)
	assert.Equal(t, testTolerationValues, d2.Spec.Template.Spec.Tolerations)
}

func TestDeploymentTopologySpreadConstraints(t *testing.T) {
	// Test default
	targetAllocator1 := v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := Params{
		TargetAllocator: targetAllocator1,
		Config:          cfg,
		Log:             logger,
	}
	d1, err := Deployment(params1)
	assert.NoError(t, err)
	assert.Equal(t, "my-instance-targetallocator", d1.Name)
	assert.Empty(t, d1.Spec.Template.Spec.TopologySpreadConstraints)

	// Test TopologySpreadConstraints
	targetAllocator2 := v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-topologyspreadconstraint",
		},
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				TopologySpreadConstraints: testTopologySpreadConstraintValue,
			},
		},
	}

	cfg = config.New()
	params2 := Params{
		TargetAllocator: targetAllocator2,
		Config:          cfg,
		Log:             logger,
	}

	d2, err := Deployment(params2)
	assert.NoError(t, err)
	assert.Equal(t, "my-instance-topologyspreadconstraint-targetallocator", d2.Name)
	assert.NotNil(t, d2.Spec.Template.Spec.TopologySpreadConstraints)
	assert.NotEmpty(t, d2.Spec.Template.Spec.TopologySpreadConstraints)
	assert.Equal(t, testTopologySpreadConstraintValue, d2.Spec.Template.Spec.TopologySpreadConstraints)
}

func TestDeploymentSetInitContainer(t *testing.T) {
	// prepare
	targetAllocator := targetAllocatorInstance()
	targetAllocator.Spec.InitContainers = []v1.Container{
		{
			Name: "test",
		},
	}
	otelcol := collectorInstance()
	params := Params{
		Collector:       otelcol,
		TargetAllocator: targetAllocator,
		Config:          config.New(),
		Log:             logger,
	}

	// test
	d, err := Deployment(params)
	require.NoError(t, err)
	assert.Len(t, d.Spec.Template.Spec.InitContainers, 1)
}

func TestDeploymentAdditionalContainers(t *testing.T) {
	// prepare
	targetAllocator := targetAllocatorInstance()
	targetAllocator.Spec.AdditionalContainers = []v1.Container{
		{
			Name: "test",
		},
	}
	otelcol := collectorInstance()
	params := Params{
		Collector:       otelcol,
		TargetAllocator: targetAllocator,
		Config:          config.New(),
		Log:             logger,
	}

	// test
	d, err := Deployment(params)
	require.NoError(t, err)
	assert.Len(t, d.Spec.Template.Spec.Containers, 2)
	assert.Equal(t, v1.Container{Name: "test"}, d.Spec.Template.Spec.Containers[0])
}

func TestDeploymentHostNetwork(t *testing.T) {
	// Test default
	targetAllocator := targetAllocatorInstance()
	otelcol := collectorInstance()
	params := Params{
		Collector:       otelcol,
		TargetAllocator: targetAllocator,
		Config:          config.New(),
		Log:             logger,
	}

	d1, err := Deployment(params)
	require.NoError(t, err)

	assert.Equal(t, d1.Spec.Template.Spec.HostNetwork, false)
	assert.Equal(t, d1.Spec.Template.Spec.DNSPolicy, v1.DNSClusterFirst)

	// Test hostNetwork=true
	params.TargetAllocator.Spec.HostNetwork = true

	d2, err := Deployment(params)
	require.NoError(t, err)
	assert.Equal(t, d2.Spec.Template.Spec.HostNetwork, true)
	assert.Equal(t, d2.Spec.Template.Spec.DNSPolicy, v1.DNSClusterFirstWithHostNet)
}

func TestDeploymentShareProcessNamespace(t *testing.T) {
	// Test default
	targetAllocator := targetAllocatorInstance()
	otelcol := collectorInstance()
	params := Params{
		Collector:       otelcol,
		TargetAllocator: targetAllocator,
		Config:          config.New(),
		Log:             logger,
	}

	d1, err := Deployment(params)
	require.NoError(t, err)
	assert.False(t, *d1.Spec.Template.Spec.ShareProcessNamespace)

	// Test ShareProcessNamespace=true
	params.TargetAllocator.Spec.ShareProcessNamespace = true

	d2, err := Deployment(params)
	require.NoError(t, err)
	assert.True(t, *d2.Spec.Template.Spec.ShareProcessNamespace)
}

func TestDeploymentPriorityClassName(t *testing.T) {
	// Test default
	targetAllocator := targetAllocatorInstance()
	otelcol := collectorInstance()
	params := Params{
		Collector:       otelcol,
		TargetAllocator: targetAllocator,
		Config:          config.New(),
		Log:             logger,
	}

	d1, err := Deployment(params)
	require.NoError(t, err)
	assert.Empty(t, d1.Spec.Template.Spec.PriorityClassName)

	// Test PriorityClassName
	params.TargetAllocator.Spec.PriorityClassName = "test-class"

	d2, err := Deployment(params)
	require.NoError(t, err)
	assert.Equal(t, params.TargetAllocator.Spec.PriorityClassName, d2.Spec.Template.Spec.PriorityClassName)
}

func TestDeploymentTerminationGracePeriodSeconds(t *testing.T) {
	// Test default
	targetAllocator := targetAllocatorInstance()
	otelcol := collectorInstance()
	params := Params{
		Collector:       otelcol,
		TargetAllocator: targetAllocator,
		Config:          config.New(),
		Log:             logger,
	}

	d1, err := Deployment(params)
	require.NoError(t, err)
	assert.Nil(t, d1.Spec.Template.Spec.TerminationGracePeriodSeconds)

	// Test TerminationGracePeriodSeconds
	gracePeriod := int64(100)
	params.TargetAllocator.Spec.TerminationGracePeriodSeconds = &gracePeriod

	d2, err := Deployment(params)
	require.NoError(t, err)
	assert.Equal(t, gracePeriod, *d2.Spec.Template.Spec.TerminationGracePeriodSeconds)
}

func TestDeploymentDNSConfig(t *testing.T) {
	// Test default
	otelcol := collectorInstance()
	// prepare
	targetAllocator := v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-namespace",
		},
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				PodDNSConfig: v1.PodDNSConfig{
					Nameservers: []string{"8.8.8.8"},
					Searches:    []string{"my.dns.search.suffix"},
				},
			},
		},
	}
	params := Params{
		Collector:       otelcol,
		TargetAllocator: targetAllocator,
		Config:          config.New(),
		Log:             logger,
	}

	// test
	d, err := Deployment(params)
	require.NoError(t, err)
	assert.Equal(t, "my-instance-targetallocator", d.Name)
	assert.Equal(t, v1.DNSPolicy("None"), d.Spec.Template.Spec.DNSPolicy)
	assert.Equal(t, d.Spec.Template.Spec.DNSConfig.Nameservers, []string{"8.8.8.8"})
}
