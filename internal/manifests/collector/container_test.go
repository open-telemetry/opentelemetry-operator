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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	. "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
)

var logger = logf.Log.WithName("unit-tests")

var metricContainerPort = corev1.ContainerPort{
	Name:          "metrics",
	ContainerPort: 8888,
	Protocol:      corev1.ProtocolTCP,
}

func TestContainerNewDefault(t *testing.T) {
	// prepare
	var defaultConfig = `receivers:
		otlp:
			protocols:
			http:
			grpc:
	exporters:
		debug:
	service:
		pipelines:
			metrics:
				receivers: [otlp]
				exporters: [debug]`

	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "metrics",
					Port:     8888,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Config: defaultConfig,
		},
	}
	cfg := config.New(config.WithCollectorImage("default-image"))

	// test
	c := Container(cfg, logger, otelcol, true)

	// verify
	assert.Equal(t, "default-image", c.Image)
	assert.Equal(t, []corev1.ContainerPort{metricContainerPort}, c.Ports)
}

func TestContainerWithImageOverridden(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Image: "overridden-image",
		},
	}
	cfg := config.New(config.WithCollectorImage("default-image"))

	// test
	c := Container(cfg, logger, otelcol, true)

	// verify
	assert.Equal(t, "overridden-image", c.Image)
}

