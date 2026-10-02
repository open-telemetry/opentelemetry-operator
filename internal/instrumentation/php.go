// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"fmt"
	"slices"

	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	phpInstrMountPath = "/otel-auto-instrumentation-php"

	// https://www.php.net/manual/en/configuration.file.php//configuration.file.scan
	phpIniScanDirEnvVarName  = "PHP_INI_SCAN_DIR"
	phpIniScanDirEnvVarValue = ":" + phpInstrMountPath

	otelPhpAutoloadEnabledrEnvVarName  = "OTEL_PHP_AUTOLOAD_ENABLED"
	otelPhpAutoloadEnabledrEnvVarValue = "true"

	linuxPhpAutoInstrumentationSrc = "/autoinstrumentation/."

	phpInitContainerName = initContainerName + "-php"
	phpVolumeName        = volumeName + "-php"

	glibc           = "glibc"
	musl            = "musl"
	Php81ApiVersion = "20210902"
	Php82ApiVersion = "20220829"
	Php83ApiVersion = "20230831"
	Php84ApiVersion = "20240924"
	Php85ApiVersion = "20250925"
	zts             = "zts"
	nonZts          = "non-zts"
)

var (
	validPlatforms    = []string{glibc, musl}
	validApiVersions  = []string{Php81ApiVersion, Php82ApiVersion, Php83ApiVersion, Php84ApiVersion, Php85ApiVersion}
	validThreadSafety = []string{nonZts, zts}
)

func injectPhpSDKToContainer(phpSpec v1alpha1.Php, container *corev1.Container, platform, apiVersion, threadSafety string) error {
	err := validateContainerEnv(container.Env, phpIniScanDirEnvVarName, otelPhpAutoloadEnabledrEnvVarName)
	if err != nil {
		return err
	}

	if platform != "" && !slices.Contains(validPlatforms, platform) {
		return fmt.Errorf("provided instrumentation.opentelemetry.io/otel-php-platform annotation value '%s' is not supported", platform)
	}

	if !slices.Contains(validApiVersions, apiVersion) {
		return fmt.Errorf("provided instrumentation.opentelemetry.io/otel-php-api-version annotation value '%s' is not supported", apiVersion)
	}

	if threadSafety != "" && !slices.Contains(validThreadSafety, threadSafety) {
		return fmt.Errorf("provided instrumentation.opentelemetry.io/otel-php-thread-safety annotation value '%s' is not supported", threadSafety)
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

func injectPhpSDKToPod(phpSpec v1alpha1.Php, pod corev1.Pod, firstContainerName string, instSpec v1alpha1.InstrumentationSpec, platform, apiVersion, threadSafety string) corev1.Pod {
	volume := instrVolume(phpSpec.VolumeClaimTemplate, phpVolumeName, phpSpec.VolumeSizeLimit)
	if platform == "" {
		platform = glibc
	}
	if threadSafety == "" {
		threadSafety = nonZts
	}
	// init container
	if isInitContainerMissing(pod, phpInitContainerName) {
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)

		initContainer := corev1.Container{
			Name:      phpInitContainerName,
			Image:     phpSpec.Image,
			Command:   []string{"/bin/sh", "-c"},
			Args:      []string{phpAgentScript, "--", linuxPhpAutoInstrumentationSrc, phpInstrMountPath, platform, apiVersion, threadSafety},
			Resources: phpSpec.Resources,
			VolumeMounts: []corev1.VolumeMount{{
				Name:      volume.Name,
				MountPath: phpInstrMountPath,
			}},
			ImagePullPolicy: instSpec.ImagePullPolicy,
		}

		pod.Spec.InitContainers = insertInitContainer(&pod, initContainer, firstContainerName)
	}

	return pod
}

// injectPhpSDK injects PHP instrumentation into the specified containers.
// Containers must point into the provided pod and be ordered with init containers first.
func injectPhpSDK(phpSpec v1alpha1.Php, pod *corev1.Pod, containers []*corev1.Container, instSpec v1alpha1.InstrumentationSpec, platform, apiVersion, threadSafety string) error {
	for _, container := range containers {
		if err := injectPhpSDKToContainer(phpSpec, container, platform, apiVersion, threadSafety); err != nil {
			return err
		}
	}
	if len(containers) > 0 {
		*pod = injectPhpSDKToPod(phpSpec, *pod, containers[0].Name, instSpec, platform, apiVersion, threadSafety)
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
