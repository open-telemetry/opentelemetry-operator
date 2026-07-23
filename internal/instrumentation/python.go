// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

const (
	envPythonPath                    = "PYTHONPATH"
	envPythonAgentPathPrefix         = "PYTHON_AUTO_INSTRUMENTATION_AGENT_PATH_PREFIX"
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

func pythonPlatformSrc(platform string) (string, error) {
	// Validate platform
	switch platform {
	case "", glibcLinux:
		return glibcLinuxAutoInstrumentationSrc, nil
	case muslLinux:
		return muslLinuxAutoInstrumentationSrc, nil
	default:
		return "", fmt.Errorf("provided instrumentation.opentelemetry.io/otel-python-platform annotation value '%s' is not supported", platform)
	}
}

func injectPythonSDKToContainer(pythonSpec v1alpha1.Python, container *corev1.Container, platform string) error {
	volume := instrVolume(pythonSpec.VolumeClaimTemplate, pythonVolumeName, pythonSpec.VolumeSizeLimit)

	// In injector mode the instrumentation is activated via LD_PRELOAD instead of PYTHONPATH.
	envToValidate := envPythonPath
	if featuregate.EnableInstrumentationInjector.IsEnabled() {
		envToValidate = envLdPreload
	}
	err := validateContainerEnv(container.Env, envToValidate)
	if err != nil {
		return err
	}

	_, err = pythonPlatformSrc(platform)
	if err != nil {
		return err
	}

	// inject Python instrumentation spec env vars.
	container.Env = appendIfNotSet(container.Env, pythonSpec.Env...)

	if featuregate.EnableInstrumentationInjector.IsEnabled() {
		injectInjectorPythonEnvVars(container)
	} else {
		idx := getIndexOfEnv(container.Env, envPythonPath)
		if idx == -1 {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  envPythonPath,
				Value: fmt.Sprintf("%s:%s", pythonPathPrefix, pythonPathSuffix),
			})
		} else if idx > -1 {
			container.Env[idx].Value = fmt.Sprintf("%s:%s:%s", pythonPathPrefix, container.Env[idx].Value, pythonPathSuffix)
		}
	}

	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      volume.Name,
		MountPath: pythonInstrMountPath,
	})
	return nil
}

// injectInjectorPythonEnvVars activates the Python instrumentation via the
// opentelemetry-injector shared object instead of setting PYTHONPATH on the
// container. The injector is loaded into every process in the container via
// LD_PRELOAD and prepends the instrumentation to PYTHONPATH in-process, only
// for Python processes, detecting the libc flavor (glibc/musl) of the process
// on its own. This makes the instrumentation.opentelemetry.io/otel-python-platform
// annotation unnecessary.
func injectInjectorPythonEnvVars(container *corev1.Container) {
	ldPreloadValue := pythonInstrMountPath + "/" + injectorLibName
	if idx := getIndexOfEnv(container.Env, envLdPreload); idx > -1 {
		ldPreloadValue = container.Env[idx].Value + ":" + ldPreloadValue
	}
	container.Env = appendOrReplace(container.Env,
		corev1.EnvVar{
			Name:  envLdPreload,
			Value: ldPreloadValue,
		},
		corev1.EnvVar{
			Name:  envPythonAgentPathPrefix,
			Value: pythonInstrMountPath,
		},
	)
}

func injectPythonSDKToPod(pythonSpec v1alpha1.Python, pod corev1.Pod, firstContainerName, platform string, instSpec v1alpha1.InstrumentationSpec) corev1.Pod {
	volume := instrVolume(pythonSpec.VolumeClaimTemplate, pythonVolumeName, pythonSpec.VolumeSizeLimit)

	// This has been validated already
	autoInstrumentationSrc, _ := pythonPlatformSrc(platform)

	// We just inject Volumes and init containers for the first processed container.
	if isInitContainerMissing(pod, pythonInitContainerName) {
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)

		command := []string{"cp", "-r", autoInstrumentationSrc, pythonInstrMountPath}
		if featuregate.EnableInstrumentationInjector.IsEnabled() {
			// The injector chooses between the glibc and musl installations at
			// runtime, so both are copied, together with the injector itself.
			command = []string{
				"sh", "-c",
				fmt.Sprintf("mkdir -p %[3]s/%[4]s %[3]s/%[5]s && cp -r %[1]s %[3]s/%[4]s && cp -r %[2]s %[3]s/%[5]s && cp /%[6]s %[3]s/",
					glibcLinuxAutoInstrumentationSrc, muslLinuxAutoInstrumentationSrc, pythonInstrMountPath, glibcLinux, muslLinux, injectorLibName),
			}
		}

		initContainer := corev1.Container{
			Name:      pythonInitContainerName,
			Image:     pythonSpec.Image,
			Command:   command,
			Resources: pythonSpec.Resources,
			VolumeMounts: []corev1.VolumeMount{{
				Name:      volume.Name,
				MountPath: pythonInstrMountPath,
			}},
			ImagePullPolicy: instSpec.ImagePullPolicy,
		}

		pod.Spec.InitContainers = insertInitContainer(&pod, initContainer, firstContainerName)
	}
	return pod
}

// injectPythonSDK injects Python instrumentation into the specified containers.
// Containers must point into the provided pod and be ordered with init containers first.
func injectPythonSDK(pythonSpec v1alpha1.Python, pod *corev1.Pod, containers []*corev1.Container, platform string, instSpec v1alpha1.InstrumentationSpec) error {
	for _, container := range containers {
		if err := injectPythonSDKToContainer(pythonSpec, container, platform); err != nil {
			return err
		}
	}
	if len(containers) > 0 {
		*pod = injectPythonSDKToPod(pythonSpec, *pod, containers[0].Name, platform, instSpec)
	}
	return nil
}

func getDefaultPythonEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		// Set OTEL_EXPORTER_OTLP_PROTOCOL to http/protobuf if not set by user because it is what our autoinstrumentation supports.
		{
			Name:  envOtelExporterOTLPProtocol,
			Value: "http/protobuf",
		},
		// Set OTEL_TRACES_EXPORTER to otlp exporter if not set by user because it is what our autoinstrumentation supports.
		{
			Name:  envOtelTracesExporter,
			Value: "otlp",
		},
		// Set OTEL_METRICS_EXPORTER to otlp exporter if not set by user because it is what our autoinstrumentation supports.
		{
			Name:  envOtelMetricsExporter,
			Value: "otlp",
		},
		// Set OTEL_LOGS_EXPORTER to otlp exporter if not set by user because it is what our autoinstrumentation supports.
		{
			Name:  envOtelLogsExporter,
			Value: "otlp",
		},
	}
}
