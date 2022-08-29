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
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	envJavaToolsOptions = "JAVA_TOOL_OPTIONS"
	javaJVMArgument     = " -javaagent:/otel-auto-instrumentation/javaagent.jar"
)

func injectJavaagent(logger logr.Logger, javaSpec v1alpha1.Java, pod corev1.Pod, index int) corev1.Pod {
	// caller checks if there is at least one container
	container := &pod.Spec.Containers[index]

	// inject env vars
	for _, env := range javaSpec.Env {
		idx := getIndexOfEnv(container.Env, env.Name)
		if idx == -1 {
			container.Env = append(container.Env, env)
		}
	}

	idx := getIndexOfEnv(container.Env, envJavaToolsOptions)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envJavaToolsOptions,
			Value: javaJVMArgument,
		})
	} else {
		if container.Env[idx].ValueFrom != nil {
			// TODO add to status object or submit it as an event
			logger.Info("Skipping javaagent injection, the container defines JAVA_TOOL_OPTIONS env var value via ValueFrom", "container", container.Name)
			return pod
		}

		container.Env[idx].Value = container.Env[idx].Value + javaJVMArgument

	}

	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      volumeName,
		MountPath: "/otel-auto-instrumentation",
	})

	// We just inject Volumes and init containers for the first processed container
	if isInitContainerMissing(pod) {
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			}})

		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:    initContainerName,
			Image:   javaSpec.Image,
			Command: []string{"cp", "/javaagent.jar", "/otel-auto-instrumentation/javaagent.jar"},
			VolumeMounts: []corev1.VolumeMount{{
				Name:      volumeName,
				MountPath: "/otel-auto-instrumentation",
			}},
		})
	}

	return pod
}
