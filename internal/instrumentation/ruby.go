// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	envRubyOptions        = "RUBYOPT"
	rubyRequireArgument   = " -r /otel-auto-instrumentation-ruby/src/autoinstrumentation"
	rubyInitContainerName = initContainerName + "-ruby"
	rubyVolumeName        = volumeName + "-ruby"
	rubyInstrMountPath    = "/otel-auto-instrumentation-ruby"
)

func injectRubySDK(rubySpec v1alpha1.Ruby, pod corev1.Pod, container *corev1.Container, instSpec v1alpha1.InstrumentationSpec) (corev1.Pod, error) {
	volume := instrVolume(rubySpec.VolumeClaimTemplate, rubyVolumeName, rubySpec.VolumeSizeLimit)

	err := validateContainerEnv(container.Env, envRubyOptions)
	if err != nil {
		return pod, err
	}

	// inject Ruby instrumentation spec env vars.
	container.Env = appendIfNotSet(container.Env, rubySpec.Env...)

	idx := getIndexOfEnv(container.Env, envRubyOptions)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envRubyOptions,
			Value: rubyRequireArgument,
		})
	} else if idx > -1 {
		container.Env[idx].Value = container.Env[idx].Value + rubyRequireArgument
	}

	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      volume.Name,
		MountPath: rubyInstrMountPath,
	})

	// We just inject Volumes and init containers for the first processed container.
	if isInitContainerMissing(pod, rubyInitContainerName) {
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:      rubyInitContainerName,
			Image:     rubySpec.Image,
			Command:   []string{"cp", "-r", "/autoinstrumentation/.", rubyInstrMountPath},
			Resources: rubySpec.Resources,
			VolumeMounts: []corev1.VolumeMount{{
				Name:      volume.Name,
				MountPath: rubyInstrMountPath,
			}},
			ImagePullPolicy: instSpec.ImagePullPolicy,
		})
	}
	return pod, nil
}
