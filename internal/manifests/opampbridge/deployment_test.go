// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opampbridge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
)

var testTolerationValues = []v1.Toleration{
	{
		Key:    "hii",
		Value:  "greeting",
		Effect: "NoSchedule",
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

func TestDeploymentNewDefault(t *testing.T) {
	// prepare
	opampBridge := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-namespace",
		},
		Spec: v1alpha1.OpAMPBridgeSpec{
			Tolerations: testTolerationValues,
		},
	}
	cfg := config.New()

	params := manifests.Params{
		Config:      cfg,
		OpAMPBridge: opampBridge,
		Log:         logger,
	}

	// test
	d := Deployment(params)

	// verify
	assert.Equal(t, "my-instance-opamp-bridge", d.Name)
	assert.Equal(t, "my-instance-opamp-bridge", d.Labels["app.kubernetes.io/name"])
	assert.Equal(t, testTolerationValues, d.Spec.Template.Spec.Tolerations)

	assert.Len(t, d.Spec.Template.Spec.Containers, 1)

	expectedLabels := map[string]string{
		"app.kubernetes.io/component":  "opentelemetry-opamp-bridge",
		"app.kubernetes.io/instance":   "my-namespace.my-instance",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/name":       "my-instance-opamp-bridge",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"app.kubernetes.io/version":    "latest",
	}
	assert.Equal(t, expectedLabels, d.Spec.Template.Labels)

	expectedSelectorLabels := map[string]string{
		"app.kubernetes.io/component":  "opentelemetry-opamp-bridge",
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

func TestDeploymentPodAnnotations(t *testing.T) {
	// prepare
	testPodAnnotationValues := map[string]string{"annotation-key": "annotation-value"}
	opampBridge := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1alpha1.OpAMPBridgeSpec{
			PodAnnotations: testPodAnnotationValues,
		},
	}
	cfg := config.New()

	params := manifests.Params{
		Config:      cfg,
		OpAMPBridge: opampBridge,
		Log:         logger,
	}

	// test
	d := Deployment(params)

	// verify
	assert.Len(t, d.Spec.Template.Annotations, 1)
	assert.Equal(t, "my-instance-opamp-bridge", d.Name)
	assert.Equal(t, testPodAnnotationValues, d.Spec.Template.Annotations)
}

func TestDeploymentPodSecurityContext(t *testing.T) {
	runAsNonRoot := true
	runAsUser := int64(1337)
	runasGroup := int64(1338)

	opampBridge := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
		Spec: v1alpha1.OpAMPBridgeSpec{
			PodSecurityContext: &v1.PodSecurityContext{
				RunAsNonRoot: &runAsNonRoot,
				RunAsUser:    &runAsUser,
				RunAsGroup:   &runasGroup,
			},
		},
	}

	cfg := config.New()

	params := manifests.Params{
		Config:      cfg,
		OpAMPBridge: opampBridge,
		Log:         logger,
	}

	d := Deployment(params)

	assert.Equal(t, &runAsNonRoot, d.Spec.Template.Spec.SecurityContext.RunAsNonRoot)
	assert.Equal(t, &runAsUser, d.Spec.Template.Spec.SecurityContext.RunAsUser)
	assert.Equal(t, &runasGroup, d.Spec.Template.Spec.SecurityContext.RunAsGroup)
}

func TestDeploymentHostNetwork(t *testing.T) {
	// Test default
	opampBridge1 := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := manifests.Params{
		Config:      cfg,
		OpAMPBridge: opampBridge1,
		Log:         logger,
	}

	d1 := Deployment(params1)

	assert.Equal(t, d1.Spec.Template.Spec.HostNetwork, false)
	assert.Equal(t, d1.Spec.Template.Spec.DNSPolicy, v1.DNSClusterFirst)

	// Test hostNetwork=true
	opampBridge2 := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-hostnetwork",
		},
		Spec: v1alpha1.OpAMPBridgeSpec{
			HostNetwork: true,
		},
	}

	cfg = config.New()

	params2 := manifests.Params{
		Config:      cfg,
		OpAMPBridge: opampBridge2,
		Log:         logger,
	}

	d2 := Deployment(params2)
	assert.Equal(t, d2.Spec.Template.Spec.HostNetwork, true)
	assert.Equal(t, d2.Spec.Template.Spec.DNSPolicy, v1.DNSClusterFirstWithHostNet)
}

func TestDeploymentFilterLabels(t *testing.T) {
	excludedLabels := map[string]string{
		"foo":         "1",
		"app.foo.bar": "1",
	}

	opampBridge := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "my-instance",
			Labels: excludedLabels,
		},
		Spec: v1alpha1.OpAMPBridgeSpec{},
	}

	cfg := config.New(config.WithLabelFilters([]string{"foo*", "app.*.bar"}))

	params := manifests.Params{
		Config:      cfg,
		OpAMPBridge: opampBridge,
		Log:         logger,
	}

	d := Deployment(params)

	assert.Len(t, d.ObjectMeta.Labels, 6)
	for k := range excludedLabels {
		assert.NotContains(t, d.ObjectMeta.Labels, k)
	}
}

