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

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
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

func TestSingleInstrumentationEnabled(t *testing.T) {
	tests := []struct {
		name             string
		instrumentations languageInstrumentations
		expectedStatus   bool
		expectedMsg      string
	}{
		{
			name: "Single instrumentation enabled",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}},
				NodeJS: instrumentationWithContainers{Instrumentation: nil},
			},
			expectedStatus: true,
			expectedMsg:    "Java",
		},
		{
			name: "Multiple instrumentations enabled",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}},
				NodeJS: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}},
			},
			expectedStatus: false,
			expectedMsg:    "",
		},
		{
			name: "Instrumentations disabled",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: nil},
				NodeJS: instrumentationWithContainers{Instrumentation: nil},
			},
			expectedStatus: false,
			expectedMsg:    "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ok, msg := isSingleInstrumentationEnabled(test.instrumentations)
			assert.Equal(t, test.expectedStatus, ok)
			assert.Equal(t, test.expectedMsg, msg)
		})
	}
}

func TestContainerNamesConfiguredForMultipleInstrumentations(t *testing.T) {
	tests := []struct {
		name             string
		instrumentations languageInstrumentations
		expectedStatus   bool
		expectedMsg      string
	}{
		{
			name: "Single instrumentation enabled without containers",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}},
				NodeJS: instrumentationWithContainers{Instrumentation: nil},
			},
			expectedStatus: true,
			expectedMsg:    "ok",
		},
		{
			name: "Multiple instrumentations enabled with containers",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: "java"},
				NodeJS: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: "nodejs"},
			},
			expectedStatus: true,
			expectedMsg:    "ok",
		},
		{
			name: "Multiple instrumentations enabled without containers",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}},
				NodeJS: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}},
			},
			expectedStatus: false,
			expectedMsg:    "incorrect instrumentation configuration - please provide container names for all instrumentations",
		},
		{
			name: "Multiple instrumentations enabled with containers for single instrumentation",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: "test"},
				NodeJS: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}},
			},
			expectedStatus: false,
			expectedMsg:    "incorrect instrumentation configuration - please provide container names for all instrumentations",
		},
		{
			name: "Disabled instrumentations",
			instrumentations: languageInstrumentations{
				NodeJS: instrumentationWithContainers{Instrumentation: nil},
			},
			expectedStatus: false,
			expectedMsg:    "instrumentation configuration not provided",
		},
		{
			name: "Multiple instrumentations enabled with duplicated containers",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: "app,app1,java"},
				NodeJS: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: "app1,app,nodejs"},
			},
			expectedStatus: false,
			expectedMsg:    "duplicated container names detected: [app app1]",
		},
		{
			name: "Multiple instrumentations enabled with duplicated containers for single instrumentation",
			instrumentations: languageInstrumentations{
				Java:   instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: "app,app,java"},
				NodeJS: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: "nodejs"},
			},
			expectedStatus: false,
			expectedMsg:    "duplicated container names detected: [app]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ok, msg := areContainerNamesConfiguredForMultipleInstrumentations(test.instrumentations)
			assert.Equal(t, test.expectedStatus, ok)
			assert.Equal(t, test.expectedMsg, msg)
		})
	}
}

func TestDuplicatedContainers(t *testing.T) {
	tests := []struct {
		name               string
		containers         []string
		expectedDuplicates []string
	}{
		{
			name:               "No duplicates",
			containers:         []string{"app1,app2", "app3", "app4,app5"},
			expectedDuplicates: []string(nil),
		},
		{
			name:               "Duplicates in containers",
			containers:         []string{"app1,app2", "app1", "app1,app3,app4", "app4"},
			expectedDuplicates: []string{"app1", "app4"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ok := findDuplicatedContainers(test.containers)
			assert.Equal(t, test.expectedDuplicates, ok)
		})
	}
}

func TestInstrumentationLanguageContainersSet(t *testing.T) {
	tests := []struct {
		name                     string
		instrumentations         languageInstrumentations
		instrumentationName      string
		containers               string
		expectedStatus           bool
		expectedInstrumentations languageInstrumentations
	}{
		{
			name: "Set containers for specific instrumentation",
			instrumentations: languageInstrumentations{
				Python: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}},
			},
			instrumentationName: "Python",
			containers:          "python,python1",
			expectedStatus:      true,
			expectedInstrumentations: languageInstrumentations{
				Python: instrumentationWithContainers{Instrumentation: &v1alpha1.Instrumentation{}, Containers: "python,python1"},
			},
		},
		{
			name:                     "Set containers for unsupported instrumentation",
			instrumentations:         languageInstrumentations{},
			instrumentationName:      "UnknownName",
			containers:               "cont1,cont2",
			expectedStatus:           false,
			expectedInstrumentations: languageInstrumentations{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ok := setInstrumentationLanguageContainers(&test.instrumentations,
				test.instrumentationName, test.containers)
			assert.Equal(t, test.expectedStatus, ok)
			assert.Equal(t, test.instrumentations, test.expectedInstrumentations)
		})
	}
}
