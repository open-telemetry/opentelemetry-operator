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

package reconcile

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator"
)

func TestExpectedDeployments(t *testing.T) {
	param := params()
	expectedDeploy := targetallocator.Deployment(param.Config, logger, param.Instance)

	t.Run("should create deployment", func(t *testing.T) {
		err := expectedDeployments(context.Background(), param, []v1.Deployment{expectedDeploy})
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &v1.Deployment{}, types.NamespacedName{Namespace: "default", Name: "test-targetallocator"})

		assert.NoError(t, err)
		assert.True(t, exists)

	})

	t.Run("should not create deployment when otel collector mode is not StatefulSet", func(t *testing.T) {
		modes := []v1alpha1.Mode{v1alpha1.ModeDaemonSet, v1alpha1.ModeDeployment, v1alpha1.ModeSidecar}

		for _, mode := range modes {
			newParam := Params{
				Client: k8sClient,
				Instance: v1alpha1.OpenTelemetryCollector{
					TypeMeta: metav1.TypeMeta{
						Kind:       "opentelemetry.io",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "default",
						UID:       instanceUID,
					},
					Spec: v1alpha1.OpenTelemetryCollectorSpec{
						Mode: mode,
						TargetAllocator: v1alpha1.OpenTelemetryTargetAllocator{
							Enabled: true,
						},
						Config: `
				receivers:
				jaeger:
					protocols:
					grpc:
				processors:
			
				exporters:
				logging:
			
				service:
				pipelines:
					traces:
					receivers: [jaeger]
					processors: []
					exporters: [logging]
			
			`,
					},
				},
				Scheme: testScheme,
				Log:    logger,
			}
			expected := []v1.Deployment{}
			if newParam.Instance.Spec.Mode == v1alpha1.ModeStatefulSet {
				expected = append(expected, targetallocator.Deployment(newParam.Config, newParam.Log, newParam.Instance))
			}

			assert.Len(t, expected, 0)
		}
	})

	t.Run("should not create deployment when targetallocator is not enabled", func(t *testing.T) {
		newParam := Params{
			Client: k8sClient,
			Instance: v1alpha1.OpenTelemetryCollector{
				TypeMeta: metav1.TypeMeta{
					Kind:       "opentelemetry.io",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
					UID:       instanceUID,
				},
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode: v1alpha1.ModeStatefulSet,
					Config: `
				receivers:
				jaeger:
					protocols:
					grpc:
				processors:
			
				exporters:
				logging:
			
				service:
				pipelines:
					traces:
					receivers: [jaeger]
					processors: []
					exporters: [logging]
			
			`,
				},
			},
			Scheme: testScheme,
			Log:    logger,
		}
		expected := []v1.Deployment{}
		if newParam.Instance.Spec.TargetAllocator.Enabled {
			expected = append(expected, targetallocator.Deployment(newParam.Config, newParam.Log, newParam.Instance))
		}

		assert.Len(t, expected, 0)
	})

	t.Run("should not update deployment container when the config is updated", func(t *testing.T) {
		ctx := context.Background()
		createObjectIfNotExists(t, "test-targetallocator", &expectedDeploy)
		orgUID := expectedDeploy.OwnerReferences[0].UID

		updatedDeploy := targetallocator.Deployment(newParams().Config, logger, param.Instance)

		err := expectedDeployments(ctx, param, []v1.Deployment{updatedDeploy})
		assert.NoError(t, err)

		actual := v1.Deployment{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-targetallocator"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, orgUID, actual.OwnerReferences[0].UID)
		assert.Equal(t, expectedDeploy.Spec.Template.Spec.Containers[0], actual.Spec.Template.Spec.Containers[0])
		assert.Equal(t, int32(1), *actual.Spec.Replicas)
	})

	t.Run("should update deployment container when the container image is updated", func(t *testing.T) {
		ctx := context.Background()
		createObjectIfNotExists(t, "test-targetallocator", &expectedDeploy)
		orgUID := expectedDeploy.OwnerReferences[0].UID

		updatedParam := newParams("test/test-img")
		updatedDeploy := targetallocator.Deployment(updatedParam.Config, logger, updatedParam.Instance)

		err := expectedDeployments(ctx, param, []v1.Deployment{updatedDeploy})
		assert.NoError(t, err)

		actual := v1.Deployment{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-targetallocator"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, orgUID, actual.OwnerReferences[0].UID)
		assert.NotEqual(t, expectedDeploy.Spec.Template.Spec.Containers[0], actual.Spec.Template.Spec.Containers[0])
		assert.Equal(t, int32(1), *actual.Spec.Replicas)
	})

	t.Run("should delete deployment", func(t *testing.T) {
		labels := map[string]string{
			"app.kubernetes.io/instance":   "test.targetallocator",
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}
		deploy := v1.Deployment{}
		deploy.Name = "dummy"
		deploy.Namespace = "default"
		deploy.Labels = labels
		deploy.Spec = v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "dummy",
						Image: "busybox",
					}},
				},
			},
		}
		createObjectIfNotExists(t, "dummy", &deploy)

		err := deleteDeployments(context.Background(), param, []v1.Deployment{expectedDeploy})
		assert.NoError(t, err)

		actual := v1.Deployment{}
		exists, _ := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "dummy"})

		assert.False(t, exists)

	})

	t.Run("should not delete deployment", func(t *testing.T) {
		labels := map[string]string{
			"app.kubernetes.io/instance":   "default.test",
			"app.kubernetes.io/managed-by": "helm-opentelemetry-operator",
		}
		deploy := v1.Deployment{}
		deploy.Name = "dummy"
		deploy.Namespace = "default"
		deploy.Spec = v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "dummy",
						Image: "busybox",
					}},
				},
			},
		}
		createObjectIfNotExists(t, "dummy", &deploy)

		err := deleteDeployments(context.Background(), param, []v1.Deployment{expectedDeploy})
		assert.NoError(t, err)

		actual := v1.Deployment{}
		exists, _ := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "dummy"})

		assert.True(t, exists)

	})
}
