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

package targetallocator

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	go_yaml "gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	targetallocator11 := v1beta1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := manifests.Params{
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
	targetAllocator2 := v1beta1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-securitycontext",
		},
		Spec: v1beta1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				PodSecurityContext: testSecurityContextValue,
			},
		},
	}

	cfg = config.New()

	params2 := manifests.Params{
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

	params := manifests.Params{
		OtelCol:         otelcol,
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

	params := manifests.Params{
		OtelCol:         otelcol,
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

func collectorInstance() v1beta1.OpenTelemetryCollector {
	configYAML, err := os.ReadFile("testdata/test.yaml")
	if err != nil {
		fmt.Printf("Error getting yaml file: %v", err)
	}
	cfg := v1beta1.Config{}
	err = go_yaml.Unmarshal(configYAML, &cfg)
	if err != nil {
		fmt.Printf("Error unmarshalling YAML: %v", err)
	}
	return v1beta1.OpenTelemetryCollector{
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

func targetAllocatorInstance() v1beta1.TargetAllocator {
	collectorInstance := collectorInstance()
	collectorInstance.Spec.TargetAllocator.Enabled = true
	params := manifests.Params{OtelCol: collectorInstance}
	targetAllocator, _ := collector.TargetAllocator(params)
	targetAllocator.Spec.Image = "ghcr.io/open-telemetry/opentelemetry-operator/opentelemetry-targetallocator:0.47.0"
	return *targetAllocator
}

func TestDeploymentNodeSelector(t *testing.T) {
	// Test default
	targetAllocator1 := v1beta1.TargetAllocator{}

	cfg := config.New()

	params1 := manifests.Params{
		TargetAllocator: targetAllocator1,
		Config:          cfg,
		Log:             logger,
	}
	d1, err := Deployment(params1)
	assert.NoError(t, err)
	assert.Empty(t, d1.Spec.Template.Spec.NodeSelector)

	// Test nodeSelector
	targetAllocator2 := v1beta1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-nodeselector",
		},
		Spec: v1beta1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				NodeSelector: map[string]string{
					"node-key": "node-value",
				},
			},
		},
	}

	cfg = config.New()

	params2 := manifests.Params{
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
	targetAllocator1 := v1beta1.TargetAllocator{}

	cfg := config.New()

	params1 := manifests.Params{
		TargetAllocator: targetAllocator1,
		Config:          cfg,
		Log:             logger,
	}
	d1, err := Deployment(params1)
	assert.NoError(t, err)
	assert.Empty(t, d1.Spec.Template.Spec.Affinity)

	// Test affinity
	targetAllocator2 := v1beta1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-affinity",
		},
		Spec: v1beta1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Affinity: testAffinityValue,
			},
		},
	}

	cfg = config.New()

	params2 := manifests.Params{
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
	targetAllocator1 := v1beta1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()
	params1 := manifests.Params{
		TargetAllocator: targetAllocator1,
		Config:          cfg,
		Log:             logger,
	}
	d1, err := Deployment(params1)
	assert.NoError(t, err)
	assert.Equal(t, "my-instance-targetallocator", d1.Name)
	assert.Empty(t, d1.Spec.Template.Spec.Tolerations)

	// Test Tolerations
	targetAllocator2 := v1beta1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-toleration",
		},
		Spec: v1beta1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Tolerations: testTolerationValues,
			},
		},
	}

	params2 := manifests.Params{
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
	targetAllocator1 := v1beta1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance",
		},
	}

	cfg := config.New()

	params1 := manifests.Params{
		TargetAllocator: targetAllocator1,
		Config:          cfg,
		Log:             logger,
	}
	d1, err := Deployment(params1)
	assert.NoError(t, err)
	assert.Equal(t, "my-instance-targetallocator", d1.Name)
	assert.Empty(t, d1.Spec.Template.Spec.TopologySpreadConstraints)

	// Test TopologySpreadConstraints
	targetAllocator2 := v1beta1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-instance-topologyspreadconstraint",
		},
		Spec: v1beta1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				TopologySpreadConstraints: testTopologySpreadConstraintValue,
			},
		},
	}

	cfg = config.New()
	params2 := manifests.Params{
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
