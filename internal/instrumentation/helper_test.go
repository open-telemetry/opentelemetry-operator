// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

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
			containers:         []string{"app1", "app2", "app3", "app4", "app5"},
			expectedDuplicates: nil,
		},
		{
			name:               "Duplicates in containers",
			containers:         []string{"app1", "app2", "app1", "app1", "app3", "app4", "app4"},
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

func TestInstrVolume(t *testing.T) {
	tests := []struct {
		name       string
		volume     corev1.PersistentVolumeClaimTemplate
		volumeName string
		quantity   *resource.Quantity
		expected   corev1.Volume
	}{
		{
			name: "With volume",
			volume: corev1.PersistentVolumeClaimTemplate{
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				},
			},
			volumeName: "default-vol",
			quantity:   nil,
			expected: corev1.Volume{
				Name: "default-vol",
				VolumeSource: corev1.VolumeSource{
					Ephemeral: &corev1.EphemeralVolumeSource{
						VolumeClaimTemplate: &corev1.PersistentVolumeClaimTemplate{
							Spec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							},
						},
					},
				}},
		},
		{
			name:       "With volume size limit",
			volume:     corev1.PersistentVolumeClaimTemplate{},
			volumeName: "default-vol",
			quantity:   &defaultVolumeLimitSize,
			expected: corev1.Volume{
				Name: "default-vol",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						SizeLimit: &defaultVolumeLimitSize,
					},
				}},
		},
		{
			name:       "No volume or size limit",
			volume:     corev1.PersistentVolumeClaimTemplate{},
			volumeName: "default-vol",
			quantity:   nil,
			expected: corev1.Volume{
				Name: "default-vol",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						SizeLimit: &defaultSize,
					},
				}},
		},
		{
			name: "With volume and size limit",
			volume: corev1.PersistentVolumeClaimTemplate{
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				},
			},
			volumeName: "default-vol",
			quantity:   &defaultVolumeLimitSize,
			expected: corev1.Volume{
				Name: "default-vol",
				VolumeSource: corev1.VolumeSource{
					Ephemeral: &corev1.EphemeralVolumeSource{
						VolumeClaimTemplate: &corev1.PersistentVolumeClaimTemplate{
							Spec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							},
						},
					},
				}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := instrVolume(test.volume, test.volumeName, test.quantity)
			assert.Equal(t, test.expected, res)
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
			containers:     instrumentationWithContainers{Containers: []string{}},
			expectedResult: 0,
		},
		{
			name:           "With containers",
			containers:     instrumentationWithContainers{Containers: []string{"ct1"}},
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
			containers:     instrumentationWithContainers{Containers: []string{}},
			expectedResult: 1,
		},
		{
			name:           "With containers",
			containers:     instrumentationWithContainers{Containers: []string{"ct1"}},
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

func TestEnsureContainer(t *testing.T) {
	tests := []struct {
		name               string
		inst               instrumentationWithContainers
		pod                corev1.Pod
		expectedContainers []string
	}{
		{
			name: "empty containers list",
			inst: instrumentationWithContainers{Containers: []string{}},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app"}},
				},
			},
			expectedContainers: []string{"app"},
		},
		{
			name: "already has containers",
			inst: instrumentationWithContainers{Containers: []string{"my-container"}},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app"}},
				},
			},
			expectedContainers: []string{"my-container"},
		},
		{
			name: "multiple pod containers, empty list",
			inst: instrumentationWithContainers{Containers: []string{}},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app1"}, {Name: "app2"}},
				},
			},
			expectedContainers: []string{"app1"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ensureContainer(&test.inst, test.pod)
			assert.Equal(t, test.expectedContainers, test.inst.Containers)
		})
	}
}

func TestContainersToInstrument(t *testing.T) {
	tests := []struct {
		name          string
		inst          instrumentationWithContainers
		pod           corev1.Pod
		expectedNames []string
	}{
		{
			name: "single container, empty inst",
			inst: instrumentationWithContainers{Containers: []string{}},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app"}},
				},
			},
			expectedNames: []string{"app"},
		},
		{
			name: "single container, matching name",
			inst: instrumentationWithContainers{Containers: []string{"app"}},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app"}},
				},
			},
			expectedNames: []string{"app"},
		},
		{
			name: "multiple containers, one match",
			inst: instrumentationWithContainers{Containers: []string{"app2"}},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app1"}, {Name: "app2"}},
				},
			},
			expectedNames: []string{"app2"},
		},
		{
			name: "multiple containers, multiple matches",
			inst: instrumentationWithContainers{Containers: []string{"app1", "app2"}},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app1"}, {Name: "app2"}, {Name: "app3"}},
				},
			},
			expectedNames: []string{"app1", "app2"},
		},
		{
			name: "no matching container",
			inst: instrumentationWithContainers{Containers: []string{"nonexistent"}},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app"}},
				},
			},
			expectedNames: []string{},
		},
		{
			name: "init containers returned first, in order",
			inst: instrumentationWithContainers{Containers: []string{"app", "init-b", "init-a"}},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "init-a"}, {Name: "init-b"}},
					Containers:     []corev1.Container{{Name: "app"}},
				},
			},
			// init-a comes before init-b in pod spec, so they should be sorted that way
			expectedNames: []string{"init-a", "init-b", "app"},
		},
		{
			name: "init containers sorted by pod order, not annotation order",
			inst: instrumentationWithContainers{Containers: []string{"init-c", "init-a", "init-b"}},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "init-a"}, {Name: "init-b"}, {Name: "init-c"}},
					Containers:     []corev1.Container{{Name: "app"}},
				},
			},
			// Annotation order is c, a, b but pod order is a, b, c
			expectedNames: []string{"init-a", "init-b", "init-c"},
		},
		{
			name: "mixed init and regular containers",
			inst: instrumentationWithContainers{Containers: []string{"app2", "init-b", "app1", "init-a"}},
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "init-a"}, {Name: "init-b"}},
					Containers:     []corev1.Container{{Name: "app1"}, {Name: "app2"}},
				},
			},
			// Init containers first (sorted by pod order), then regular containers (in annotation order)
			expectedNames: []string{"init-a", "init-b", "app2", "app1"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := containersToInstrument(&test.inst, &test.pod)
			names := make([]string, len(result))
			for i, c := range result {
				names[i] = c.Name
			}
			assert.Equal(t, test.expectedNames, names)
		})
	}
}

