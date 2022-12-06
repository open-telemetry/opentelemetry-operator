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

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator"
)

func TestExpectedDeployments(t *testing.T) {
	param := params()
	expectedDeploy := collector.Deployment(param.Config, logger, param.Instance)
	expectedTADeploy := targetallocator.Deployment(param.Config, logger, param.Instance)

	t.Run("should create collector deployment", func(t *testing.T) {
		err := expectedDeployments(context.Background(), param, []v1.Deployment{expectedDeploy})
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &v1.Deployment{}, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)

	})

	t.Run("should create target allocator deployment", func(t *testing.T) {
		err := expectedDeployments(context.Background(), param, []v1.Deployment{expectedTADeploy})
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &v1.Deployment{}, types.NamespacedName{Namespace: "default", Name: "test-targetallocator"})

		assert.NoError(t, err)
		assert.True(t, exists)

	})

	t.Run("should not create target allocator deployment when targetallocator is not enabled", func(t *testing.T) {
		paramTargetAllocator := Params{
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
			Log: logger,
		}
		expected := []v1.Deployment{}
		if paramTargetAllocator.Instance.Spec.TargetAllocator.Enabled {
			expected = append(expected, targetallocator.Deployment(paramTargetAllocator.Config, paramTargetAllocator.Log, paramTargetAllocator.Instance))
		}

		assert.Len(t, expected, 0)
	})

	t.Run("should update target allocator deployment when the prometheusCR is updated", func(t *testing.T) {
		ctx := context.Background()
		createObjectIfNotExists(t, "test-targetallocator", &expectedTADeploy)
		orgUID := expectedTADeploy.OwnerReferences[0].UID

		updatedParam, err := newParams(expectedTADeploy.Spec.Template.Spec.Containers[0].Image, "")
		assert.NoError(t, err)
		updatedParam.Instance.Spec.TargetAllocator.PrometheusCR.Enabled = true
		updatedDeploy := targetallocator.Deployment(updatedParam.Config, logger, updatedParam.Instance)

		err = expectedDeployments(ctx, param, []v1.Deployment{updatedDeploy})
		assert.NoError(t, err)

		actual := v1.Deployment{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-targetallocator"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, orgUID, actual.OwnerReferences[0].UID)
		assert.ElementsMatch(t, actual.Spec.Template.Spec.Containers[0].Args, []string{"--enable-prometheus-cr-watcher"})
		assert.Equal(t, int32(1), *actual.Spec.Replicas)
	})

	t.Run("should not update target allocator deployment replicas when collector max replicas is set", func(t *testing.T) {
		replicas, maxReplicas := int32(2), int32(10)
		oneReplica := int32(1)
		paramMaxReplicas := Params{
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
					MaxReplicas: &maxReplicas,
					Replicas:    &replicas,
					Mode:        v1alpha1.ModeStatefulSet,
					TargetAllocator: v1alpha1.OpenTelemetryTargetAllocator{
						Enabled:  true,
						Replicas: &oneReplica,
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
			Log: logger,
		}
		expected := []v1.Deployment{}
		allocator := targetallocator.Deployment(paramMaxReplicas.Config, paramMaxReplicas.Log, paramMaxReplicas.Instance)
		expected = append(expected, allocator)

		assert.Len(t, expected, 1)
		assert.Equal(t, *allocator.Spec.Replicas, int32(1))
	})

	t.Run("should update target allocator deployment replicas when changed", func(t *testing.T) {
		initialReplicas, nextReplicas := int32(1), int32(2)
		paramReplicas := Params{
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
					Replicas: &initialReplicas,
					Mode:     v1alpha1.ModeStatefulSet,
					TargetAllocator: v1alpha1.OpenTelemetryTargetAllocator{
						Enabled:  true,
						Replicas: &initialReplicas,
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
			Log: logger,
		}
		expected := []v1.Deployment{}
		allocator := targetallocator.Deployment(paramReplicas.Config, paramReplicas.Log, paramReplicas.Instance)
		expected = append(expected, allocator)

		assert.Len(t, expected, 1)
		assert.Equal(t, *allocator.Spec.Replicas, int32(1))
		param.Instance.Spec.TargetAllocator.Replicas = &nextReplicas
		finalAllocator := targetallocator.Deployment(param.Config, param.Log, param.Instance)
		assert.Equal(t, *finalAllocator.Spec.Replicas, int32(2))
	})

	t.Run("should update deployment", func(t *testing.T) {
		createObjectIfNotExists(t, "test-collector", &expectedDeploy)
		err := expectedDeployments(context.Background(), param, []v1.Deployment{expectedDeploy})
		assert.NoError(t, err)

		actual := v1.Deployment{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)
		assert.Equal(t, int32(2), *actual.Spec.Replicas)
	})

	t.Run("should update target allocator deployment when the container image is updated", func(t *testing.T) {
		ctx := context.Background()
		createObjectIfNotExists(t, "test-targetallocator", &expectedTADeploy)
		orgUID := expectedTADeploy.OwnerReferences[0].UID

		updatedParam, err := newParams("test/test-img", "")
		assert.NoError(t, err)
		updatedDeploy := targetallocator.Deployment(updatedParam.Config, logger, updatedParam.Instance)

		err = expectedDeployments(ctx, param, []v1.Deployment{updatedDeploy})
		assert.NoError(t, err)

		actual := v1.Deployment{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-targetallocator"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, orgUID, actual.OwnerReferences[0].UID)
		assert.NotEqual(t, expectedTADeploy.Spec.Template.Spec.Containers[0].Image, actual.Spec.Template.Spec.Containers[0].Image)
		assert.Equal(t, int32(1), *actual.Spec.Replicas)
	})

	t.Run("should delete deployment", func(t *testing.T) {
		labels := map[string]string{
			"app.kubernetes.io/instance":   "default.test",
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

	t.Run("change Spec.Selector should recreate deployment", func(t *testing.T) {

		oldDeploy := collector.Deployment(param.Config, logger, param.Instance)
		oldDeploy.Spec.Selector.MatchLabels["app.kubernetes.io/version"] = "latest"
		oldDeploy.Spec.Template.Labels["app.kubernetes.io/version"] = "latest"
		oldDeploy.Name = "update-deploy"

		err := expectedDeployments(context.Background(), param, []v1.Deployment{oldDeploy})
		assert.NoError(t, err)
		exists, err := populateObjectIfExists(t, &v1.Deployment{}, types.NamespacedName{Namespace: "default", Name: oldDeploy.Name})
		assert.NoError(t, err)
		assert.True(t, exists)

		newDeploy := collector.Deployment(param.Config, logger, param.Instance)
		newDeploy.Name = oldDeploy.Name
		err = expectedDeployments(context.Background(), param, []v1.Deployment{newDeploy})
		assert.NoError(t, err)
		exists, err = populateObjectIfExists(t, &v1.Deployment{}, types.NamespacedName{Namespace: "default", Name: oldDeploy.Name})
		assert.NoError(t, err)
		assert.False(t, exists)

		err = expectedDeployments(context.Background(), param, []v1.Deployment{newDeploy})
		assert.NoError(t, err)
		actual := v1.Deployment{}
		exists, err = populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: oldDeploy.Name})
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, newDeploy.Spec.Selector.MatchLabels, actual.Spec.Selector.MatchLabels)
	})
}

func TestCurrentReplicasWithHPA(t *testing.T) {
	minReplicas := int32(2)
	maxReplicas := int32(5)
	spec := v1alpha1.OpenTelemetryCollectorSpec{
		Replicas:    &minReplicas,
		MaxReplicas: &maxReplicas,
	}

	res := currentReplicasWithHPA(spec, 10)
	assert.Equal(t, int32(5), res)

	res = currentReplicasWithHPA(spec, 1)
	assert.Equal(t, int32(2), res)

	res = currentReplicasWithHPA(spec, 3)
	assert.Equal(t, int32(3), res)
}
