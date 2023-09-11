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
	envPythonPath                      = "PYTHONPATH"
	envOtelTracesExporter              = "OTEL_TRACES_EXPORTER"
	envOtelMetricsExporter             = "OTEL_METRICS_EXPORTER"
	envOtelExporterOTLPTracesProtocol  = "OTEL_EXPORTER_OTLP_TRACES_PROTOCOL"
	envOtelExporterOTLPMetricsProtocol = "OTEL_EXPORTER_OTLP_METRICS_PROTOCOL"
	pythonPathPrefix                   = "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation"
	pythonPathSuffix                   = "/otel-auto-instrumentation-python"
	pythonInstrMountPath               = "/otel-auto-instrumentation-python"
	pythonVolumeName                   = volumeName + "-python"
	pythonInitContainerName            = initContainerName + "-python"
)

func injectPythonSDK(pythonSpec v1alpha1.Python, pod corev1.Pod, index int) (corev1.Pod, error) {
	// caller checks if there is at least one container.
	container := &pod.Spec.Containers[index]

	err := validateContainerEnv(container.Env, envPythonPath)
	if err != nil {
		return pod, err
	}

	// inject Python instrumentation spec env vars.
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
		container.Env[idx].Value = fmt.Sprintf("%s:%s:%s", pythonPathPrefix, container.Env[idx].Value, pythonPathSuffix)
	}

	// Set OTEL_TRACES_EXPORTER to HTTP exporter if not set by user because it is what our autoinstrumentation supports.
	idx = getIndexOfEnv(container.Env, envOtelTracesExporter)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envOtelTracesExporter,
			Value: "otlp",
		})
	}

	// Set OTEL_EXPORTER_OTLP_TRACES_PROTOCOL to http/protobuf if not set by user because it is what our autoinstrumentation supports.
	idx = getIndexOfEnv(container.Env, envOtelExporterOTLPTracesProtocol)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envOtelExporterOTLPTracesProtocol,
			Value: "http/protobuf",
		})
	}

	// Set OTEL_METRICS_EXPORTER to HTTP exporter if not set by user because it is what our autoinstrumentation supports.
	idx = getIndexOfEnv(container.Env, envOtelMetricsExporter)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envOtelMetricsExporter,
			Value: "otlp",
		})
	}

	// Set OTEL_EXPORTER_OTLP_METRICS_PROTOCOL to http/protobuf if not set by user because it is what our autoinstrumentation supports.
	idx = getIndexOfEnv(container.Env, envOtelExporterOTLPMetricsProtocol)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envOtelExporterOTLPMetricsProtocol,
			Value: "http/protobuf",
		})
	}

	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      pythonVolumeName,
		MountPath: pythonInstrMountPath,
	})

	// We just inject Volumes and init containers for the first processed container.
	if isInitContainerMissing(pod, pythonInitContainerName) {
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name: pythonVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: volumeSize(pythonSpec.VolumeSizeLimit),
				},
			}})

		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:      pythonInitContainerName,
			Image:     pythonSpec.Image,
			Command:   []string{"cp", "-a", "/autoinstrumentation/.", pythonInstrMountPath},
			Resources: pythonSpec.Resources,
			VolumeMounts: []corev1.VolumeMount{{
				Name:      pythonVolumeName,
				MountPath: pythonInstrMountPath,
			}},
		})
	}
	return pod, nil
}
