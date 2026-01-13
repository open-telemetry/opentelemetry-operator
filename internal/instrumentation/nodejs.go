// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	envNodeOptions          = "NODE_OPTIONS"
	nodeRequireArgument     = " --require /otel-auto-instrumentation-nodejs/autoinstrumentation.js"
	nodejsInitContainerName = initContainerName + "-nodejs"
	nodejsVolumeName        = volumeName + "-nodejs"
	nodejsInstrMountPath    = "/otel-auto-instrumentation-nodejs"
)

func injectNodeJSSDK(nodeJSSpec v1alpha1.NodeJS, pod corev1.Pod, container *corev1.Container, instSpec v1alpha1.InstrumentationSpec) (corev1.Pod, error) {
	volume := instrVolume(nodeJSSpec.VolumeClaimTemplate, nodejsVolumeName, nodeJSSpec.VolumeSizeLimit)

	err := validateContainerEnv(container.Env, envNodeOptions)
	if err != nil {
		return pod, err
	}

	// inject NodeJS instrumentation spec env vars.
	container.Env = appendIfNotSet(container.Env, nodeJSSpec.Env...)

	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      volume.Name,
		MountPath: nodejsInstrMountPath,
	})

	// We just inject Volumes and init containers for the first processed container
	if isInitContainerMissing(pod, nodejsInitContainerName) {
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:      nodejsInitContainerName,
			Image:     nodeJSSpec.Image,
			Command:   []string{"cp", "-r", "/autoinstrumentation/.", nodejsInstrMountPath},
			Resources: nodeJSSpec.Resources,
			VolumeMounts: []corev1.VolumeMount{{
				Name:      volume.Name,
				MountPath: nodejsInstrMountPath,
			}},
			ImagePullPolicy: instSpec.ImagePullPolicy,
		})
	}
	return pod, nil
}

func getDefaultNodeJSEnvVars(container *corev1.Container) []corev1.EnvVar {
	idx := getIndexOfEnv(container.Env, envNodeOptions)
	if idx == -1 {
		return []corev1.EnvVar{
			{
				Name:  envNodeOptions,
				Value: nodeRequireArgument,
			},
		}
	} else if idx > -1 {
		// Don't modify NODE_OPTIONS if it uses ValueFrom
		if container.Env[idx].ValueFrom != nil {
			return []corev1.EnvVar{}
		}
		// NODE_OPTIONS is set, append the required argument
		return []corev1.EnvVar{
			{
				Name:  envNodeOptions,
				Value: container.Env[idx].Value + nodeRequireArgument,
			},
		}
	}
	return []corev1.EnvVar{}
}
