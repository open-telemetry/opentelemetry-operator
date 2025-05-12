// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

const collectorCfg = `---
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

func Test_tov1beta1_config(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfgV1 := OpenTelemetryCollector{
			Spec: OpenTelemetryCollectorSpec{
				Config: collectorCfg,
				Args: map[string]string{
					"test": "something",
				},
			},
		}

		cfgV2, err := tov1beta1(cfgV1)
		assert.Nil(t, err)
		assert.NotNil(t, cfgV2)
		assert.Equal(t, cfgV1.Spec.Args, cfgV2.Spec.Args)

		yamlCfg, err := yaml.Marshal(&cfgV2.Spec.Config)
		assert.Nil(t, err)
		assert.YAMLEq(t, collectorCfg, string(yamlCfg))
	})
	t.Run("invalid config", func(t *testing.T) {
		config := `!!!`
		cfgV1 := OpenTelemetryCollector{
			Spec: OpenTelemetryCollectorSpec{
				Config: config,
			},
		}

		_, err := tov1beta1(cfgV1)
		assert.ErrorContains(t, err, "could not convert config json to v1beta1.Config")
	})
}

func Test_tov1alpha1_config(t *testing.T) {
	cfg := v1beta1.Config{}
	err := yaml.Unmarshal([]byte(collectorCfg), &cfg)
	require.NoError(t, err)

	beta1Col := v1beta1.OpenTelemetryCollector{
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Config: cfg,
		},
	}
	alpha1Col, err := tov1alpha1(beta1Col)
	require.NoError(t, err)
	assert.YAMLEq(t, collectorCfg, alpha1Col.Spec.Config)
}

func Test_tov1beta1AndBack(t *testing.T) {
	one := int32(1)
	two := int64(2)
	intstrAAA := intstr.FromString("aaa")
	boolTrue := true
	ingressClass := "someClass"
	colalpha1 := &OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "otel",
			Namespace:   "observability",
			Labels:      map[string]string{"foo": "bar"},
			Annotations: map[string]string{"bax": "foo"},
		},
		Spec: OpenTelemetryCollectorSpec{
			ManagementState: ManagementStateManaged,
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
			NodeSelector: map[string]string{"aaa": "ccc"},
			Args:         map[string]string{"foo": "bar"},
			Replicas:     &one,
			Autoscaler: &AutoscalerSpec{
				MinReplicas: &one,
				MaxReplicas: &one,
				Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
					ScaleUp: &autoscalingv2.HPAScalingRules{
						Policies: []autoscalingv2.HPAScalingPolicy{
							{
								Type:          "aaa",
								Value:         2,
								PeriodSeconds: 4,
							},
						},
					},
				},
				Metrics: []MetricSpec{
					{
						Type: autoscalingv2.ContainerResourceMetricSourceType,
						Pods: &autoscalingv2.PodsMetricSource{
							Metric: autoscalingv2.MetricIdentifier{
								Name: "rrrrt",
							},
						},
					},
				},
				TargetCPUUtilization:    &one,
				TargetMemoryUtilization: &one,
			},
			PodDisruptionBudget: &PodDisruptionBudgetSpec{
				MinAvailable:   &intstrAAA,
				MaxUnavailable: &intstrAAA,
			},
			SecurityContext: &v1.SecurityContext{
				RunAsUser: &two,
			},
			PodSecurityContext: &v1.PodSecurityContext{
				RunAsNonRoot: &boolTrue,
			},
			PodAnnotations:  map[string]string{"foo": "bar"},
			TargetAllocator: createTA(),
			Mode:            ModeDeployment,
			ServiceAccount:  "foo",
			Image:           "baz/bar:1.0",
			UpgradeStrategy: UpgradeStrategyAutomatic,
			ImagePullPolicy: v1.PullAlways,
			Config:          collectorCfg,
			VolumeMounts: []v1.VolumeMount{
				{
					Name: "aaa",
				},
			},
			Ports: []PortsSpec{{
				ServicePort: v1.ServicePort{
					Name: "otlp",
				},
				HostPort: 4317,
			}},
			Env: []v1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
					ValueFrom: &v1.EnvVarSource{
						ResourceFieldRef: &v1.ResourceFieldSelector{
							ContainerName: "bbb",
							Resource:      "aaa",
							Divisor:       resource.Quantity{},
						},
					},
				},
			},
			EnvFrom: []v1.EnvFromSource{
				{
					Prefix: "aa",
					ConfigMapRef: &v1.ConfigMapEnvSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "bbb",
						},
					},
				},
			},
			VolumeClaimTemplates: []v1.PersistentVolumeClaim{
				{
					TypeMeta:   metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{},
					Spec: v1.PersistentVolumeClaimSpec{
						VolumeName: "aaaa",
					},
				},
			},
			Tolerations: []v1.Toleration{
				{
					Key:      "11",
					Operator: "33",
					Value:    "44",
					Effect:   "55",
				},
			},
			Volumes: []v1.Volume{
				{
					Name:         "cfg",
					VolumeSource: v1.VolumeSource{},
				},
			},
			Ingress: Ingress{
				Type:        IngressTypeRoute,
				RuleType:    IngressRuleTypePath,
				Hostname:    "foo.com",
				Annotations: map[string]string{"aa": "bb"},
				TLS: []networkingv1.IngressTLS{
					{
						Hosts:      []string{"foo"},
						SecretName: "bar",
					},
				},
				IngressClassName: &ingressClass,
				Route: OpenShiftRoute{
					Termination: TLSRouteTerminationTypeEdge,
				},
			},
			HostNetwork:           true,
			ShareProcessNamespace: true,
			PriorityClassName:     "foobar",
			Affinity: &v1.Affinity{
				NodeAffinity: &v1.NodeAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []v1.PreferredSchedulingTerm{{
						Weight: 444,
					}},
				},
			},
			Lifecycle: &v1.Lifecycle{
				PostStart: &v1.LifecycleHandler{
					Exec: &v1.ExecAction{
						Command: []string{"/bin"},
					},
				},
			},
			TerminationGracePeriodSeconds: &two,
			LivenessProbe: &Probe{
				PeriodSeconds: &one,
			},
			InitContainers: []v1.Container{
				{
					Name: "init",
				},
			},
			AdditionalContainers: []v1.Container{
				{
					Name: "some",
				},
			},
			Observability: ObservabilitySpec{
				Metrics: MetricsConfigSpec{
					EnableMetrics:                true,
					DisablePrometheusAnnotations: true,
				},
			},
			TopologySpreadConstraints: []v1.TopologySpreadConstraint{
				{
					TopologyKey: "key",
				},
			},
			ConfigMaps: []ConfigMapsSpec{
				{
					Name:      "aaa",
					MountPath: "bbb",
				},
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
			},
			DeploymentUpdateStrategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
		},
		Status: OpenTelemetryCollectorStatus{
			Scale: ScaleSubresourceStatus{
				Selector:       "bar",
				Replicas:       1,
				StatusReplicas: "foo",
			},
			Version: "1.0",
			Image:   "foo/bar:1.0",
		},
	}

	colbeta1, err := tov1beta1(*colalpha1)
	require.NoError(t, err)
	colalpha1Converted, err := tov1alpha1(colbeta1)
	require.NoError(t, err)

	assert.YAMLEq(t, colalpha1.Spec.Config, colalpha1Converted.Spec.Config)

	// empty the config to enable assertion on the entire objects
	colalpha1.Spec.Config = ""
	colalpha1Converted.Spec.Config = ""
	assert.Equal(t, colalpha1, colalpha1Converted)
}

func Test_tov1beta1AndBack_prometheus_selectors(t *testing.T) {
	t.Run("nil-selectors", func(t *testing.T) {
		colalpha1 := OpenTelemetryCollector{
			Spec: OpenTelemetryCollectorSpec{
				TargetAllocator: OpenTelemetryTargetAllocator{
					PrometheusCR: OpenTelemetryTargetAllocatorPrometheusCR{
						// nil or empty map means select everything
						PodMonitorSelector:     nil,
						ServiceMonitorSelector: nil,
					},
				},
			},
		}

		colbeta1 := v1beta1.OpenTelemetryCollector{}
		err := colalpha1.ConvertTo(&colbeta1)
		require.NoError(t, err)

		// nil LabelSelector means select nothing
		// empty LabelSelector mean select everything
		assert.NotNil(t, colbeta1.Spec.TargetAllocator.PrometheusCR.PodMonitorSelector)
		assert.NotNil(t, colbeta1.Spec.TargetAllocator.PrometheusCR.ServiceMonitorSelector)
		assert.Equal(t, 0, len(colalpha1.Spec.TargetAllocator.PrometheusCR.PodMonitorSelector))
		assert.Equal(t, 0, len(colalpha1.Spec.TargetAllocator.PrometheusCR.ServiceMonitorSelector))

		err = colalpha1.ConvertFrom(&colbeta1)
		require.NoError(t, err)
		assert.Nil(t, colalpha1.Spec.TargetAllocator.PrometheusCR.PodMonitorSelector)
		assert.Nil(t, colalpha1.Spec.TargetAllocator.PrometheusCR.ServiceMonitorSelector)
	})
	t.Run("empty-selectors", func(t *testing.T) {
		colalpha1 := OpenTelemetryCollector{
			Spec: OpenTelemetryCollectorSpec{
				TargetAllocator: OpenTelemetryTargetAllocator{
					PrometheusCR: OpenTelemetryTargetAllocatorPrometheusCR{
						// nil or empty map means select everything
						PodMonitorSelector:     map[string]string{},
						ServiceMonitorSelector: map[string]string{},
					},
				},
			},
		}

		colbeta1 := v1beta1.OpenTelemetryCollector{}
		err := colalpha1.ConvertTo(&colbeta1)
		require.NoError(t, err)

		// nil LabelSelector means select nothing
		// empty LabelSelector mean select everything
		assert.NotNil(t, colbeta1.Spec.TargetAllocator.PrometheusCR.PodMonitorSelector)
		assert.NotNil(t, colbeta1.Spec.TargetAllocator.PrometheusCR.ServiceMonitorSelector)

		err = colalpha1.ConvertFrom(&colbeta1)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{}, colalpha1.Spec.TargetAllocator.PrometheusCR.PodMonitorSelector)
		assert.Equal(t, map[string]string{}, colalpha1.Spec.TargetAllocator.PrometheusCR.ServiceMonitorSelector)
	})
}

func Test_tov1beta1AndBack_deprecated_replicas(t *testing.T) {
	one := int32(1)
	two := int32(2)
	colalpha1 := OpenTelemetryCollector{
		Spec: OpenTelemetryCollectorSpec{
			MinReplicas: &one,
			MaxReplicas: &two,
		},
	}

	colbeta1 := v1beta1.OpenTelemetryCollector{}
	err := colalpha1.ConvertTo(&colbeta1)
	require.NoError(t, err)

	assert.Equal(t, one, *colbeta1.Spec.Autoscaler.MinReplicas)
	assert.Equal(t, two, *colbeta1.Spec.Autoscaler.MaxReplicas)

	err = colalpha1.ConvertFrom(&colbeta1)
	require.NoError(t, err)
	assert.Nil(t, colalpha1.Spec.MinReplicas)
	assert.Nil(t, colalpha1.Spec.MaxReplicas)
	assert.Equal(t, one, *colalpha1.Spec.Autoscaler.MinReplicas)
	assert.Equal(t, two, *colalpha1.Spec.Autoscaler.MaxReplicas)
}

func createTA() OpenTelemetryTargetAllocator {
	replicas := int32(2)
	runAsNonRoot := true
	privileged := true
	runAsUser := int64(1337)
	runasGroup := int64(1338)
	return OpenTelemetryTargetAllocator{
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
		AllocationStrategy: OpenTelemetryTargetAllocatorAllocationStrategyConsistentHashing,
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
		PrometheusCR: OpenTelemetryTargetAllocatorPrometheusCR{
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
		Observability: ObservabilitySpec{
			Metrics: MetricsConfigSpec{
				EnableMetrics: true,
			},
		},
		PodDisruptionBudget: &PodDisruptionBudgetSpec{
			MaxUnavailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			},
		},
	}
}

func TestConvertTo(t *testing.T) {
	col := OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "otel",
		},
		Spec: OpenTelemetryCollectorSpec{
			ServiceAccount: "otelcol",
		},
		Status: OpenTelemetryCollectorStatus{
			Image: "otel/col",
		},
	}
	colbeta1 := v1beta1.OpenTelemetryCollector{}
	err := col.ConvertTo(&colbeta1)
	require.NoError(t, err)
	assert.Equal(t, v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "otel",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				ServiceAccount: "otelcol",
			},
			TargetAllocator: v1beta1.TargetAllocatorEmbedded{
				PrometheusCR: v1beta1.TargetAllocatorPrometheusCR{
					PodMonitorSelector:     &metav1.LabelSelector{},
					ServiceMonitorSelector: &metav1.LabelSelector{},
				},
			},
		},
		Status: v1beta1.OpenTelemetryCollectorStatus{
			Image: "otel/col",
		},
	}, colbeta1)
}

func TestConvertFrom(t *testing.T) {
	colbeta1 := v1beta1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "otel",
		},
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				ServiceAccount: "otelcol",
			},
		},
		Status: v1beta1.OpenTelemetryCollectorStatus{
			Image: "otel/col",
		},
	}
	col := OpenTelemetryCollector{}
	err := col.ConvertFrom(&colbeta1)
	require.NoError(t, err)
	// set config to empty. The v1beta1 marshals config with empty receivers, exporters..
	col.Spec.Config = ""
	assert.Equal(t, OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "otel",
		},
		Spec: OpenTelemetryCollectorSpec{
			ServiceAccount: "otelcol",
		},
		Status: OpenTelemetryCollectorStatus{
			Image: "otel/col",
		},
	}, col)
}
