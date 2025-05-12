// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	colfg "go.opentelemetry.io/collector/featuregate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

var logger = logf.Log.WithName("unit-tests")

func TestContainerNewDefault(t *testing.T) {
	// prepare
	targetAllocator := v1alpha1.TargetAllocator{}
	cfg := config.New(config.WithTargetAllocatorImage("default-image"))

	// test
	c := Container(cfg, logger, targetAllocator)

	// verify
	assert.Equal(t, "default-image", c.Image)
}

func TestContainerWithImageOverridden(t *testing.T) {
	// prepare
	targetAllocator := v1alpha1.TargetAllocator{
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Image: "overridden-image",
			},
		},
	}
	cfg := config.New(config.WithTargetAllocatorImage("default-image"))

	// test
	c := Container(cfg, logger, targetAllocator)

	// verify
	assert.Equal(t, "overridden-image", c.Image)
}

func TestContainerDefaultPorts(t *testing.T) {
	// prepare
	targetAllocator := v1alpha1.TargetAllocator{}
	cfg := config.New()

	// test
	c := Container(cfg, logger, targetAllocator)

	// verify
	assert.Len(t, c.Ports, 1)
	assert.Equal(t, "http", c.Ports[0].Name)
	assert.Equal(t, int32(8080), c.Ports[0].ContainerPort)
}

func TestContainerDefaultVolumes(t *testing.T) {
	// prepare
	targetAllocator := v1alpha1.TargetAllocator{}
	cfg := config.New()

	// test
	c := Container(cfg, logger, targetAllocator)

	// verify
	assert.Len(t, c.VolumeMounts, 1)
	assert.Equal(t, naming.TAConfigMapVolume(), c.VolumeMounts[0].Name)
}

func TestContainerResourceRequirements(t *testing.T) {
	targetAllocator := v1alpha1.TargetAllocator{
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
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
		},
	}

	cfg := config.New()
	resourceTest := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128M"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("200m"),
			corev1.ResourceMemory: resource.MustParse("256M"),
		},
	}
	// test
	c := Container(cfg, logger, targetAllocator)
	resourcesValues := c.Resources

	// verify
	assert.Equal(t, resourceTest, resourcesValues)
}

func TestContainerHasEnvVars(t *testing.T) {
	// prepare
	targetAllocator := v1alpha1.TargetAllocator{
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Env: []corev1.EnvVar{
					{
						Name:  "TEST_ENV",
						Value: "test",
					},
				},
			},
		},
	}
	cfg := config.New(config.WithTargetAllocatorImage("default-image"))

	expected := corev1.Container{
		Name:  "ta-container",
		Image: "default-image",
		Env: []corev1.EnvVar{
			{
				Name:  "TEST_ENV",
				Value: "test",
			},
			{
				Name:  "OTELCOL_NAMESPACE",
				Value: "",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "",
						FieldPath:  "metadata.namespace",
					},
					ResourceFieldRef: nil,
					ConfigMapKeyRef:  nil,
					SecretKeyRef:     nil,
				},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:             "ta-internal",
				ReadOnly:         false,
				MountPath:        "/conf",
				SubPath:          "",
				MountPropagation: nil,
				SubPathExpr:      "",
			},
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: 8080,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/readyz",
					Port: intstr.FromInt(8080),
				},
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/livez",
					Port: intstr.FromInt(8080),
				},
			},
		},
	}

	// test
	c := Container(cfg, logger, targetAllocator)

	// verify
	assert.Equal(t, expected, c)
}

func TestContainerHasProxyEnvVars(t *testing.T) {
	err := os.Setenv("NO_PROXY", "localhost")
	require.NoError(t, err)
	defer os.Unsetenv("NO_PROXY")

	// prepare
	targetAllocator := v1alpha1.TargetAllocator{
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Env: []corev1.EnvVar{
					{
						Name:  "TEST_ENV",
						Value: "test",
					},
				},
			},
		},
	}
	cfg := config.New(config.WithTargetAllocatorImage("default-image"))

	// test
	c := Container(cfg, logger, targetAllocator)

	// verify
	require.Len(t, c.Env, 4)
	assert.Equal(t, corev1.EnvVar{Name: "NO_PROXY", Value: "localhost"}, c.Env[2])
	assert.Equal(t, corev1.EnvVar{Name: "no_proxy", Value: "localhost"}, c.Env[3])
}

