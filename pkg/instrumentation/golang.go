// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

const (
	envOtelTargetExe = "OTEL_GO_AUTO_TARGET_EXE"

	kernelDebugVolumeName = "kernel-debug"
	kernelDebugVolumePath = "/sys/kernel/debug"
)

func injectGoSDK(goSpec v1alpha1.Go, pod corev1.Pod, cfg config.Config) (corev1.Pod, error) {
	// skip instrumentation if share process namespaces is explicitly disabled
	if pod.Spec.ShareProcessNamespace != nil && !*pod.Spec.ShareProcessNamespace {
		return pod, fmt.Errorf("shared process namespace has been explicitly disabled")
	}

	// skip instrumentation when more than one containers provided
	containerNames := ""
	ok := false
	if cfg.EnableMultiInstrumentation() {
		containerNames, ok = pod.Annotations[annotationInjectGoContainersName]
	} else {
		containerNames, ok = pod.Annotations[annotationInjectContainerName]
	}

	if ok && len(strings.Split(containerNames, ",")) > 1 {
		return pod, fmt.Errorf("go instrumentation cannot be injected into a pod, multiple containers configured")
	}

	true := true
	zero := int64(0)
	pod.Spec.ShareProcessNamespace = &true

	goAgent := corev1.Container{
		Name:      sideCarName,
		Image:     goSpec.Image,
		Resources: goSpec.Resources,
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:  &zero,
			Privileged: &true,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				MountPath: "/sys/kernel/debug",
				Name:      kernelDebugVolumeName,
			},
		},
	}

	// Annotation takes precedence for OTEL_GO_AUTO_TARGET_EXE
	execPath, ok := pod.Annotations[annotationGoExecPath]
	if ok {
		goAgent.Env = append(goAgent.Env, corev1.EnvVar{
			Name:  envOtelTargetExe,
			Value: execPath,
		})
	}

	// Inject Go instrumentation spec env vars.
	// For Go, env vars must be added to the agent contain
	for _, env := range goSpec.Env {
		idx := getIndexOfEnv(goAgent.Env, env.Name)
		if idx == -1 {
			goAgent.Env = append(goAgent.Env, env)
		}
	}

	pod.Spec.Containers = append(pod.Spec.Containers, goAgent)
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: kernelDebugVolumeName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: kernelDebugVolumePath,
			},
		},
	})
	return pod, nil
}
