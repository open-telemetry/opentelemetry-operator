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

	"github.com/signalfx/splunk-otel-operator/pkg/collector"
)

func TestExpectedStatefulsets(t *testing.T) {
	param := params()
	expectedSs := collector.StatefulSet(param.Config, logger, param.Instance)

	t.Run("should create StatefulSet", func(t *testing.T) {
		err := expectedStatefulSets(context.Background(), param, []v1.StatefulSet{expectedSs})
		assert.NoError(t, err)

		actual := v1.StatefulSet{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, int32(2), *actual.Spec.Replicas)

	})
	t.Run("should update statefulset", func(t *testing.T) {
		createObjectIfNotExists(t, "test-collector", &expectedSs)
		err := expectedStatefulSets(context.Background(), param, []v1.StatefulSet{expectedSs})
		assert.NoError(t, err)

		actual := v1.StatefulSet{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)
	})

	t.Run("should delete statefulset", func(t *testing.T) {

		labels := map[string]string{
			"app.kubernetes.io/instance":   "default.test",
			"app.kubernetes.io/managed-by": "splunk-otel-operator",
		}
		ds := v1.StatefulSet{}
		ds.Name = "dummy"
		ds.Namespace = "default"
		ds.Labels = labels
		ds.Spec = v1.StatefulSetSpec{
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

		createObjectIfNotExists(t, "dummy", &ds)

		err := deleteStatefulSets(context.Background(), param, []v1.StatefulSet{expectedSs})
		assert.NoError(t, err)

		actual := v1.StatefulSet{}
		exists, _ := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "dummy"})

		assert.False(t, exists)

	})

	t.Run("should not delete statefulset", func(t *testing.T) {

		labels := map[string]string{
			"app.kubernetes.io/managed-by": "helm-splunk-otel-operator",
		}
		ds := v1.StatefulSet{}
		ds.Name = "dummy"
		ds.Namespace = "default"
		ds.Labels = labels
		ds.Spec = v1.StatefulSetSpec{
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

		createObjectIfNotExists(t, "dummy", &ds)

		err := deleteStatefulSets(context.Background(), param, []v1.StatefulSet{expectedSs})
		assert.NoError(t, err)

		actual := v1.StatefulSet{}
		exists, _ := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "dummy"})

		assert.True(t, exists)

	})
}
