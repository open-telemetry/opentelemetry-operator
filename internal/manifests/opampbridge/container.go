// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opampbridge

import (
	"slices"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

// Container builds a container for the given OpAMPBridge.
func Container(cfg config.Config, _ logr.Logger, opampBridge v1alpha1.OpAMPBridge) corev1.Container {
	image := opampBridge.Spec.Image
	if image == "" {
		image = cfg.OperatorOpAMPBridgeImage
	}

	volumeMounts := []corev1.VolumeMount{{
		Name:      naming.OpAMPBridgeConfigMapVolume(),
		MountPath: "/conf",
	}}

	if len(opampBridge.Spec.VolumeMounts) > 0 {
		volumeMounts = append(volumeMounts, opampBridge.Spec.VolumeMounts...)
	}

	envVars := slices.Clone(opampBridge.Spec.Env)
	if opampBridge.Spec.Env == nil {
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
					ContainerName: naming.OpAMPBridgeContainer(),
				},
			},
		},
			corev1.EnvVar{
				Name: "GOMAXPROCS",
				ValueFrom: &corev1.EnvVarSource{
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						Resource:      "limits.cpu",
						ContainerName: naming.OpAMPBridgeContainer(),
					},
				},
			},
		)
	}

	envVars = append(envVars, cfg.ProxyEnvVars...)

	return corev1.Container{
		Name:            naming.OpAMPBridgeContainer(),
		Image:           image,
		ImagePullPolicy: opampBridge.Spec.ImagePullPolicy,
		Env:             envVars,
		VolumeMounts:    volumeMounts,
		EnvFrom:         opampBridge.Spec.EnvFrom,
		Resources:       opampBridge.Spec.Resources,
		SecurityContext: opampBridge.Spec.SecurityContext,
		Ports: []corev1.ContainerPort{
			{
				Name:          "opamp",
				ContainerPort: 8080,
				Protocol:      corev1.ProtocolTCP,
			},
			{
				Name:          "healthz",
				ContainerPort: 8081,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromString("healthz"),
				},
			},
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromString("healthz"),
				},
			},
		},
	}
}
