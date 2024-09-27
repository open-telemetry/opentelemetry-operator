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

package manifests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMutateServiceAccount(t *testing.T) {
	existing := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: "simplest",
			Annotations: map[string]string{
				"config.openshift.io/serving-cert-secret-name": "my-secret",
			},
		},
	}
	desired := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: "simplest",
		},
	}

	mutateFn := MutateFuncFor(&existing, &desired)
	err := mutateFn()
	require.NoError(t, err)
	assert.Equal(t, corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "simplest",
			Annotations: map[string]string{"config.openshift.io/serving-cert-secret-name": "my-secret"},
		},
	}, existing)
}

func TestMutateDaemonsetAdditionalContainers(t *testing.T) {
	tests := []struct {
		name     string
		existing appsv1.DaemonSet
		desired  appsv1.DaemonSet
	}{
		{
			name: "add container to daemonset",
			existing: appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "daemonset",
				},
				Spec: appsv1.DaemonSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
							},
						},
					},
				},
			},
			desired: appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "daemonset",
				},
				Spec: appsv1.DaemonSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
								{
									Name:  "alpine",
									Image: "alpine:latest",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "remove container from daemonset",
			existing: appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "daemonset",
				},
				Spec: appsv1.DaemonSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
								{
									Name:  "alpine",
									Image: "alpine:latest",
								},
							},
						},
					},
				},
			},
			desired: appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "daemonset",
				},
				Spec: appsv1.DaemonSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "modify container in daemonset",
			existing: appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "daemonset",
				},
				Spec: appsv1.DaemonSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
								{
									Name:  "alpine",
									Image: "alpine:latest",
								},
							},
						},
					},
				},
			},
			desired: appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "daemonset",
				},
				Spec: appsv1.DaemonSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
								{
									Name:  "alpine",
									Image: "alpine:1.0",
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutateFn := MutateFuncFor(&tt.existing, &tt.desired)
			err := mutateFn()
			require.NoError(t, err)
			assert.Equal(t, tt.existing, tt.desired)
		})
	}
}

func TestMutateDeploymentAdditionalContainers(t *testing.T) {
	tests := []struct {
		name     string
		existing appsv1.Deployment
		desired  appsv1.Deployment
	}{
		{
			name: "add container to deployment",
			existing: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "deployment",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
							},
						},
					},
				},
			},
			desired: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "deployment",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
								{
									Name:  "alpine",
									Image: "alpine:latest",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "remove container from deployment",
			existing: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "deployment",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
								{
									Name:  "alpine",
									Image: "alpine:latest",
								},
							},
						},
					},
				},
			},
			desired: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "deployment",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "modify container in deployment",
			existing: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "deployment",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
								{
									Name:  "alpine",
									Image: "alpine:latest",
								},
							},
						},
					},
				},
			},
			desired: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: "deployment",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
								{
									Name:  "alpine",
									Image: "alpine:1.0",
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutateFn := MutateFuncFor(&tt.existing, &tt.desired)
			err := mutateFn()
			require.NoError(t, err)
			assert.Equal(t, tt.existing, tt.desired)
		})
	}
}

func TestMutateStatefulSetAdditionalContainers(t *testing.T) {
	tests := []struct {
		name     string
		existing appsv1.StatefulSet
		desired  appsv1.StatefulSet
	}{
		{
			name: "add container to statefulset",
			existing: appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "statefulset",
				},
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
							},
						},
					},
				},
			},
			desired: appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "statefulset",
				},
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
								{
									Name:  "alpine",
									Image: "alpine:latest",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "remove container from statefulset",
			existing: appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "statefulset",
				},
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
								{
									Name:  "alpine",
									Image: "alpine:latest",
								},
							},
						},
					},
				},
			},
			desired: appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "statefulset",
				},
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "modify container in statefulset",
			existing: appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "statefulset",
				},
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
								{
									Name:  "alpine",
									Image: "alpine:latest",
								},
							},
						},
					},
				},
			},
			desired: appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "statefulset",
				},
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "collector",
									Image: "collector:latest",
								},
								{
									Name:  "alpine",
									Image: "alpine:1.0",
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mutateFn := MutateFuncFor(&tt.existing, &tt.desired)
			err := mutateFn()
			require.NoError(t, err)
			assert.Equal(t, tt.existing, tt.desired)
		})
	}
}
