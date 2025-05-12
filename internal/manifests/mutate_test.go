// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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
		tt := tt
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
		tt := tt
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutateFn := MutateFuncFor(&tt.existing, &tt.desired)
			err := mutateFn()
			require.NoError(t, err)
			assert.Equal(t, tt.existing, tt.desired)
		})
	}
}

func TestMutateDaemonsetAffinity(t *testing.T) {
	tests := []struct {
		name     string
		existing appsv1.DaemonSet
		desired  appsv1.DaemonSet
	}{
		{
			name: "add affinity to daemonset",
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
							},
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchFields: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/os",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"linux"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "remove affinity from daemonset",
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
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchFields: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/os",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"linux"},
													},
												},
											},
										},
									},
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
			name: "modify affinity in daemonset",
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
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchFields: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/os",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"linux"},
													},
												},
											},
										},
									},
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
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchFields: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/os",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"windows"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutateFn := MutateFuncFor(&tt.existing, &tt.desired)
			err := mutateFn()
			require.NoError(t, err)
			assert.Equal(t, tt.existing, tt.desired)
		})
	}
}

func TestMutateDeploymentAffinity(t *testing.T) {
	tests := []struct {
		name     string
		existing appsv1.Deployment
		desired  appsv1.Deployment
	}{
		{
			name: "add affinity to deployment",
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
							},
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchFields: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/os",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"linux"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "remove affinity from deployment",
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
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchFields: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/os",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"linux"},
													},
												},
											},
										},
									},
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
			name: "modify affinity in deployment",
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
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchFields: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/os",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"linux"},
													},
												},
											},
										},
									},
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
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchFields: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/os",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"windows"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutateFn := MutateFuncFor(&tt.existing, &tt.desired)
			err := mutateFn()
			require.NoError(t, err)
			assert.Equal(t, tt.existing, tt.desired)
		})
	}
}

func TestMutateStatefulSetAffinity(t *testing.T) {
	tests := []struct {
		name     string
		existing appsv1.StatefulSet
		desired  appsv1.StatefulSet
	}{
		{
			name: "add affinity to statefulset",
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
							},
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchFields: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/os",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"linux"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "remove affinity from statefulset",
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
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchFields: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/os",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"linux"},
													},
												},
											},
										},
									},
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
			name: "modify affinity in statefulset",
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
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchFields: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/os",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"linux"},
													},
												},
											},
										},
									},
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
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchFields: []corev1.NodeSelectorRequirement{
													{
														Key:      "kubernetes.io/os",
														Operator: corev1.NodeSelectorOpIn,
														Values:   []string{"windows"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutateFn := MutateFuncFor(&tt.existing, &tt.desired)
			err := mutateFn()
			require.NoError(t, err)
			assert.Equal(t, tt.existing, tt.desired)
		})
	}
}

