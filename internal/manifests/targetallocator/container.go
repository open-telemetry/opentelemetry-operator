// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"fmt"
	"sort"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/proxy"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

// Container builds a container for the given TargetAllocator.
func Container(cfg config.Config, logger logr.Logger, instance v1alpha1.TargetAllocator) corev1.Container {
	image := instance.Spec.Image
	if len(image) == 0 {
		image = cfg.TargetAllocatorImage()
	}

	ports := make([]corev1.ContainerPort, 0)
	ports = append(ports, corev1.ContainerPort{
		Name:          "http",
		ContainerPort: 8080,
		Protocol:      corev1.ProtocolTCP,
	})
	for _, p := range instance.Spec.Ports {
		ports = append(ports, corev1.ContainerPort{
			Name:          p.Name,
			ContainerPort: p.Port,
			Protocol:      p.Protocol,
			HostPort:      p.HostPort,
		})
	}

	volumeMounts := []corev1.VolumeMount{{
		Name:      naming.TAConfigMapVolume(),
		MountPath: "/conf",
	}}
	volumeMounts = append(volumeMounts, instance.Spec.VolumeMounts...)

	var envVars = instance.Spec.Env
	if envVars == nil {
		envVars = []corev1.EnvVar{}
	}

	idx := -1
	for i := range envVars {
		if envVars[i].Name == "OTELCOL_NAMESPACE" {
			idx = i
		}
	}
	if idx == -1 {
		envVars = append(envVars, corev1.EnvVar{
			Name: "OTELCOL_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		})
	}

	if featuregate.SetGolangFlags.IsEnabled() {
		envVars = append(envVars, corev1.EnvVar{
			Name: "GOMEMLIMIT",
			ValueFrom: &corev1.EnvVarSource{
				ResourceFieldRef: &corev1.ResourceFieldSelector{
					Resource:      "limits.memory",
					ContainerName: naming.TAContainer(),
				},
			},
		},
			corev1.EnvVar{
				Name: "GOMAXPROCS",
				ValueFrom: &corev1.EnvVarSource{
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						Resource:      "limits.cpu",
						ContainerName: naming.TAContainer(),
					},
				},
			},
		)
	}

	var args []string
	// ensure that the args are ordered when moved to container.Args, so the output doesn't depend on map iteration
	argsMap := instance.Spec.Args
	if argsMap == nil {
		argsMap = map[string]string{}
	}
	for k, v := range argsMap {
		args = append(args, fmt.Sprintf("--%s=%s", k, v))
	}
	sort.Strings(args)
	readinessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/readyz",
				Port: intstr.FromInt(8080),
			},
		},
	}
	livenessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/livez",
				Port: intstr.FromInt(8080),
			},
		},
	}

	if cfg.CertManagerAvailability() == certmanager.Available && featuregate.EnableTargetAllocatorMTLS.IsEnabled() {
		ports = append(ports, corev1.ContainerPort{
			Name:          "https",
			ContainerPort: 8443,
			Protocol:      corev1.ProtocolTCP,
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      naming.TAServerCertificate(instance.Name),
			MountPath: constants.TACollectorTLSDirPath,
		})
	}

	envVars = append(envVars, proxy.ReadProxyVarsFromEnv()...)
	return corev1.Container{
		Name:            naming.TAContainer(),
		Image:           image,
		ImagePullPolicy: instance.Spec.ImagePullPolicy,
		Ports:           ports,
		VolumeMounts:    volumeMounts,
		Args:            args,
		Env:             envVars,
		EnvFrom:         instance.Spec.EnvFrom,
		Resources:       instance.Spec.Resources,
		SecurityContext: instance.Spec.SecurityContext,
		LivenessProbe:   livenessProbe,
		ReadinessProbe:  readinessProbe,
		Lifecycle:       instance.Spec.Lifecycle,
	}
}
