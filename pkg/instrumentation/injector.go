// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	envLdPreload              = "LD_PRELOAD"
	injectorInitContainerName = initContainerName + "-injector"
	injectorVolumeName        = volumeName + "-injector"
	injectorInstrMountPath    = "/otel-auto-instrumentation-injector"
	ldPreloadValue            = injectorInstrMountPath + "/instrumentation/injector.so"
)

func injectInjector(_ logr.Logger, injectorSpec v1alpha1.Injector, pod corev1.Pod, index int) (corev1.Pod, error) {
	volume := instrVolume(injectorSpec.VolumeClaimTemplate, injectorVolumeName, injectorSpec.VolumeSizeLimit)

	// caller checks if there is at least one container.
	container := &pod.Spec.Containers[index]

	err := validateContainerEnv(container.Env, envLdPreload)
	if err != nil {
		return pod, err
	}

	// inject injector env vars.
	for _, env := range injectorSpec.Env {
		idx := getIndexOfEnv(container.Env, env.Name)
		if idx == -1 {
			container.Env = append(container.Env, env)
		}
	}

	idx := getIndexOfEnv(container.Env, envLdPreload)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envLdPreload,
			Value: ldPreloadValue,
		})
	} else {
		// TODO Should we extend the validateContainerEnv method instead?
		return pod, fmt.Errorf("the container defines env var value via Value, envVar: %s", envLdPreload)
	}

	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      volume.Name,
		MountPath: injectorInstrMountPath,
	})

	// We just inject Volumes and init containers for the first processed container.
	if isInitContainerMissing(pod, injectorInitContainerName) {
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:      injectorInitContainerName,
			Image:     injectorSpec.Image,
			Resources: injectorSpec.Resources,
			Env: []corev1.EnvVar{{
				Name:  "INSTRUMENTATION_FOLDER_DESTINATION",
				Value: injectorInstrMountPath,
			}},
			VolumeMounts: []corev1.VolumeMount{{
				Name:      volume.Name,
				MountPath: injectorInstrMountPath,
			}},
		})
	}
	return pod, err
}