func TestMutateDaemonsetCollectorArgs(t *testing.T) {
	tests := []struct {
		name     string
		existing appsv1.DaemonSet
		desired  appsv1.DaemonSet
	}{
		{
			name: "add argument to collector container in daemonset",
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
									Args:  []string{"--default-arg=true"},
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
									Args:  []string{"--default-arg=true", "extra-arg=yes"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "remove extra arg from collector container in daemonset",
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
									Args:  []string{"--default-arg=true", "extra-arg=yes"},
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
									Args:  []string{"--default-arg=true"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "modify extra arg in collector container in daemonset",
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
									Args:  []string{"--default-arg=true", "extra-arg=yes"},
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
									Args:  []string{"--default-arg=true", "extra-arg=no"},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutateFn := MutateFuncFor(&tt.existing, &tt.desired)
			err := mutateFn()
			require.NoError(t, err)
			assert.Equal(t, tt.existing, tt.desired)
		})
	}
}

func TestMutateDeploymentCollectorArgs(t *testing.T) {
	tests := []struct {
		name     string
		existing appsv1.Deployment
		desired  appsv1.Deployment
	}{
		{
			name: "add argument to collector container in deployment",
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
									Args:  []string{"--default-arg=true"},
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
									Args:  []string{"--default-arg=true", "extra-arg=yes"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "remove extra arg from collector container in deployment",
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
									Args:  []string{"--default-arg=true", "extra-arg=yes"},
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
									Args:  []string{"--default-arg=true"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "modify extra arg in collector container in deployment",
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
									Args:  []string{"--default-arg=true", "extra-arg=yes"},
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
									Args:  []string{"--default-arg=true", "extra-arg=no"},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutateFn := MutateFuncFor(&tt.existing, &tt.desired)
			err := mutateFn()
			require.NoError(t, err)
			assert.Equal(t, tt.existing, tt.desired)
		})
	}
}

func TestMutateStatefulSetCollectorArgs(t *testing.T) {
	tests := []struct {
		name     string
		existing appsv1.StatefulSet
		desired  appsv1.StatefulSet
	}{
		{
			name: "add argument to collector container in statefulset",
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
									Args:  []string{"--default-arg=true"},
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
									Args:  []string{"--default-arg=true", "extra-arg=yes"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "remove extra arg from collector container in statefulset",
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
									Args:  []string{"--default-arg=true", "extra-arg=yes"},
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
									Args:  []string{"--default-arg=true"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "modify extra arg in collector container in statefulset",
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
									Args:  []string{"--default-arg=true", "extra-arg=yes"},
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
									Args:  []string{"--default-arg=true", "extra-arg=no"},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutateFn := MutateFuncFor(&tt.existing, &tt.desired)
			err := mutateFn()
			require.NoError(t, err)
			assert.Equal(t, tt.existing, tt.desired)
		})
	}
}

func TestNoImmutableLabelChange(t *testing.T) {
	existingSelectorLabels := map[string]string{
		"app.kubernetes.io/component":  "opentelemetry-collector",
		"app.kubernetes.io/instance":   "default.deployment",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/part-of":    "opentelemetry",
	}
	desiredLabels := map[string]string{
		"app.kubernetes.io/component":  "opentelemetry-collector",
		"app.kubernetes.io/instance":   "default.deployment",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/part-of":    "opentelemetry",
		"extra-label":                  "true",
	}
	err := hasImmutableLabelChange(existingSelectorLabels, desiredLabels)
	require.NoError(t, err)
	assert.NoError(t, err)
}

func TestHasImmutableLabelChange(t *testing.T) {
	existingSelectorLabels := map[string]string{
		"app.kubernetes.io/component":  "opentelemetry-collector",
		"app.kubernetes.io/instance":   "default.deployment",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/part-of":    "opentelemetry",
	}
	desiredLabels := map[string]string{
		"app.kubernetes.io/component":  "opentelemetry-collector",
		"app.kubernetes.io/instance":   "default.deployment",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/part-of":    "not-opentelemetry",
	}
	err := hasImmutableLabelChange(existingSelectorLabels, desiredLabels)
	assert.Error(t, err)
}

func TestMissingImmutableLabelChange(t *testing.T) {
	existingSelectorLabels := map[string]string{
		"app.kubernetes.io/component":  "opentelemetry-collector",
		"app.kubernetes.io/instance":   "default.deployment",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/part-of":    "opentelemetry",
	}
	desiredLabels := map[string]string{
		"app.kubernetes.io/component":  "opentelemetry-collector",
		"app.kubernetes.io/instance":   "default.deployment",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
	}
	err := hasImmutableLabelChange(existingSelectorLabels, desiredLabels)
	assert.Error(t, err)
}

func TestMutateDaemonsetError(t *testing.T) {
	tests := []struct {
		name     string
		existing appsv1.DaemonSet
		desired  appsv1.DaemonSet
	}{
		{
			name: "modified immutable label in daemonset",
			existing: appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Now(),
					Name:              "daemonset",
				},
				Spec: appsv1.DaemonSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.daemonset",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.daemonset",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
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
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.daemonset",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.daemonset",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "not-opentelemetry",
							},
						},
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
			name: "modified immutable selector in daemonset",
			existing: appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Now(),
					Name:              "daemonset",
				},
				Spec: appsv1.DaemonSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.daemonset",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.daemonset",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
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
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.daemonset",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "not-opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.daemonset",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
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
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutateFn := MutateFuncFor(&tt.existing, &tt.desired)
			err := mutateFn()
			assert.Error(t, err)
		})
	}
}

func TestMutateDeploymentError(t *testing.T) {
	tests := []struct {
		name     string
		existing appsv1.Deployment
		desired  appsv1.Deployment
	}{
		{
			name: "modified immutable label in deployment",
			existing: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Now(),
					Name:              "deployment",
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.deployment",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.deployment",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
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
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.deployment",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.deployment",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "not-opentelemetry",
							},
						},
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
			name: "modified immutable selector in deployment",
			existing: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Now(),
					Name:              "deployment",
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.deployment",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.deployment",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
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
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.deployment",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "not-opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.deployment",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
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
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutateFn := MutateFuncFor(&tt.existing, &tt.desired)
			err := mutateFn()
			assert.Error(t, err)
		})
	}
}

