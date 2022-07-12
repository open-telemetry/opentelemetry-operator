package instrumentation

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	kernelDebugVolume = "kernel-debug"
)

func injectGolangSDK(_ logr.Logger, golangSpec v1alpha1.Golang, pod corev1.Pod) corev1.Pod {
	// skip instrumentation if share process namespaces is explicitly disabled
	if pod.Spec.ShareProcessNamespace != nil && *pod.Spec.ShareProcessNamespace == false {
		return pod
	}
	execPath, ok := pod.Annotations[annotationGolangExecPath]
	if !ok {
		return pod
	}
	zero := int64(0)
	truee := bool(true)
	pod.Spec.ShareProcessNamespace = &truee
	pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{
		Name:  initContainerName,
		Image: golangSpec.Image,
		Env: []corev1.EnvVar{
			{
				Name:  "OTEL_TARGET_EXE",
				Value: execPath,
			},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:  &zero,
			Privileged: &truee,
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"SYS_PTRACE"},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				MountPath: "/sys/kernel/debug",
				Name:      kernelDebugVolume,
			},
		},
	})
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: kernelDebugVolume,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/sys/kernel/debug",
			},
		},
	})
	return pod
}
