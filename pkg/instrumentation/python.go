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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	envPythonPath         = "PYTHONPATH"
	envOtelTracesExporter = "OTEL_TRACES_EXPORTER"
	pythonPathPrefix      = "/otel-auto-instrumentation/opentelemetry/instrumentation/auto_instrumentation"
	pythonPathSuffix      = "/otel-auto-instrumentation"
)

func injectPythonSDK(logger logr.Logger, pythonSpec v1alpha1.Python, pod corev1.Pod, index int) corev1.Pod {
	// caller checks if there is at least one container
	container := &pod.Spec.Containers[index]

	// inject env vars
	for _, env := range pythonSpec.Env {
		idx := getIndexOfEnv(container.Env, env.Name)
		if idx == -1 {
			container.Env = append(container.Env, env)
		}
	}

	idx := getIndexOfEnv(container.Env, envPythonPath)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envPythonPath,
			Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
		})
	} else if idx > -1 {
		if container.Env[idx].ValueFrom != nil {
			// TODO add to status object or submit it as an event
			logger.Info("Skipping Python SDK injection, the container defines PYTHONPATH env var value via ValueFrom", "container", container.Name)
			return pod
		}

		container.Env[idx].Value = fmt.Sprintf("%s:%s:%s", pythonPathPrefix, container.Env[idx].Value, pythonPathSuffix)

	}

	// Set OTEL_TRACES_EXPORTER to HTTP exporter if not set by user because it is what our autoinstrumentation supports.
	idx = getIndexOfEnv(container.Env, envOtelTracesExporter)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envOtelTracesExporter,
			Value: "otlp_proto_http",
		})
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
			Image:   pythonSpec.Image,
			Command: []string{"cp", "-a", "/autoinstrumentation/.", "/otel-auto-instrumentation/"},
			VolumeMounts: []corev1.VolumeMount{{
				Name:      volumeName,
				MountPath: "/otel-auto-instrumentation",
			}},
		})
	}

	return pod
}