func TestMutateStatefulSetError(t *testing.T) {
	tests := []struct {
		name     string
		existing appsv1.StatefulSet
		desired  appsv1.StatefulSet
	}{
		{
			name: "modified immutable label in statefulset",
			existing: appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Now(),
					Name:              "statefulset",
				},
				Spec: appsv1.StatefulSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.statefulset",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.statefulset",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
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
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.statefulset",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.statefulset",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "not-opentelemetry",
							},
						},
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
			name: "modified immutable selector in statefulset",
			existing: appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Now(),
					Name:              "statefulset",
				},
				Spec: appsv1.StatefulSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.statefulset",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.statefulset",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
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
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.statefulset",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "not-opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.statefulset",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
							},
						},
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
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutateFn := MutateFuncFor(&tt.existing, &tt.desired)
			err := mutateFn()
			assert.Error(t, err)
		})
	}
}

func TestMutateDaemonsetLabelChange(t *testing.T) {
	tests := []struct {
		name     string
		existing appsv1.DaemonSet
		desired  appsv1.DaemonSet
	}{
		{
			name: "modified label in daemonset",
			existing: appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Now(),
					Name:              "daemonset",
				},
				Spec: appsv1.DaemonSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.daemonset",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.daemonset",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"user-label":                   "existing",
							},
						},
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
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.daemonset",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.daemonset",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"user-label":                   "desired",
							},
						},
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
			name: "new label in daemonset",
			existing: appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Now(),
					Name:              "daemonset",
				},
				Spec: appsv1.DaemonSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.daemonset",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.daemonset",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"user-label":                   "existing",
							},
						},
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
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.daemonset",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.daemonset",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"user-label":                   "existing",
								"new-user-label":               "desired",
							},
						},
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
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutateFn := MutateFuncFor(&tt.existing, &tt.desired)
			err := mutateFn()
			require.NoError(t, err)
			assert.Equal(t, tt.existing.Spec, tt.desired.Spec)
		})
	}
}

func TestMutateDeploymentLabelChange(t *testing.T) {
	tests := []struct {
		name     string
		existing appsv1.Deployment
		desired  appsv1.Deployment
	}{
		{
			name: "modified label in deployment",
			existing: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Now(),
					Name:              "deployment",
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.deployment",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.deployment",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"user-label":                   "existing",
							},
						},
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
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.deployment",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.deployment",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"user-label":                   "desired",
							},
						},
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
			name: "new label in deployment",
			existing: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Now(),
					Name:              "deployment",
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.deployment",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.deployment",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"user-label":                   "existing",
							},
						},
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
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.deployment",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.deployment",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"user-label":                   "existing",
								"new-user-label":               "desired",
							},
						},
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
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutateFn := MutateFuncFor(&tt.existing, &tt.desired)
			err := mutateFn()
			require.NoError(t, err)
			assert.Equal(t, tt.existing.Spec, tt.desired.Spec)
		})
	}
}

func TestMutateStatefulSetLabelChange(t *testing.T) {
	tests := []struct {
		name     string
		existing appsv1.StatefulSet
		desired  appsv1.StatefulSet
	}{
		{
			name: "modified label in statefulset",
			existing: appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Now(),
					Name:              "statefulset",
				},
				Spec: appsv1.StatefulSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.statefulset",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.statefulset",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"user-label":                   "existing",
							},
						},
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
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.statefulset",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.statefulset",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"user-label":                   "desired",
							},
						},
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
			name: "new label in statefulset",
			existing: appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					CreationTimestamp: metav1.Now(),
					Name:              "statefulset",
				},
				Spec: appsv1.StatefulSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.statefulset",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.statefulset",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"user-label":                   "existing",
							},
						},
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
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component":  "opentelemetry-collector",
							"app.kubernetes.io/instance":   "default.statefulset",
							"app.kubernetes.io/managed-by": "opentelemetry-operator",
							"app.kubernetes.io/part-of":    "opentelemetry",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "default.statefulset",
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"user-label":                   "existing",
								"new-user-label":               "desired",
							},
						},
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
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mutateFn := MutateFuncFor(&tt.existing, &tt.desired)
			err := mutateFn()
			require.NoError(t, err)
			assert.Equal(t, tt.existing.Spec, tt.desired.Spec)
		})
	}
}