func TestDeploymentFilterAnnotations(t *testing.T) {
	excludedAnnotations := map[string]string{
		"foo":         "1",
		"app.foo.bar": "1",
		"opampbridge": "true",
	}

	opampBridge := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-instance",
			Annotations: excludedAnnotations,
		},
		Spec: v1alpha1.OpAMPBridgeSpec{},
	}

	cfg := config.New(config.WithAnnotationFilters([]string{"foo*", "app.*.bar"}))

	params := manifests.Params{
		Config:      cfg,
		OpAMPBridge: opampBridge,
		Log:         logger,
	}

	d := Deployment(params)

	assert.Len(t, d.ObjectMeta.Annotations, 2)
	assert.NotContains(t, d.ObjectMeta.Annotations, "foo")
	assert.NotContains(t, d.ObjectMeta.Annotations, "app.foo.bar")
}

func TestDeploymentNodeSelector(t *testing.T) {
	// Test default
	opampBridge1 := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := manifests.Params{
		Config:      cfg,
		OpAMPBridge: opampBridge1,
		Log:         logger,
	}

	d1 := Deployment(params1)

	assert.Empty(t, d1.Spec.Template.Spec.NodeSelector)

	// Test nodeSelector
	opampBridge2 := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-nodeselector",
		},
		Spec: v1alpha1.OpAMPBridgeSpec{
			HostNetwork: true,
			NodeSelector: map[string]string{
				"node-key": "node-value",
			},
		},
	}

	cfg = config.New()

	params2 := manifests.Params{
		Config:      cfg,
		OpAMPBridge: opampBridge2,
		Log:         logger,
	}

	d2 := Deployment(params2)
	assert.Equal(t, d2.Spec.Template.Spec.NodeSelector, map[string]string{"node-key": "node-value"})
}

func TestDeploymentPriorityClassName(t *testing.T) {
	opampBridge1 := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := manifests.Params{
		Config:      cfg,
		OpAMPBridge: opampBridge1,
		Log:         logger,
	}

	d1 := Deployment(params1)
	assert.Empty(t, d1.Spec.Template.Spec.PriorityClassName)

	priorityClassName := "test-class"

	opampBridge2 := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-priortyClassName",
		},
		Spec: v1alpha1.OpAMPBridgeSpec{
			PriorityClassName: priorityClassName,
		},
	}

	cfg = config.New()

	params2 := manifests.Params{
		Config:      cfg,
		OpAMPBridge: opampBridge2,
		Log:         logger,
	}

	d2 := Deployment(params2)
	assert.Equal(t, priorityClassName, d2.Spec.Template.Spec.PriorityClassName)
}

func TestDeploymentAffinity(t *testing.T) {
	opampBridge1 := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := manifests.Params{
		Config:      cfg,
		OpAMPBridge: opampBridge1,
		Log:         logger,
	}

	d1 := Deployment(params1)
	assert.Nil(t, d1.Spec.Template.Spec.Affinity)

	opampBridge2 := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-priortyClassName",
		},
		Spec: v1alpha1.OpAMPBridgeSpec{
			Affinity: testAffinityValue,
		},
	}

	cfg = config.New()

	params2 := manifests.Params{
		Config:      cfg,
		OpAMPBridge: opampBridge2,
		Log:         logger,
	}

	d2 := Deployment(params2)
	assert.NotNil(t, d2.Spec.Template.Spec.Affinity)
	assert.Equal(t, *testAffinityValue, *d2.Spec.Template.Spec.Affinity)
}

func TestDeploymentTopologySpreadConstraints(t *testing.T) {
	// Test default
	opampBridge1 := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := manifests.Params{
		Config:      cfg,
		OpAMPBridge: opampBridge1,
		Log:         logger,
	}

	d1 := Deployment(params1)
	assert.Equal(t, "my-instance-opamp-bridge", d1.Name)
	assert.Empty(t, d1.Spec.Template.Spec.TopologySpreadConstraints)

	// Test TopologySpreadConstraints
	opampBridge2 := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-topologyspreadconstraint",
		},
		Spec: v1alpha1.OpAMPBridgeSpec{
			TopologySpreadConstraints: testTopologySpreadConstraintValue,
		},
	}

	cfg = config.New()

	params2 := manifests.Params{
		Config:      cfg,
		OpAMPBridge: opampBridge2,
		Log:         logger,
	}

	d2 := Deployment(params2)
	assert.Equal(t, "my-instance-topologyspreadconstraint-opamp-bridge", d2.Name)
	assert.NotNil(t, d2.Spec.Template.Spec.TopologySpreadConstraints)
	assert.NotEmpty(t, d2.Spec.Template.Spec.TopologySpreadConstraints)
	assert.Equal(t, testTopologySpreadConstraintValue, d2.Spec.Template.Spec.TopologySpreadConstraints)
}

func TestDeploymentDNSConfig(t *testing.T) {
	// prepare
	opAmpBridge := v1alpha1.OpAMPBridge{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "my-namespace",
		},
		Spec: v1alpha1.OpAMPBridgeSpec{
			PodDNSConfig: v1.PodDNSConfig{
				Nameservers: []string{"8.8.8.8"},
				Searches:    []string{"my.dns.search.suffix"},
			},
		},
	}

	cfg := config.New()

	params := manifests.Params{
		Config:      cfg,
		OpAMPBridge: opAmpBridge,
		Log:         logger,
	}

	// test
	d := Deployment(params)
	assert.Equal(t, "my-instance-opamp-bridge", d.Name)
	assert.Equal(t, v1.DNSPolicy("None"), d.Spec.Template.Spec.DNSPolicy)
	assert.Equal(t, d.Spec.Template.Spec.DNSConfig.Nameservers, []string{"8.8.8.8"})
}