func TestContainerPorts(t *testing.T) {
	var goodConfig = `receivers:
  examplereceiver:
    endpoint: "0.0.0.0:12345"
exporters:
  debug:
service:
  pipelines:
    metrics:
      receivers: [examplereceiver]
      exporters: [debug]`

	tests := []struct {
		description   string
		specConfig    string
		specPorts     []corev1.ServicePort
		expectedPorts []corev1.ContainerPort
	}{
		{
			description:   "bad spec config",
			specConfig:    "🦄",
			specPorts:     nil,
			expectedPorts: []corev1.ContainerPort{},
		},
		{
			description: "couldn't build ports from spec config",
			specConfig:  "",
			specPorts: []corev1.ServicePort{
				{
					Name:     "metrics",
					Port:     8888,
					Protocol: corev1.ProtocolTCP,
				},
			},
			expectedPorts: []corev1.ContainerPort{metricContainerPort},
		},
		{
			description: "ports in spec Config",
			specConfig:  goodConfig,
			specPorts:   nil,
			expectedPorts: []corev1.ContainerPort{
				{
					Name:          "examplereceiver",
					ContainerPort: 12345,
				},
				metricContainerPort,
			},
		},
		{
			description: "ports in spec ContainerPorts",
			specPorts: []corev1.ServicePort{
				{
					Name:     "metrics",
					Port:     8888,
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name: "testport1",
					Port: 12345,
				},
			},
			expectedPorts: []corev1.ContainerPort{
				metricContainerPort,
				{
					Name:          "testport1",
					ContainerPort: 12345,
				},
			},
		},
		{
			description: "ports in spec Config and ContainerPorts",
			specConfig:  goodConfig,
			specPorts: []corev1.ServicePort{
				{
					Name: "testport1",
					Port: 12345,
				},
				{
					Name:     "testport2",
					Port:     54321,
					Protocol: corev1.ProtocolUDP,
				},
			},
			expectedPorts: []corev1.ContainerPort{
				{
					Name:          "examplereceiver",
					ContainerPort: 12345,
				},
				metricContainerPort,
				{
					Name:          "testport1",
					ContainerPort: 12345,
				},
				{
					Name:          "testport2",
					ContainerPort: 54321,
					Protocol:      corev1.ProtocolUDP,
				},
			},
		},
		{
			description: "duplicate port name",
			specConfig:  goodConfig,
			specPorts: []corev1.ServicePort{
				{
					Name: "testport1",
					Port: 12345,
				},
				{
					Name: "testport1",
					Port: 11111,
				},
			},
			expectedPorts: []corev1.ContainerPort{
				{
					Name:          "examplereceiver",
					ContainerPort: 12345,
				},
				metricContainerPort,
				{
					Name:          "testport1",
					ContainerPort: 11111,
				},
			},
		},
		{
			description: "prometheus exporter",
			specConfig: `exporters:
    prometheus:
        endpoint: "0.0.0.0:9090"
	debug:
service:
    pipelines:
        metrics:
			receivers: [otlp]
            exporters: [prometheus, debug]
`,
			specPorts: []corev1.ServicePort{
				{
					Name:     "metrics",
					Port:     8888,
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name: "prometheus",
					Port: 9090,
				},
			},
			expectedPorts: []corev1.ContainerPort{
				{
					Name:          "metrics",
					ContainerPort: 8888,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          "prometheus",
					ContainerPort: 9090,
				},
			},
		},
		{
			description: "multiple prometheus exporters",
			specConfig: `exporters:
    prometheus/prod:
        endpoint: "0.0.0.0:9090"
    prometheus/dev:
        endpoint: "0.0.0.0:9091"
	debug:
service:
    pipelines:
        metrics:
            exporters: [prometheus/prod, prometheus/dev, debug]
`,
			specPorts: []corev1.ServicePort{
				{
					Name:     "metrics",
					Port:     8888,
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name: "prometheus-dev",
					Port: 9091,
				},
				{
					Name: "prometheus-prod",
					Port: 9090,
				},
			},
			expectedPorts: []corev1.ContainerPort{
				metricContainerPort,
				{
					Name:          "prometheus-dev",
					ContainerPort: 9091,
				},
				{
					Name:          "prometheus-prod",
					ContainerPort: 9090,
				},
			},
		},
		{
			description: "prometheus RW exporter",
			specConfig: `exporters:
    prometheusremotewrite/prometheus:
        endpoint: http://prometheus-server.monitoring/api/v1/write`,
			specPorts: []corev1.ServicePort{
				{
					Name:     "metrics",
					Port:     8888,
					Protocol: corev1.ProtocolTCP,
				},
			},
			expectedPorts: []corev1.ContainerPort{metricContainerPort},
		},
		{
			description: "multiple prometheus exporters and prometheus RW exporter",
			specConfig: `exporters:
    prometheus/prod:
        endpoint: "0.0.0.0:9090"
    prometheus/dev:
        endpoint: "0.0.0.0:9091"
    prometheusremotewrite/prometheus:
        endpoint: http://prometheus-server.monitoring/api/v1/write
	debug:
service:
    pipelines:
        metrics:
            exporters: [prometheus/prod, prometheus/dev, prometheusremotewrite/prometheus, debug]`,
			specPorts: []corev1.ServicePort{
				{
					Name:     "metrics",
					Port:     8888,
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name: "prometheus-dev",
					Port: 9091,
				},
				{
					Name: "prometheus-prod",
					Port: 9090,
				},
			},
			expectedPorts: []corev1.ContainerPort{
				metricContainerPort,
				{
					Name:          "prometheus-dev",
					ContainerPort: 9091,
				},
				{
					Name:          "prometheus-prod",
					ContainerPort: 9090,
				},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.description, func(t *testing.T) {
			// prepare
			otelcol := v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Config: testCase.specConfig,
					Ports:  testCase.specPorts,
				},
			}

			cfg := config.New(config.WithCollectorImage("default-image"))

			// test
			c := Container(cfg, logger, otelcol, true)
			// verify
			assert.ElementsMatch(t, testCase.expectedPorts, c.Ports, testCase.description)
		})
	}
}

func TestContainerConfigFlagIsIgnored(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Args: map[string]string{
				"key":    "value",
				"config": "/some-custom-file.yaml",
			},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol, true)

	// verify
	assert.Len(t, c.Args, 2)
	assert.Contains(t, c.Args, "--key=value")
	assert.NotContains(t, c.Args, "--config=/some-custom-file.yaml")
}

func TestContainerCustomVolumes(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			VolumeMounts: []corev1.VolumeMount{{
				Name: "custom-volume-mount",
			}},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol, true)

	// verify
	assert.Len(t, c.VolumeMounts, 2)
	assert.Equal(t, "custom-volume-mount", c.VolumeMounts[1].Name)
}

