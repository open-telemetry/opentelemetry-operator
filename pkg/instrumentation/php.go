// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package instrumentation

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
    phpInstrMountPath = "/otel-auto-instrumentation-php"

    phpIniScanDirEnvVarName = "PHP_INI_SCAN_DIR"
    // https://www.php.net/manual/en/configuration.file.php//configuration.file.scan
    //       If a blank directory is given in PHP_INI_SCAN_DIR, PHP will also scan the directory given at compile time via --with-config-file-scan-dir.
    //       PHP_INI_SCAN_DIR=:/usr/local/etc/php.d php
    //                        ^ separator after empty string
    //           PHP will load all files in /etc/php.d/*.ini, then /usr/local/etc/php.d/*.ini as configuration files.
	phpIniScanDirEnvVarValue = ":/" + phpInstrMountPath + "/php_ini_scan_dir"

    otelPhpAutoloadEnabledrEnvVarName = "OTEL_PHP_AUTOLOAD_ENABLED"
    otelPhpAutoloadEnabledrEnvVarValue = "true"

    phpInitContainerName = initContainerName + "-php"
	phpVolumeName = volumeName + "-php"
)

func injectPhpAutoInstrumentation(phpSpec v1alpha1.PHP, pod corev1.Pod, index int) (corev1.Pod, error) {
	// caller checks if there is at least one container.
	container := &pod.Spec.Containers[index]

    err := validateContainerEnv(container.Env, phpIniScanDirEnvVarName, otelPhpAutoloadEnabledrEnvVarName)
    if err != nil {
        return pod, err
    }

	// inject PHP instrumentation spec env vars.
	for _, env := range phpSpec.Env {
		idx := getIndexOfEnv(container.Env, env.Name)
		if idx == -1 {
			container.Env = append(container.Env, env)
		}
	}

    appendToPhpPathLikeEnvVar(container, phpIniScanDirEnvVarName, phpIniScanDirEnvVarValue)
    setPhpEnvVarIfNotSetYet(container, otelPhpAutoloadEnabledrEnvVarName, otelPhpAutoloadEnabledrEnvVarValue)

	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      phpVolumeName,
		MountPath: phpInstrMountPath,
	})

	// We just inject Volumes and init containers for the first processed container.
	if isInitContainerMissing(pod, phpInitContainerName) {
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name: phpVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: volumeSize(phpSpec.VolumeSizeLimit),
				},
			}})

		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:      phpInitContainerName,
			Image:     phpSpec.Image,
			Command:   []string{"cp", "-r", "/autoinstrumentation/", phpInstrMountPath},
			Resources: phpSpec.Resources,
			VolumeMounts: []corev1.VolumeMount{{
				Name:      phpVolumeName,
				MountPath: phpInstrMountPath,
			}},
		})
	}
	return pod, err
}

func setPhpEnvVarIfNotSetYet(container *corev1.Container, envVarName string, envVarValue string) {
	idx := getIndexOfEnv(container.Env, envVarName)
	if idx < 0 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envVarName,
			Value: envVarValue,
		})
		return
	}
}

func appendToPhpPathLikeEnvVar(container *corev1.Container, envVarName string, envVarValue string) {
	idx := getIndexOfEnv(container.Env, envVarName)
	if idx < 0 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envVarName,
			Value: envVarValue,
		})
		return
	}
    container.Env[idx].Value = fmt.Sprintf("%s:%s", container.Env[idx].Value, envVarValue)
}

