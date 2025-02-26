// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opampbridge

import (
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/proxy"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

// Container builds a container for the given OpAMPBridge.
func Container(cfg config.Config, logger logr.Logger, opampBridge v1alpha1.OpAMPBridge) corev1.Container {
	image := opampBridge.Spec.Image
	if len(image) == 0 {
		image = cfg.OperatorOpAMPBridgeImage()
	}

	volumeMounts := []corev1.VolumeMount{{
		Name:      naming.OpAMPBridgeConfigMapVolume(),
		MountPath: "/conf",
	}}

	if len(opampBridge.Spec.VolumeMounts) > 0 {
		volumeMounts = append(volumeMounts, opampBridge.Spec.VolumeMounts...)
	}

	var envVars = opampBridge.Spec.Env
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

	envVars = append(envVars, proxy.ReadProxyVarsFromEnv()...)

	return corev1.Container{
		Name:            naming.OpAMPBridgeContainer(),
		Image:           image,
		ImagePullPolicy: opampBridge.Spec.ImagePullPolicy,
		Env:             envVars,
		VolumeMounts:    volumeMounts,
		EnvFrom:         opampBridge.Spec.EnvFrom,
		Resources:       opampBridge.Spec.Resources,
		SecurityContext: opampBridge.Spec.SecurityContext,
	}
}
