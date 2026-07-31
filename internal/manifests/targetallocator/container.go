// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"fmt"
	"slices"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

// Container builds a container for the given TargetAllocator.
func Container(cfg config.Config, _ logr.Logger, ta v1alpha1.TargetAllocator) corev1.Container {
	image := ta.Spec.Image
	if image == "" {
		image = cfg.TargetAllocatorImage
	}

	ports := make([]corev1.ContainerPort, 0)
	ports = append(ports, corev1.ContainerPort{
		Name:          "http",
		ContainerPort: 8080,
		Protocol:      corev1.ProtocolTCP,
	})
	for _, p := range ta.Spec.Ports {
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
	volumeMounts = append(volumeMounts, ta.Spec.VolumeMounts...)

	envVars := slices.Clone(ta.Spec.Env)
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
	argsMap := ta.Spec.Args
	if argsMap == nil {
		argsMap = map[string]string{}
	}
	for k, v := range argsMap {
		args = append(args, fmt.Sprintf("--%s=%s", k, v))
	}
	slices.Sort(args)

	readinessProbe := ta.Spec.ReadinessProbe
	if readinessProbe == nil {
		readinessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/readyz",
					Port: intstr.FromInt(8080),
				},
			},
		}
	}

	livenessProbe := ta.Spec.LivenessProbe
	if livenessProbe == nil {
		livenessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/livez",
					Port: intstr.FromInt(8080),
				},
			},
		}
	}

	if manifestutils.IsTAMTLSEnabled(ta.Spec.Mtls) {
		ports = append(ports, corev1.ContainerPort{
			Name:          "https",
			ContainerPort: 8443,
			Protocol:      corev1.ProtocolTCP,
		})
		_, serverMounts := manifestutils.TAServerCertificateVolumes(&ta)
		volumeMounts = append(volumeMounts, serverMounts...)
	}

	envVars = append(envVars, cfg.ProxyEnvVars...)
	return corev1.Container{
		Name:            naming.TAContainer(),
		Image:           image,
		ImagePullPolicy: ta.Spec.ImagePullPolicy,
		Ports:           ports,
		VolumeMounts:    volumeMounts,
		Args:            args,
		Env:             envVars,
		EnvFrom:         ta.Spec.EnvFrom,
		Resources:       ta.Spec.Resources,
		SecurityContext: ta.Spec.SecurityContext,
		LivenessProbe:   livenessProbe,
		ReadinessProbe:  readinessProbe,
		Lifecycle:       ta.Spec.Lifecycle,
	}
}
