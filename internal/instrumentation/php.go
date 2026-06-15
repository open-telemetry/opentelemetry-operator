// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	phpInstrMountPath = "/otel-auto-instrumentation-php"
	phpCloneMountPath = "/otel-auto-instrumentation-php-clone"

	// https://www.php.net/manual/en/configuration.file.php//configuration.file.scan
	phpIniScanDirEnvVarName  = "PHP_INI_SCAN_DIR"
	phpIniScanDirEnvVarValue = ":" + phpInstrMountPath

	otelPhpAutoloadEnabledrEnvVarName  = "OTEL_PHP_AUTOLOAD_ENABLED"
	otelPhpAutoloadEnabledrEnvVarValue = "true"

	linuxPhpAutoInstrumentationSrc = "/autoinstrumentation/."

	phpInitContainerName  = initContainerName + "-php"
	phpVolumeName         = volumeName + "-php"
	phpCloneContainerName = initContainerName + "-clone"
	phpCloneVolumeName    = volumeName + "-clone"
)

func injectPhpSDKToContainer(phpSpec v1alpha1.Php, container *corev1.Container) error {
	err := validateContainerEnv(container.Env, phpIniScanDirEnvVarName, otelPhpAutoloadEnabledrEnvVarName)
	if err != nil {
		return err
	}

	// inject Php instrumentation spec env vars.
	container.Env = appendIfNotSet(container.Env, phpSpec.Env...)

	volume := instrVolume(phpSpec.VolumeClaimTemplate, phpVolumeName, phpSpec.VolumeSizeLimit)
	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      volume.Name,
		MountPath: phpInstrMountPath,
	})

	return nil
}

func injectPhpSDKToPodByContainer(phpSpec v1alpha1.Php, pod corev1.Pod, firstContainerName string, container *corev1.Container, instSpec v1alpha1.InstrumentationSpec) corev1.Pod {
	volume := instrVolume(phpSpec.VolumeClaimTemplate, phpVolumeName, phpSpec.VolumeSizeLimit)
	cloneVolume := instrVolume(phpSpec.VolumeClaimTemplate, phpCloneVolumeName, phpSpec.VolumeSizeLimit)
	// init container
	if isInitContainerMissing(pod, phpInitContainerName) {
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
		pod.Spec.Volumes = append(pod.Spec.Volumes, cloneVolume)

		initContainer := corev1.Container{
			Name:      phpInitContainerName,
			Image:     phpSpec.Image,
			Command:   []string{"/bin/sh", "-c"},
			Args:      []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpCloneMountPath, phpInstrMountPath},
			Resources: phpSpec.Resources,
			VolumeMounts: []corev1.VolumeMount{{
				Name:      cloneVolume.Name,
				MountPath: phpCloneMountPath,
			}, {
				Name:      volume.Name,
				MountPath: phpInstrMountPath,
			}},
			ImagePullPolicy: instSpec.ImagePullPolicy,
		}

		pod.Spec.InitContainers = insertInitContainer(&pod, initContainer, firstContainerName)
	}

	// PHP clone container; insert before init container
	if isInitContainerMissing(pod, phpCloneContainerName) {
		cloneContainer := corev1.Container{
			Name:      phpCloneContainerName,
			Image:     container.Image,
			Command:   []string{"/bin/sh", "-c"},
			Args:      []string{phpCloneScript, "--", phpCloneMountPath},
			Resources: phpSpec.Resources,
			VolumeMounts: []corev1.VolumeMount{{
				Name:      cloneVolume.Name,
				MountPath: phpCloneMountPath,
			}},
			ImagePullPolicy: instSpec.ImagePullPolicy,
		}

		pod.Spec.InitContainers = insertInitContainer(&pod, cloneContainer, phpInitContainerName)
	}

	return pod
}

// injectPhpSDK injects PHP instrumentation into the specified containers.
// Containers must point into the provided pod and be ordered with init containers first.
func injectPhpSDK(phpSpec v1alpha1.Php, pod *corev1.Pod, containers []*corev1.Container, instSpec v1alpha1.InstrumentationSpec) error {
	for _, container := range containers {
		if isInitContainer(container.Name, pod) {
			continue
		}
		if err := injectPhpSDKToContainer(phpSpec, container); err != nil {
			return err
		}
		*pod = injectPhpSDKToPodByContainer(phpSpec, *pod, containers[0].Name, container, instSpec)
	}
	return nil
}

func getDefaultPhpEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  phpIniScanDirEnvVarName,
			Value: phpIniScanDirEnvVarValue,
		},
		{
			Name:  otelPhpAutoloadEnabledrEnvVarName,
			Value: otelPhpAutoloadEnabledrEnvVarValue,
		},
	}
}
