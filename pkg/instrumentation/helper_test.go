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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestInitContainerMissing(t *testing.T) {
	tests := []struct {
		name     string
		pod      corev1.Pod
		expected bool
	}{
		{
			name: "InitContainer_Already_Inject",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: "istio-init",
						},
						{
							Name: initContainerName,
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "InitContainer_Absent_1",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: "istio-init",
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "InitContainer_Absent_2",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{},
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsInitContainerMissing(test.pod)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestOtAiVolumeMissing(t *testing.T) {
	tests := []struct {
		name     string
		volume   []corev1.VolumeMount
		expected bool
	}{
		{
			name: "Volume_Already_Inject",
			volume: []corev1.VolumeMount{
				{
					Name: volumeName,
				},
			},
			expected: false,
		},
		{
			name: "Volume_Absent_1",
			volume: []corev1.VolumeMount{
				{
					Name: "magic-volume",
				},
			},
			expected: true,
		},
		{
			name:     "Volume_Absent_2",
			volume:   []corev1.VolumeMount{},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsOtAIVolumeMissing(test.volume)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestEnvVarInstrumentationValueMissing(t *testing.T) {
	tests := []struct {
		name               string
		envVar             corev1.EnvVar
		instrumentationStr string
		expected           bool
	}{
		{
			name: "EnvVar_Instrumentation_Value_Already_Inject",
			envVar: corev1.EnvVar{
				Name:  envJavaToolsOptions,
				Value: javaJVMArgument,
			},
			instrumentationStr: javaJVMArgument,
			expected:           false,
		},
		{
			name: "EnvVar_Instrumentation_Value_Absent_1",
			envVar: corev1.EnvVar{

				Name:  envNodeOptions,
				Value: "some-magic-node-options",
			},
			instrumentationStr: envNodeOptions,
			expected:           true,
		}, {
			name:               "EnvVar_Instrumentation_Value_Absent_2",
			envVar:             corev1.EnvVar{},
			instrumentationStr: envNodeOptions,
			expected:           true,
		}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsEnvVarValueInstrumentationMissing(test.envVar, test.instrumentationStr)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestAutoInstrumentationInjected(t *testing.T) {
	tests := []struct {
		name     string
		pod      corev1.Pod
		expected bool
	}{
		{
			name: "AutoInstrumentation_Already_Inject",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: "magic-init",
						},
						{
							Name: initContainerName,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "AutoInstrumentation_Absent_1",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: "magic-init",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "AutoInstrumentation_Absent_2",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{},
			},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsAutoInstrumentationInjected(test.pod)
			assert.Equal(t, test.expected, result)
		})
	}
}
