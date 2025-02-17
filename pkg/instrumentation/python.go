// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	envPythonPath                    = "PYTHONPATH"
	envOtelTracesExporter            = "OTEL_TRACES_EXPORTER"
	envOtelMetricsExporter           = "OTEL_METRICS_EXPORTER"
	envOtelLogsExporter              = "OTEL_LOGS_EXPORTER"
	envOtelExporterOTLPProtocol      = "OTEL_EXPORTER_OTLP_PROTOCOL"
	glibcLinuxAutoInstrumentationSrc = "/autoinstrumentation/."
	muslLinuxAutoInstrumentationSrc  = "/autoinstrumentation-musl/."
	pythonPathPrefix                 = "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation"
	pythonPathSuffix                 = "/otel-auto-instrumentation-python"
	pythonInstrMountPath             = "/otel-auto-instrumentation-python"
	pythonVolumeName                 = volumeName + "-python"
	pythonInitContainerName          = initContainerName + "-python"
	glibcLinux                       = "glibc"
	muslLinux                        = "musl"
)

func injectPythonSDK(pythonSpec v1alpha1.Python, pod corev1.Pod, index int, platform string) (corev1.Pod, error) {
	volume := instrVolume(pythonSpec.VolumeClaimTemplate, pythonVolumeName, pythonSpec.VolumeSizeLimit)

	// caller checks if there is at least one container.
	container := &pod.Spec.Containers[index]

	err := validateContainerEnv(container.Env, envPythonPath)
	if err != nil {
		return pod, err
	}

	autoInstrumentationSrc := ""
	switch platform {
	case "", glibcLinux:
		autoInstrumentationSrc = glibcLinuxAutoInstrumentationSrc
	case muslLinux:
		autoInstrumentationSrc = muslLinuxAutoInstrumentationSrc
	default:
		return pod, fmt.Errorf("provided instrumentation.opentelemetry.io/otel-python-platform annotation value '%s' is not supported", platform)
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

	// Set OTEL_EXPORTER_OTLP_PROTOCOL to http/protobuf if not set by user because it is what our autoinstrumentation supports.
	idx = getIndexOfEnv(container.Env, envOtelExporterOTLPProtocol)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envOtelExporterOTLPProtocol,
			Value: "http/protobuf",
		})
	}

	// Set OTEL_TRACES_EXPORTER to otlp exporter if not set by user because it is what our autoinstrumentation supports.
	idx = getIndexOfEnv(container.Env, envOtelTracesExporter)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envOtelTracesExporter,
			Value: "otlp",
		})
	}

	// Set OTEL_METRICS_EXPORTER to otlp exporter if not set by user because it is what our autoinstrumentation supports.
	idx = getIndexOfEnv(container.Env, envOtelMetricsExporter)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envOtelMetricsExporter,
			Value: "otlp",
		})
	}

	// Set OTEL_LOGS_EXPORTER to otlp exporter if not set by user because it is what our autoinstrumentation supports.
	idx = getIndexOfEnv(container.Env, envOtelLogsExporter)
	if idx == -1 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envOtelLogsExporter,
			Value: "otlp",
		})
	}

	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      volume.Name,
		MountPath: pythonInstrMountPath,
	})

	// We just inject Volumes and init containers for the first processed container.
	if isInitContainerMissing(pod, pythonInitContainerName) {
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:      pythonInitContainerName,
			Image:     pythonSpec.Image,
			Command:   []string{"cp", "-r", autoInstrumentationSrc, pythonInstrMountPath},
			Resources: pythonSpec.Resources,
			VolumeMounts: []corev1.VolumeMount{{
				Name:      volume.Name,
				MountPath: pythonInstrMountPath,
			}},
		})
	}
	return pod, nil
}
