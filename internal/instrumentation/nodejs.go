// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

const (
	envNodeOptions          = "NODE_OPTIONS"
	envNodeJSAgentPath      = "NODEJS_AUTO_INSTRUMENTATION_AGENT_PATH"
	nodeRequireArgument     = " --require /otel-auto-instrumentation-nodejs/autoinstrumentation.js"
	nodejsAgentPath         = nodejsInstrMountPath + "/autoinstrumentation.js"
	nodejsInitContainerName = initContainerName + "-nodejs"
	nodejsVolumeName        = volumeName + "-nodejs"
	nodejsInstrMountPath    = "/otel-auto-instrumentation-nodejs"
)

func injectNodeJSSDKToContainer(nodeJSSpec v1alpha1.NodeJS, container *corev1.Container) error {
	volume := instrVolume(nodeJSSpec.VolumeClaimTemplate, nodejsVolumeName, nodeJSSpec.VolumeSizeLimit)

	// In injector mode the instrumentation is activated via LD_PRELOAD instead of NODE_OPTIONS.
	envToValidate := envNodeOptions
	if featuregate.EnableInstrumentationInjector.IsEnabled() {
		envToValidate = envLdPreload
	}
	err := validateContainerEnv(container.Env, envToValidate)
	if err != nil {
		return err
	}

	// inject NodeJS instrumentation spec env vars.
	container.Env = appendIfNotSet(container.Env, nodeJSSpec.Env...)

	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      volume.Name,
		MountPath: nodejsInstrMountPath,
	})
	return nil
}

func injectNodeJSSDKToPod(nodeJSSpec v1alpha1.NodeJS, pod corev1.Pod, firstContainerName string, instSpec v1alpha1.InstrumentationSpec) corev1.Pod {
	volume := instrVolume(nodeJSSpec.VolumeClaimTemplate, nodejsVolumeName, nodeJSSpec.VolumeSizeLimit)

	// We just inject Volumes and init containers for the first processed container
	if isInitContainerMissing(pod, nodejsInitContainerName) {
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)

		initContainer := corev1.Container{
			Name:      nodejsInitContainerName,
			Image:     nodeJSSpec.Image,
			Command:   []string{"cp", "-r", "/autoinstrumentation/.", nodejsInstrMountPath},
			Resources: nodeJSSpec.Resources,
			VolumeMounts: []corev1.VolumeMount{{
				Name:      volume.Name,
				MountPath: nodejsInstrMountPath,
			}},
			ImagePullPolicy: instSpec.ImagePullPolicy,
		}

		pod.Spec.InitContainers = insertInitContainer(&pod, initContainer, firstContainerName)
	}
	return pod
}

// injectNodeJSSDK injects Node.js instrumentation into the specified containers.
// Containers must point into the provided pod and be ordered with init containers first.
func injectNodeJSSDK(nodeJSSpec v1alpha1.NodeJS, pod *corev1.Pod, containers []*corev1.Container, instSpec v1alpha1.InstrumentationSpec) error {
	for _, container := range containers {
		if err := injectNodeJSSDKToContainer(nodeJSSpec, container); err != nil {
			return err
		}
	}
	if len(containers) > 0 {
		*pod = injectNodeJSSDKToPod(nodeJSSpec, *pod, containers[0].Name, instSpec)
	}
	return nil
}

func getDefaultNodeJSEnvVars(container *corev1.Container) []corev1.EnvVar {
	if featuregate.EnableInstrumentationInjector.IsEnabled() {
		return getInjectorNodeJSEnvVars(container)
	}

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

// getInjectorNodeJSEnvVars activates the Node.js instrumentation via the
// opentelemetry-injector shared object instead of setting --require in
// NODE_OPTIONS on the container. The injector is loaded into every process in
// the container via LD_PRELOAD and prepends the --require flag to NODE_OPTIONS
// in-process, only for Node.js processes.
func getInjectorNodeJSEnvVars(container *corev1.Container) []corev1.EnvVar {
	ldPreloadValue := nodejsInstrMountPath + "/" + injectorLibName
	if idx := getIndexOfEnv(container.Env, envLdPreload); idx > -1 {
		ldPreloadValue = container.Env[idx].Value + ":" + ldPreloadValue
	}
	return []corev1.EnvVar{
		{
			Name:  envLdPreload,
			Value: ldPreloadValue,
		},
		{
			Name:  envNodeJSAgentPath,
			Value: nodejsAgentPath,
		},
	}
}
