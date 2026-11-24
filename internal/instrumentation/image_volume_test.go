// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestInjectImageVolumes(t *testing.T) {
	tests := []struct {
		name       string
		image      string
		volumeName string
		mountPath  string
		subPath    string
		inject     func(*testing.T, corev1.Pod) corev1.Pod
	}{
		{
			name:       "java",
			image:      "java:latest",
			volumeName: javaVolumeName,
			mountPath:  javaInstrMountPath + "-app",
			inject: func(t *testing.T, pod corev1.Pod) corev1.Pod {
				require.NoError(t, injectJavaagentToContainer(v1alpha1.Java{Image: "java:latest"}, &pod.Spec.Containers[0], "", true))
				return injectJavaagentToPod(v1alpha1.Java{Image: "java:latest"}, pod, "app", v1alpha1.InstrumentationSpec{}, true, nil)
			},
		},
		{
			name:       "nodejs",
			image:      "nodejs:latest",
			volumeName: nodejsVolumeName,
			mountPath:  nodejsInstrMountPath,
			subPath:    "autoinstrumentation",
			inject: func(t *testing.T, pod corev1.Pod) corev1.Pod {
				require.NoError(t, injectNodeJSSDKToContainer(v1alpha1.NodeJS{Image: "nodejs:latest"}, &pod.Spec.Containers[0], true))
				return injectNodeJSSDKToPod(v1alpha1.NodeJS{Image: "nodejs:latest"}, pod, "app", v1alpha1.InstrumentationSpec{}, true)
			},
		},
		{
			name:       "python",
			image:      "python:latest",
			volumeName: pythonVolumeName,
			mountPath:  pythonInstrMountPath,
			subPath:    "autoinstrumentation",
			inject: func(t *testing.T, pod corev1.Pod) corev1.Pod {
				require.NoError(t, injectPythonSDKToContainer(v1alpha1.Python{Image: "python:latest"}, &pod.Spec.Containers[0], "", true))
				return injectPythonSDKToPod(v1alpha1.Python{Image: "python:latest"}, pod, "app", "", v1alpha1.InstrumentationSpec{}, true)
			},
		},
		{
			name:       "ruby",
			image:      "ruby:latest",
			volumeName: rubyVolumeName,
			mountPath:  rubyInstrMountPath,
			subPath:    "autoinstrumentation",
			inject: func(t *testing.T, pod corev1.Pod) corev1.Pod {
				require.NoError(t, injectRubySDKToContainer(v1alpha1.Ruby{Image: "ruby:latest"}, &pod.Spec.Containers[0], true))
				return injectRubySDKToPod(v1alpha1.Ruby{Image: "ruby:latest"}, pod, "app", v1alpha1.InstrumentationSpec{}, true)
			},
		},
		{
			name:       "dotnet",
			image:      "dotnet:latest",
			volumeName: dotnetVolumeName,
			mountPath:  dotnetInstrMountPath,
			subPath:    "autoinstrumentation",
			inject: func(t *testing.T, pod corev1.Pod) corev1.Pod {
				require.NoError(t, injectDotNetSDKToContainer(v1alpha1.DotNet{Image: "dotnet:latest"}, &pod.Spec.Containers[0], "", true))
				return injectDotNetSDKToPod(v1alpha1.DotNet{Image: "dotnet:latest"}, pod, "app", v1alpha1.InstrumentationSpec{}, true)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := tt.inject(t, corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app"}},
				},
			})

			require.Len(t, pod.Spec.Volumes, 1)
			require.NotNil(t, pod.Spec.Volumes[0].Image)
			require.Equal(t, tt.volumeName, pod.Spec.Volumes[0].Name)
			require.Equal(t, tt.image, pod.Spec.Volumes[0].Image.Reference)
			require.Empty(t, pod.Spec.InitContainers)
			require.Len(t, pod.Spec.Containers[0].VolumeMounts, 1)
			require.Equal(t, tt.mountPath, pod.Spec.Containers[0].VolumeMounts[0].MountPath)
			require.Equal(t, tt.subPath, pod.Spec.Containers[0].VolumeMounts[0].SubPath)
		})
	}
}

func TestInjectJavaImageVolumeExtensions(t *testing.T) {
	java := v1alpha1.Java{
		Image: "java:latest",
		Extensions: []v1alpha1.Extensions{{
			Image: "extension:latest",
			Dir:   "/extension",
		}},
	}
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app"}},
		},
	}
	require.NoError(t, injectJavaagentToContainer(java, &pod.Spec.Containers[0], "", true))
	pod = injectJavaagentToPod(java, pod, "app", v1alpha1.InstrumentationSpec{}, true, nil)

	require.Len(t, pod.Spec.Volumes, 2)
	require.Equal(t, "java:latest", pod.Spec.Volumes[0].Image.Reference)
	require.Equal(t, "extension:latest", pod.Spec.Volumes[1].Image.Reference)
	require.Empty(t, pod.Spec.InitContainers)
	require.Len(t, pod.Spec.Containers[0].VolumeMounts, 2)
	require.Equal(t, "extension", pod.Spec.Containers[0].VolumeMounts[1].SubPath)
}