func TestContainerCustomConfigMapsVolumes(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			ConfigMaps: []v1alpha1.ConfigMapsSpec{{
				Name:      "test",
				MountPath: "/",
			}, {
				Name:      "test2",
				MountPath: "/dir",
			}},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol, true)

	// verify
	assert.Len(t, c.VolumeMounts, 3)
	assert.Equal(t, "configmap-test", c.VolumeMounts[1].Name)
	assert.Equal(t, "/var/conf/configmap-test", c.VolumeMounts[1].MountPath)
	assert.Equal(t, "configmap-test2", c.VolumeMounts[2].Name)
	assert.Equal(t, "/var/conf/dir/configmap-test2", c.VolumeMounts[2].MountPath)
}

func TestContainerCustomSecurityContext(t *testing.T) {
	// default config without security context
	c1 := Container(config.New(), logger, v1alpha1.OpenTelemetryCollector{Spec: v1alpha1.OpenTelemetryCollectorSpec{}}, true)

	// verify
	assert.Nil(t, c1.SecurityContext)

	// prepare
	isPrivileged := true
	uid := int64(1234)

	// test
	c2 := Container(config.New(), logger, v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			SecurityContext: &corev1.SecurityContext{
				Privileged: &isPrivileged,
				RunAsUser:  &uid,
			},
		},
	}, true)

	// verify
	assert.NotNil(t, c2.SecurityContext)
	assert.True(t, *c2.SecurityContext.Privileged)
	assert.Equal(t, *c2.SecurityContext.RunAsUser, uid)
}

func TestContainerEnvVarsOverridden(t *testing.T) {
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Env: []corev1.EnvVar{
				{
					Name:  "foo",
					Value: "bar",
				},
			},
		},
	}

	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol, true)

	// verify
	assert.Len(t, c.Env, 2)
	assert.Equal(t, "foo", c.Env[0].Name)
	assert.Equal(t, "bar", c.Env[0].Value)
}

func TestContainerDefaultEnvVars(t *testing.T) {
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{},
	}

	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol, true)

	// verify
	assert.Len(t, c.Env, 1)
	assert.Equal(t, c.Env[0].Name, "POD_NAME")
}

func TestContainerProxyEnvVars(t *testing.T) {
	err := os.Setenv("NO_PROXY", "localhost")
	require.NoError(t, err)
	defer os.Unsetenv("NO_PROXY")
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{},
	}

	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol, true)

	// verify
	require.Len(t, c.Env, 3)
	assert.Equal(t, "POD_NAME", c.Env[0].Name)
	assert.Equal(t, corev1.EnvVar{Name: "NO_PROXY", Value: "localhost"}, c.Env[1])
	assert.Equal(t, corev1.EnvVar{Name: "no_proxy", Value: "localhost"}, c.Env[2])
}

func TestContainerResourceRequirements(t *testing.T) {
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("128M"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("256M"),
				},
			},
		},
	}

	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol, true)

	// verify
	assert.Equal(t, resource.MustParse("100m"), *c.Resources.Limits.Cpu())
	assert.Equal(t, resource.MustParse("128M"), *c.Resources.Limits.Memory())
	assert.Equal(t, resource.MustParse("200m"), *c.Resources.Requests.Cpu())
	assert.Equal(t, resource.MustParse("256M"), *c.Resources.Requests.Memory())
}

func TestContainerDefaultResourceRequirements(t *testing.T) {
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{},
	}

	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol, true)

	// verify
	assert.Empty(t, c.Resources)
}

func TestContainerArgs(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Args: map[string]string{
				"metrics-level": "detailed",
				"log-level":     "debug",
			},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol, true)

	// verify
	assert.Contains(t, c.Args, "--metrics-level=detailed")
	assert.Contains(t, c.Args, "--log-level=debug")
}

func TestContainerOrderedArgs(t *testing.T) {
	// prepare a scenario where the debug level and a feature gate has been enabled
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Args: map[string]string{
				"log-level":     "debug",
				"feature-gates": "+random-feature",
			},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol, true)

	// verify that the first args is (always) the config, and the remaining args are ordered alphabetically
	// by the key
	assert.Equal(t, "--config=/conf/collector.yaml", c.Args[0])
	assert.Equal(t, "--feature-gates=+random-feature", c.Args[1])
	assert.Equal(t, "--log-level=debug", c.Args[2])
}

