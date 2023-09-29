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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
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
							Name: javaInitContainerName,
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
			result := isInitContainerMissing(test.pod, javaInitContainerName)
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
							Name: nodejsInitContainerName,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "AutoInstrumentation_Already_Inject_go",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{},
					Containers: []corev1.Container{
						{
							Name: sideCarName,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "AutoInstrumentation_Already_Inject_no_init_containers",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{},
					Containers: []corev1.Container{
						{
							Name: "my-app",
							Env: []corev1.EnvVar{
								{
									Name:  constants.EnvNodeName,
									Value: "value",
								},
							},
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
			result := isAutoInstrumentationInjected(test.pod)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestDuplicatedContainers(t *testing.T) {
	tests := []struct {
		name               string
		containers         []string
		expectedDuplicates error
	}{
		{
			name:               "No duplicates",
			containers:         []string{"app1,app2", "app3", "app4,app5"},
			expectedDuplicates: nil,
		},
		{
			name:               "Duplicates in containers",
			containers:         []string{"app1,app2", "app1", "app1,app3,app4", "app4"},
			expectedDuplicates: fmt.Errorf("duplicated container names detected: [app1 app4]"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ok := findDuplicatedContainers(test.containers)
			assert.Equal(t, test.expectedDuplicates, ok)
		})
	}
}

func TestInstrWithContainers(t *testing.T) {
	tests := []struct {
		name           string
		containers     instrumentationWithContainers
		expectedResult int
	}{
		{
			name:           "No containers",
			containers:     instrumentationWithContainers{Containers: ""},
			expectedResult: 0,
		},
		{
			name:           "With containers",
			containers:     instrumentationWithContainers{Containers: "ct1"},
			expectedResult: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := isInstrWithContainers(test.containers)
			assert.Equal(t, test.expectedResult, res)
		})
	}
}

func TestInstrWithoutContainers(t *testing.T) {
	tests := []struct {
		name           string
		containers     instrumentationWithContainers
		expectedResult int
	}{
		{
			name:           "No containers",
			containers:     instrumentationWithContainers{Containers: ""},
			expectedResult: 1,
		},
		{
			name:           "With containers",
			containers:     instrumentationWithContainers{Containers: "ct1"},
			expectedResult: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := isInstrWithoutContainers(test.containers)
			assert.Equal(t, test.expectedResult, res)
		})
	}
}
