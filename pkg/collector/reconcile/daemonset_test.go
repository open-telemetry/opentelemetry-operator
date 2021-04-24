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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

func TestExpectedDaemonsets(t *testing.T) {
	param := params()
	expectedDs := collector.DaemonSet(param.Config, logger, param.Instance)

	t.Run("should create Daemonset", func(t *testing.T) {
		err := expectedDaemonSets(context.Background(), param, []v1.DaemonSet{expectedDs})
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &v1.DaemonSet{}, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)

	})
	t.Run("should update Daemonset", func(t *testing.T) {
		ds := daemonset("test-collector")
		createObjectIfNotExists(t, "test-collector", &ds)
		//err := expectedDaemonSets(context.Background(), param, []v1.DaemonSet{expectedDs})
		//assert.NoError(t, err)
		//
		//actual := v1.DaemonSet{}
		//exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})
		//
		//assert.NoError(t, err)
		//assert.True(t, exists)
		//assert.Equal(t, expectedDs, actual)

	})

	t.Run("should cleanup daemonsets", func(t *testing.T) {

		ds := daemonset("dummy")
		createObjectIfNotExists(t, "dummy", &ds)

		err := deleteDaemonSets(context.Background(), param, []v1.DaemonSet{expectedDs})
		assert.NoError(t, err)

		actual := v1.DaemonSet{}
		exists, _ := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "dummy"})

		assert.False(t, exists)

	})
}

func daemonset(name string) v1.DaemonSet {
	return v1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
				"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params().Instance.Namespace, params().Instance.Name),
			},
		},
		Spec: v1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "dummy",
						Image: "busybox",
					}},
				},
			},
		},
	}
}
