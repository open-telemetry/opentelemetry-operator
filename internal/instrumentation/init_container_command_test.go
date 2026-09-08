// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

func TestCustomInitContainerCommand(t *testing.T) {
	custom := v1alpha1.InitContainer{
		Command: []string{"/bin/sh", "-c"},
		Args:    []string{"copy-vendor-instrumentation"},
	}
	pod := corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app"}}}}
	instSpec := v1alpha1.InstrumentationSpec{}

	tests := []struct {
		name   string
		inject func() corev1.Pod
	}{
		{
			name: "java",
			inject: func() corev1.Pod {
				return injectJavaagentToPod(v1alpha1.Java{Image: "vendor/java", InitContainer: custom}, pod, "app", instSpec, nil)
			},
		},
		{
			name: "nodejs",
			inject: func() corev1.Pod {
				return injectNodeJSSDKToPod(v1alpha1.NodeJS{Image: "vendor/nodejs", InitContainer: custom}, pod, "app", instSpec)
			},
		},
		{
			name: "python",
			inject: func() corev1.Pod {
				return injectPythonSDKToPod(v1alpha1.Python{Image: "vendor/python", InitContainer: custom}, pod, "app", glibcLinux, instSpec)
			},
		},
		{
			name: "dotnet",
			inject: func() corev1.Pod {
				return injectDotNetSDKToPod(v1alpha1.DotNet{Image: "vendor/dotnet", InitContainer: custom}, pod, "app", instSpec)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.inject()
			if assert.Len(t, got.Spec.InitContainers, 1) {
				assert.Equal(t, custom.Command, got.Spec.InitContainers[0].Command)
				assert.Equal(t, custom.Args, got.Spec.InitContainers[0].Args)
			}
		})
	}
}

func TestInitContainerCommandUsesDefault(t *testing.T) {
	defaultCommand := []string{"cp", "/source", "/destination"}
	command, args := initContainerCommand(defaultCommand, nil, nil)
	assert.Equal(t, defaultCommand, command)
	assert.Nil(t, args)
}
