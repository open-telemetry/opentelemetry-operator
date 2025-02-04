// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	envDotNetCoreClrEnableProfiling     = "CORECLR_ENABLE_PROFILING"
	envDotNetCoreClrProfiler            = "CORECLR_PROFILER"
	envDotNetCoreClrProfilerPath        = "CORECLR_PROFILER_PATH"
	envDotNetAdditionalDeps             = "DOTNET_ADDITIONAL_DEPS"
	envDotNetSharedStore                = "DOTNET_SHARED_STORE"
	envDotNetStartupHook                = "DOTNET_STARTUP_HOOKS"
	envDotNetOTelAutoHome               = "OTEL_DOTNET_AUTO_HOME"
	dotNetCoreClrEnableProfilingEnabled = "1"
	dotNetCoreClrProfilerID             = "{918728DD-259F-4A6A-AC2B-B85E1B658318}"
	dotNetCoreClrProfilerGlibcPath      = "/otel-auto-instrumentation-dotnet/linux-x64/OpenTelemetry.AutoInstrumentation.Native.so"
	dotNetCoreClrProfilerMuslPath       = "/otel-auto-instrumentation-dotnet/linux-musl-x64/OpenTelemetry.AutoInstrumentation.Native.so"
	dotNetAdditionalDepsPath            = "/otel-auto-instrumentation-dotnet/AdditionalDeps"
	dotNetOTelAutoHomePath              = "/otel-auto-instrumentation-dotnet"
	dotNetSharedStorePath               = "/otel-auto-instrumentation-dotnet/store"
	dotNetStartupHookPath               = "/otel-auto-instrumentation-dotnet/net/OpenTelemetry.AutoInstrumentation.StartupHook.dll"
	dotnetVolumeName                    = volumeName + "-dotnet"
	dotnetInitContainerName             = initContainerName + "-dotnet"
	dotnetInstrMountPath                = "/otel-auto-instrumentation-dotnet"
)

// Supported .NET runtime identifiers (https://learn.microsoft.com/en-us/dotnet/core/rid-catalog), can be set by instrumentation.opentelemetry.io/inject-dotnet.
const (
	dotNetRuntimeLinuxGlibc = "linux-x64"
	dotNetRuntimeLinuxMusl  = "linux-musl-x64"
)

func injectDotNetSDK(dotNetSpec v1alpha1.DotNet, pod corev1.Pod, index int, runtime string) (corev1.Pod, error) {

	volume := instrVolume(dotNetSpec.VolumeClaimTemplate, dotnetVolumeName, dotNetSpec.VolumeSizeLimit)

	// caller checks if there is at least one container.
	container := &pod.Spec.Containers[index]

	err := validateContainerEnv(container.Env, envDotNetStartupHook, envDotNetAdditionalDeps, envDotNetSharedStore)
	if err != nil {
		return pod, err
	}

	// check if OTEL_DOTNET_AUTO_HOME env var is already set in the container
	// if it is already set, then we assume that .NET Auto-instrumentation is already configured for this container
	if getIndexOfEnv(container.Env, envDotNetOTelAutoHome) > -1 {
		return pod, errors.New("OTEL_DOTNET_AUTO_HOME environment variable is already set in the container")
	}

	// check if OTEL_DOTNET_AUTO_HOME env var is already set in the .NET instrumentation spec
	// if it is already set, then we assume that .NET Auto-instrumentation is already configured for this container
	if getIndexOfEnv(dotNetSpec.Env, envDotNetOTelAutoHome) > -1 {
		return pod, errors.New("OTEL_DOTNET_AUTO_HOME environment variable is already set in the .NET instrumentation spec")
	}

	coreClrProfilerPath := ""
	switch runtime {
	case "", dotNetRuntimeLinuxGlibc:
		coreClrProfilerPath = dotNetCoreClrProfilerGlibcPath
	case dotNetRuntimeLinuxMusl:
		coreClrProfilerPath = dotNetCoreClrProfilerMuslPath
	default:
		return pod, fmt.Errorf("provided instrumentation.opentelemetry.io/dotnet-runtime annotation value '%s' is not supported", runtime)
	}

	// inject .NET instrumentation spec env vars.
	for _, env := range dotNetSpec.Env {
		idx := getIndexOfEnv(container.Env, env.Name)
		if idx == -1 {
			container.Env = append(container.Env, env)
		}
	}

	const (
		doNotConcatEnvValues = false
		concatEnvValues      = true
	)

	setDotNetEnvVar(container, envDotNetCoreClrEnableProfiling, dotNetCoreClrEnableProfilingEnabled, doNotConcatEnvValues)

	setDotNetEnvVar(container, envDotNetCoreClrProfiler, dotNetCoreClrProfilerID, doNotConcatEnvValues)

	setDotNetEnvVar(container, envDotNetCoreClrProfilerPath, coreClrProfilerPath, doNotConcatEnvValues)

	setDotNetEnvVar(container, envDotNetStartupHook, dotNetStartupHookPath, concatEnvValues)

	setDotNetEnvVar(container, envDotNetAdditionalDeps, dotNetAdditionalDepsPath, concatEnvValues)

	setDotNetEnvVar(container, envDotNetOTelAutoHome, dotNetOTelAutoHomePath, doNotConcatEnvValues)

	setDotNetEnvVar(container, envDotNetSharedStore, dotNetSharedStorePath, concatEnvValues)

	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      volume.Name,
		MountPath: dotnetInstrMountPath,
	})

	// We just inject Volumes and init containers for the first processed container.
	if isInitContainerMissing(pod, dotnetInitContainerName) {
		pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, corev1.Container{
			Name:      dotnetInitContainerName,
			Image:     dotNetSpec.Image,
			Command:   []string{"cp", "-r", "/autoinstrumentation/.", dotnetInstrMountPath},
			Resources: dotNetSpec.Resources,
			VolumeMounts: []corev1.VolumeMount{{
				Name:      volume.Name,
				MountPath: dotnetInstrMountPath,
			}},
		})
	}
	return pod, nil
}

// setDotNetEnvVar function sets env var to the container if not exist already.
// value of concatValues should be set to true if the env var supports multiple values separated by :.
// If it is set to false, the original container's env var value has priority.
func setDotNetEnvVar(container *corev1.Container, envVarName string, envVarValue string, concatValues bool) {
	idx := getIndexOfEnv(container.Env, envVarName)
	if idx < 0 {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  envVarName,
			Value: envVarValue,
		})
		return
	}
	if concatValues {
		container.Env[idx].Value = fmt.Sprintf("%s:%s", container.Env[idx].Value, envVarValue)
	}
}
