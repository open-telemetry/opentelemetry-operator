// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	envJavaToolsOptions   = "JAVA_TOOL_OPTIONS"
	javaAgent             = " -javaagent:/otel-auto-instrumentation-java/javaagent.jar"
	javaInitContainerName = initContainerName + "-java"
	javaVolumeName        = volumeName + "-java"
	javaInstrMountPath    = "/otel-auto-instrumentation-java"
)

func injectJavaagentToContainer(javaSpec v1alpha1.Java, container *corev1.Container) error {
	volume := instrVolume(javaSpec.VolumeClaimTemplate, javaVolumeName, javaSpec.VolumeSizeLimit)

	err := validateContainerEnv(container.Env, envJavaToolsOptions)
	if err != nil {
		return err
	}

	// inject Java instrumentation spec env vars.
	container.Env = appendIfNotSet(container.Env, javaSpec.Env...)

	// Create unique mount path for this container
	containerMountPath := fmt.Sprintf("%s-%s", javaInstrMountPath, container.Name)

	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      volume.Name,
		MountPath: containerMountPath,
	})
	return nil
}

func injectJavaagentToPod(javaSpec v1alpha1.Java, pod corev1.Pod, firstContainerName string, instSpec v1alpha1.InstrumentationSpec) corev1.Pod {
	volume := instrVolume(javaSpec.VolumeClaimTemplate, javaVolumeName, javaSpec.VolumeSizeLimit)

	// We just inject Volumes and init containers for the first processed container.
	if isInitContainerMissing(pod, javaInitContainerName) {
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)

		initContainer := corev1.Container{
			Name:      javaInitContainerName,
			Image:     javaSpec.Image,
			Command:   []string{"cp", "/javaagent.jar", javaInstrMountPath + "/javaagent.jar"},
			Resources: javaSpec.Resources,
			VolumeMounts: []corev1.VolumeMount{{
				Name:      volume.Name,
				MountPath: javaInstrMountPath,
			}},
			ImagePullPolicy: instSpec.ImagePullPolicy,
		}

		pod.Spec.InitContainers = insertInitContainer(&pod, initContainer, firstContainerName)

		for i, extension := range javaSpec.Extensions {
			extContainer := corev1.Container{
				Name:      initContainerName + fmt.Sprintf("-extension-%d", i),
				Image:     extension.Image,
				Command:   []string{"cp", "-r", extension.Dir + "/.", javaInstrMountPath + "/extensions"},
				Resources: javaSpec.Resources,
				VolumeMounts: []corev1.VolumeMount{{
					Name:      volume.Name,
					MountPath: javaInstrMountPath,
				}},
			}
			pod.Spec.InitContainers = insertInitContainer(&pod, extContainer, firstContainerName)
		}
	}
	return pod
}

// injectJavaagent injects Java instrumentation into the specified containers.
// Containers must point into the provided pod and be ordered with init containers first.
func injectJavaagent(javaSpec v1alpha1.Java, pod *corev1.Pod, containers []*corev1.Container, instSpec v1alpha1.InstrumentationSpec) error {
	for _, container := range containers {
		if err := injectJavaagentToContainer(javaSpec, container); err != nil {
			return err
		}
	}
	if len(containers) > 0 {
		*pod = injectJavaagentToPod(javaSpec, *pod, containers[0].Name, instSpec)
	}
	return nil
}

func getDefaultJavaEnvVars(container *corev1.Container, javaSpec v1alpha1.Java) []corev1.EnvVar {
	containerMountPath := fmt.Sprintf("%s-%s", javaInstrMountPath, container.Name)

	javaJVMArgument := fmt.Sprintf(" -javaagent:%s/javaagent.jar", containerMountPath)
	if len(javaSpec.Extensions) > 0 {
		javaJVMArgument = javaJVMArgument + fmt.Sprintf(" -Dotel.javaagent.extensions=%s/extensions", containerMountPath)
	}

	idx := getIndexOfEnv(container.Env, envJavaToolsOptions)
	if idx == -1 {
		return []corev1.EnvVar{
			{
				Name:  envJavaToolsOptions,
				Value: javaJVMArgument,
			},
		}
	} else {
		// Don't modify JAVA_TOOL_OPTIONS if it uses ValueFrom
		if container.Env[idx].ValueFrom != nil {
			return []corev1.EnvVar{}
		}
		// JAVA_TOOL_OPTIONS present, append our argument to its value
		return []corev1.EnvVar{
			{
				Name:  envJavaToolsOptions,
				Value: container.Env[idx].Value + javaJVMArgument,
			},
		}
	}
}