func TestIsInitContainer(t *testing.T) {
	tests := []struct {
		name     string
		pod      corev1.Pod
		target   string
		expected bool
	}{
		{
			name: "matches init container",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "init-db"}},
					Containers:     []corev1.Container{{Name: "app"}},
				},
			},
			target:   "init-db",
			expected: true,
		},
		{
			name: "matches regular container",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "init-db"}},
					Containers:     []corev1.Container{{Name: "app"}},
				},
			},
			target:   "app",
			expected: false,
		},
		{
			name: "container not found",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "init-db"}},
					Containers:     []corev1.Container{{Name: "app"}},
				},
			},
			target:   "nonexistent",
			expected: false,
		},
		{
			name:     "empty pod",
			pod:      corev1.Pod{},
			target:   "any",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := isInitContainer(test.target, &test.pod)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestFindContainerByName(t *testing.T) {
	tests := []struct {
		name         string
		pod          corev1.Pod
		target       string
		expectedName string
		expectNil    bool
	}{
		{
			name: "finds regular container",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "init-db"}},
					Containers:     []corev1.Container{{Name: "app"}},
				},
			},
			target:       "app",
			expectedName: "app",
		},
		{
			name: "finds init container",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "init-db"}},
					Containers:     []corev1.Container{{Name: "app"}},
				},
			},
			target:       "init-db",
			expectedName: "init-db",
		},
		{
			name: "container not found",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app"}},
				},
			},
			target:    "nonexistent",
			expectNil: true,
		},
		{
			name:      "empty pod",
			pod:       corev1.Pod{},
			target:    "any",
			expectNil: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := findContainerByName(test.target, &test.pod)
			if test.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, test.expectedName, result.Name)
			}
		})
	}
}

func TestInsertInitContainer(t *testing.T) {
	tests := []struct {
		name                   string
		pod                    corev1.Pod
		toInsert               corev1.Container
		targetContainerName    string
		expectedInitContainers []string
	}{
		{
			name: "insert before init container - first position",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "init-a"}, {Name: "init-b"}},
					Containers:     []corev1.Container{{Name: "app"}},
				},
			},
			toInsert:               corev1.Container{Name: "otel-java"},
			targetContainerName:    "init-a",
			expectedInitContainers: []string{"otel-java", "init-a", "init-b"},
		},
		{
			name: "insert before init container - middle position",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "init-a"}, {Name: "init-b"}, {Name: "init-c"}},
					Containers:     []corev1.Container{{Name: "app"}},
				},
			},
			toInsert:               corev1.Container{Name: "otel-java"},
			targetContainerName:    "init-b",
			expectedInitContainers: []string{"init-a", "otel-java", "init-b", "init-c"},
		},
		{
			name: "insert before init container - last position",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "init-a"}, {Name: "init-b"}},
					Containers:     []corev1.Container{{Name: "app"}},
				},
			},
			toInsert:               corev1.Container{Name: "otel-java"},
			targetContainerName:    "init-b",
			expectedInitContainers: []string{"init-a", "otel-java", "init-b"},
		},
		{
			name: "append for regular container",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "init-a"}},
					Containers:     []corev1.Container{{Name: "app"}},
				},
			},
			toInsert:               corev1.Container{Name: "otel-java"},
			targetContainerName:    "app",
			expectedInitContainers: []string{"init-a", "otel-java"},
		},
		{
			name: "append for regular container - no existing init containers",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app"}},
				},
			},
			toInsert:               corev1.Container{Name: "otel-java"},
			targetContainerName:    "app",
			expectedInitContainers: []string{"otel-java"},
		},
		{
			name: "append when target not found",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "init-a"}},
					Containers:     []corev1.Container{{Name: "app"}},
				},
			},
			toInsert:               corev1.Container{Name: "otel-java"},
			targetContainerName:    "nonexistent",
			expectedInitContainers: []string{"init-a", "otel-java"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := insertInitContainer(&test.pod, test.toInsert, test.targetContainerName)
			names := make([]string, len(result))
			for i, c := range result {
				names[i] = c.Name
			}
			assert.Equal(t, test.expectedInitContainers, names)
		})
	}
}
