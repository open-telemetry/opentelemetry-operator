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
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	envPythonPath               = "PYTHONPATH"
	envOtelTracesExporter       = "OTEL_TRACES_EXPORTER"
	envOtelMetricsExporter      = "OTEL_METRICS_EXPORTER"
	envOtelLogsExporter         = "OTEL_LOGS_EXPORTER"
	envOtelExporterOTLPProtocol = "OTEL_EXPORTER_OTLP_PROTOCOL"
	pythonPathPrefix            = "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation"
	pythonPathSuffix            = "/otel-auto-instrumentation-python"
	pythonInstrMountPath        = "/otel-auto-instrumentation-python"
	pythonVolumeName            = volumeName + "-python"
	pythonInitContainerName     = initContainerName + "-python"
)

func injectPythonSDK(pythonSpec v1alpha1.Python, pod corev1.Pod, container Container) (corev1.Pod, error) {
	volume := instrVolume(pythonSpec.VolumeClaimTemplate, pythonVolumeName, pythonSpec.VolumeSizeLimit)

	err := container.validate(&pod, envPythonPath)
	if err != nil {
		return pod, err
	}

	// inject Python instrumentation spec env vars.
	for _, env := range pythonSpec.Env {
		container.appendEnvVarIfNotExists(&pod, env)
	}

	envPythonPathVar, err := container.getOrMakeEnvVar(&pod, envPythonPath)
	if err != nil {
		return pod, err
	}
	envPythonPathVar.Value = concatWithColon(pythonPathPrefix, envPythonPathVar.Value, pythonPathSuffix)
	container.setOrAppendEnvVar(&pod, envPythonPathVar)

	// Set OTEL_EXPORTER_OTLP_PROTOCOL to http/protobuf if not set by user because it is what our autoinstrumentation supports.
	container.appendIfNotExists(&pod, envOtelExporterOTLPProtocol, "http/protobuf")

	// Set OTEL_TRACES_EXPORTER to otlp exporter if not set by user because it is what our autoinstrumentation supports.
	container.appendIfNotExists(&pod, envOtelTracesExporter, "otlp")

	// Set OTEL_METRICS_EXPORTER to otlp exporter if not set by user because it is what our autoinstrumentation supports.
	container.appendIfNotExists(&pod, envOtelMetricsExporter, "otlp")

	// Set OTEL_LOGS_EXPORTER to otlp exporter if not set by user because it is what our autoinstrumentation supports.
	container.appendIfNotExists(&pod, envOtelLogsExporter, "otlp")

	pod.Spec.Containers[container.index].VolumeMounts = append(pod.Spec.Containers[container.index].VolumeMounts, corev1.VolumeMount{
		Name:      volume.Name,
		MountPath: pythonInstrMountPath,
	})

	// We just inject Volumes and init containers for the first processed container.
	if isInitContainerMissing(pod, pythonInitContainerName) {
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:      pythonInitContainerName,
			Image:     pythonSpec.Image,
			Command:   []string{"cp", "-r", "/autoinstrumentation/.", pythonInstrMountPath},
			Resources: pythonSpec.Resources,
			VolumeMounts: []corev1.VolumeMount{{
				Name:      volume.Name,
				MountPath: pythonInstrMountPath,
			}},
		})
	}
	return pod, nil
}