func TestContainerImagePullPolicy(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			ImagePullPolicy: corev1.PullIfNotPresent,
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol, true)

	// verify
	assert.Equal(t, c.ImagePullPolicy, corev1.PullIfNotPresent)
}

func TestContainerEnvFrom(t *testing.T) {
	//prepare
	envFrom1 := corev1.EnvFromSource{
		SecretRef: &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: "env-as-secret",
			},
		},
	}
	envFrom2 := corev1.EnvFromSource{
		ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: "env-as-configmap",
			},
		},
	}
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			EnvFrom: []corev1.EnvFromSource{
				envFrom1,
				envFrom2,
			},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol, true)

	// verify
	assert.Contains(t, c.EnvFrom, envFrom1)
	assert.Contains(t, c.EnvFrom, envFrom2)
}

func TestContainerProbe(t *testing.T) {
	// prepare
	initialDelaySeconds := int32(10)
	timeoutSeconds := int32(11)
	periodSeconds := int32(12)
	successThreshold := int32(13)
	failureThreshold := int32(14)
	terminationGracePeriodSeconds := int64(15)
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Config: `extensions:
  health_check:
service:
  extensions: [health_check]`,
			LivenessProbe: &v1alpha1.Probe{
				InitialDelaySeconds:           &initialDelaySeconds,
				TimeoutSeconds:                &timeoutSeconds,
				PeriodSeconds:                 &periodSeconds,
				SuccessThreshold:              &successThreshold,
				FailureThreshold:              &failureThreshold,
				TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
			},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol, true)

	// verify
	assert.Equal(t, "/", c.LivenessProbe.HTTPGet.Path)
	assert.Equal(t, int32(13133), c.LivenessProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, "", c.LivenessProbe.HTTPGet.Host)

	assert.Equal(t, initialDelaySeconds, c.LivenessProbe.InitialDelaySeconds)
	assert.Equal(t, timeoutSeconds, c.LivenessProbe.TimeoutSeconds)
	assert.Equal(t, periodSeconds, c.LivenessProbe.PeriodSeconds)
	assert.Equal(t, successThreshold, c.LivenessProbe.SuccessThreshold)
	assert.Equal(t, failureThreshold, c.LivenessProbe.FailureThreshold)
	assert.Equal(t, terminationGracePeriodSeconds, *c.LivenessProbe.TerminationGracePeriodSeconds)
}

func TestContainerProbeEmptyConfig(t *testing.T) {
	// prepare

	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Config: `extensions:
  health_check:
service:
  extensions: [health_check]`,
			LivenessProbe: &v1alpha1.Probe{},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol, true)

	// verify
	assert.Equal(t, "/", c.LivenessProbe.HTTPGet.Path)
	assert.Equal(t, int32(13133), c.LivenessProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, "", c.LivenessProbe.HTTPGet.Host)
}

func TestContainerProbeNoConfig(t *testing.T) {
	// prepare

	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Config: `extensions:
  health_check:
service:
  extensions: [health_check]`,
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol, true)

	// verify
	assert.Equal(t, "/", c.LivenessProbe.HTTPGet.Path)
	assert.Equal(t, int32(13133), c.LivenessProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, "", c.LivenessProbe.HTTPGet.Host)
}

func TestContainerLifecycle(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpenTelemetryCollector{
		Spec: v1alpha1.OpenTelemetryCollectorSpec{
			Lifecycle: &corev1.Lifecycle{
				PostStart: &corev1.LifecycleHandler{
					Exec: &corev1.ExecAction{Command: []string{"sh", "sleep 100"}},
				},
				PreStop: &corev1.LifecycleHandler{
					Exec: &corev1.ExecAction{Command: []string{"sh", "sleep 300"}},
				},
			},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, otelcol, true)

	expectedLifecycleHooks := corev1.Lifecycle{
		PostStart: &corev1.LifecycleHandler{
			Exec: &corev1.ExecAction{Command: []string{"sh", "sleep 100"}},
		},
		PreStop: &corev1.LifecycleHandler{
			Exec: &corev1.ExecAction{Command: []string{"sh", "sleep 300"}},
		},
	}

	// verify
	assert.Equal(t, expectedLifecycleHooks, *c.Lifecycle)
}