func TestContainerDoesNotOverrideEnvVars(t *testing.T) {
	// prepare
	targetAllocator := v1alpha1.TargetAllocator{
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Env: []corev1.EnvVar{
					{
						Name:  "OTELCOL_NAMESPACE",
						Value: "test",
					},
				},
			},
		},
	}
	cfg := config.New(config.WithTargetAllocatorImage("default-image"))

	expected := corev1.Container{
		Name:  "ta-container",
		Image: "default-image",
		Env: []corev1.EnvVar{
			{
				Name:  "OTELCOL_NAMESPACE",
				Value: "test",
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:             "ta-internal",
				ReadOnly:         false,
				MountPath:        "/conf",
				SubPath:          "",
				MountPropagation: nil,
				SubPathExpr:      "",
			},
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: 8080,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/readyz",
					Port: intstr.FromInt(8080),
				},
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/livez",
					Port: intstr.FromInt(8080),
				},
			},
		},
	}

	// test
	c := Container(cfg, logger, targetAllocator)

	// verify
	assert.Equal(t, expected, c)
}
func TestReadinessProbe(t *testing.T) {
	targetAllocator := v1alpha1.TargetAllocator{}
	cfg := config.New()
	expected := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/readyz",
				Port: intstr.FromInt(8080),
			},
		},
	}

	// test
	c := Container(cfg, logger, targetAllocator)

	// verify
	assert.Equal(t, expected, c.ReadinessProbe)
}
func TestLivenessProbe(t *testing.T) {
	// prepare
	targetAllocator := v1alpha1.TargetAllocator{}
	cfg := config.New()
	expected := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/livez",
				Port: intstr.FromInt(8080),
			},
		},
	}

	// test
	c := Container(cfg, logger, targetAllocator)

	// verify
	assert.Equal(t, expected, c.LivenessProbe)
}

func TestSecurityContext(t *testing.T) {
	runAsNonRoot := true
	securityContext := &corev1.SecurityContext{
		RunAsNonRoot: &runAsNonRoot,
	}
	// prepare
	targetAllocator := v1alpha1.TargetAllocator{
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				SecurityContext: securityContext,
			},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, targetAllocator)

	// verify
	assert.Equal(t, securityContext, c.SecurityContext)
}

func TestArgs(t *testing.T) {
	// prepare
	targetAllocator := v1alpha1.TargetAllocator{
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Args: map[string]string{
					"key":  "value",
					"akey": "avalue",
				},
			},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, targetAllocator)

	// verify
	expected := []string{"--akey=avalue", "--key=value"}
	assert.Equal(t, expected, c.Args)
}

func TestContainerWithCertManagerAvailable(t *testing.T) {
	// prepare
	targetAllocator := v1alpha1.TargetAllocator{}

	flgs := featuregate.Flags(colfg.GlobalRegistry())
	err := flgs.Parse([]string{"--feature-gates=operator.targetallocator.mtls"})
	require.NoError(t, err)

	cfg := config.New(config.WithCertManagerAvailability(certmanager.Available))

	// test
	c := Container(cfg, logger, targetAllocator)

	// verify
	assert.Equal(t, "http", c.Ports[0].Name)
	assert.Equal(t, int32(8080), c.Ports[0].ContainerPort)
	assert.Equal(t, "https", c.Ports[1].Name)
	assert.Equal(t, int32(8443), c.Ports[1].ContainerPort)

	assert.Contains(t, c.VolumeMounts, corev1.VolumeMount{
		Name:      naming.TAServerCertificate(""),
		MountPath: constants.TACollectorTLSDirPath,
	})
}

func TestContainerCustomVolumes(t *testing.T) {
	// prepare
	targetAllocator := v1alpha1.TargetAllocator{
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				VolumeMounts: []corev1.VolumeMount{{
					Name: "custom-volume-mount",
				}},
			},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, targetAllocator)

	// verify
	assert.Len(t, c.VolumeMounts, 2)
	assert.Equal(t, "custom-volume-mount", c.VolumeMounts[1].Name)
}

func TestContainerCustomPorts(t *testing.T) {
	// prepare
	targetAllocator := v1alpha1.TargetAllocator{
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Ports: []v1beta1.PortsSpec{
					{
						ServicePort: corev1.ServicePort{
							Name:     "testport1",
							Port:     12345,
							Protocol: corev1.ProtocolTCP,
						},
						HostPort: 54321,
					},
				},
			},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, targetAllocator)

	// verify
	assert.Len(t, c.Ports, 2)
	actual := c.Ports[1]
	expected := corev1.ContainerPort{
		Name:          "testport1",
		ContainerPort: 12345,
		Protocol:      corev1.ProtocolTCP,
		HostPort:      54321,
	}
	assert.Equal(t, expected, actual)
}

func TestContainerLifecycle(t *testing.T) {
	// prepare
	targetAllocator := v1alpha1.TargetAllocator{
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Lifecycle: &corev1.Lifecycle{
					PostStart: &corev1.LifecycleHandler{
						Exec: &corev1.ExecAction{Command: []string{"sh", "sleep 100"}},
					},
					PreStop: &corev1.LifecycleHandler{
						Exec: &corev1.ExecAction{Command: []string{"sh", "sleep 300"}},
					},
				},
			},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, targetAllocator)

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
	// prepare
	targetAllocator := v1alpha1.TargetAllocator{
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				EnvFrom: []corev1.EnvFromSource{
					envFrom1,
					envFrom2,
				},
			},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, targetAllocator)

	// verify
	assert.Contains(t, c.EnvFrom, envFrom1)
	assert.Contains(t, c.EnvFrom, envFrom2)
}

func TestContainerImagePullPolicy(t *testing.T) {
	// prepare
	targetAllocator := v1alpha1.TargetAllocator{
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				ImagePullPolicy: corev1.PullIfNotPresent,
			},
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, targetAllocator)

	// verify
	assert.Equal(t, c.ImagePullPolicy, corev1.PullIfNotPresent)
}
