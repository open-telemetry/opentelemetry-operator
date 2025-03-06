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
