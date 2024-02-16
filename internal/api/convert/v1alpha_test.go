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

package convert

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha2"
)

func Test_V1Alpha1to2(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := `---
receivers:
  otlp:
    protocols:
      grpc:
processors:
  resourcedetection:
    detectors: [kubernetes]
exporters:
  otlp:
    endpoint: "otlp:4317"
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [resourcedetection]
      exporters: [otlp]
`
		cfgV1 := v1alpha1.OpenTelemetryCollector{
			Spec: v1alpha1.OpenTelemetryCollectorSpec{
				Config: config,
			},
		}

		cfgV2, err := V1Alpha1to2(cfgV1)
		assert.Nil(t, err)
		assert.NotNil(t, cfgV2)

		yamlCfg, err := yaml.Marshal(&cfgV2.Spec.Config)
		assert.Nil(t, err)
		assert.YAMLEq(t, config, string(yamlCfg))
	})
	t.Run("invalid config", func(t *testing.T) {
		config := `!!!`
		cfgV1 := v1alpha1.OpenTelemetryCollector{
			Spec: v1alpha1.OpenTelemetryCollectorSpec{
				Config: config,
			},
		}

		_, err := V1Alpha1to2(cfgV1)
		assert.ErrorContains(t, err, "could not convert config json to v1alpha2.Config")
	})
}

func Test_TargetAllocator(t *testing.T) {
	replicas := int32(2)
	runAsNonRoot := true
	privileged := true
	runAsUser := int64(1337)
	runasGroup := int64(1338)
	input := v1alpha1.OpenTelemetryTargetAllocator{
		Replicas:     &replicas,
		NodeSelector: map[string]string{"key": "value"},
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Requests: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("500m"),
				v1.ResourceMemory: resource.MustParse("128Mi"),
			},
		},
		AllocationStrategy: v1alpha1.OpenTelemetryTargetAllocatorAllocationStrategyConsistentHashing,
		FilterStrategy:     "relabel-config",
		ServiceAccount:     "serviceAccountName",
		Image:              "custom_image",
		Enabled:            true,
		Affinity: &v1.Affinity{
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
		},
		PrometheusCR: v1alpha1.OpenTelemetryTargetAllocatorPrometheusCR{
			Enabled:                true,
			ScrapeInterval:         &metav1.Duration{Duration: time.Second},
			PodMonitorSelector:     map[string]string{"podmonitorkey": "podmonitorvalue"},
			ServiceMonitorSelector: map[string]string{"servicemonitorkey": "servicemonitorkey"},
		},
		PodSecurityContext: &v1.PodSecurityContext{
			RunAsNonRoot: &runAsNonRoot,
			RunAsUser:    &runAsUser,
			RunAsGroup:   &runasGroup,
		},
		SecurityContext: &v1.SecurityContext{
			RunAsUser:  &runAsUser,
			Privileged: &privileged,
		},
		TopologySpreadConstraints: []v1.TopologySpreadConstraint{
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
		},
		Tolerations: []v1.Toleration{
			{
				Key:    "hii",
				Value:  "greeting",
				Effect: "NoSchedule",
			},
		},
		Env: []v1.EnvVar{
			{
				Name: "POD_NAME",
				ValueFrom: &v1.EnvVarSource{
					FieldRef: &v1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		},
		Observability: v1alpha1.ObservabilitySpec{
			Metrics: v1alpha1.MetricsConfigSpec{
				EnableMetrics: true,
			},
		},
		PodDisruptionBudget: &v1alpha1.PodDisruptionBudgetSpec{
			MaxUnavailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			},
		},
	}

	expected := v1alpha2.TargetAllocatorEmbedded{
		Replicas:           input.Replicas,
		NodeSelector:       input.NodeSelector,
		Resources:          input.Resources,
		AllocationStrategy: v1alpha2.TargetAllocatorAllocationStrategyConsistentHashing,
		FilterStrategy:     v1alpha2.TargetAllocatorFilterStrategyRelabelConfig,
		ServiceAccount:     input.ServiceAccount,
		Image:              input.Image,
		Enabled:            input.Enabled,
		Affinity:           input.Affinity,
		PrometheusCR: v1alpha2.TargetAllocatorPrometheusCR{
			Enabled:        input.PrometheusCR.Enabled,
			ScrapeInterval: input.PrometheusCR.ScrapeInterval,
			PodMonitorSelector: &metav1.LabelSelector{
				MatchLabels: input.PrometheusCR.PodMonitorSelector,
			},
			ServiceMonitorSelector: &metav1.LabelSelector{
				MatchLabels: input.PrometheusCR.ServiceMonitorSelector,
			},
		},
		SecurityContext:           input.SecurityContext,
		PodSecurityContext:        input.PodSecurityContext,
		TopologySpreadConstraints: input.TopologySpreadConstraints,
		Tolerations:               input.Tolerations,
		Env:                       input.Env,
		Observability:             input.Observability,
		PodDisruptionBudget: &v1alpha2.PodDisruptionBudgetSpec{
			MinAvailable:   input.PodDisruptionBudget.MinAvailable,
			MaxUnavailable: input.PodDisruptionBudget.MaxUnavailable,
		},
	}

	assert.Equal(t, expected, TargetAllocatorEmbedded(input))
}
