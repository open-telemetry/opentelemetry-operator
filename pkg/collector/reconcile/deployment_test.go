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

func TestExpectedDeployments(t *testing.T) {
	param := params()
	expectedDeploy := collector.Deployment(param.Config, logger, param.Instance)

	t.Run("should create deployment", func(t *testing.T) {
		err := expectedDeployments(context.Background(), param, []v1.Deployment{expectedDeploy})
		assert.NoError(t, err)

		exists, err := populateObjectIfExists(t, &v1.Deployment{}, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)

	})
	t.Run("should update deployment", func(t *testing.T) {
		deploy := deployment("test-collector")
		createObjectIfNotExists(t, "test-collector", &deploy)
		err := expectedDeployments(context.Background(), param, []v1.Deployment{expectedDeploy})
		assert.NoError(t, err)

		actual := v1.Deployment{}
		exists, err := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "test-collector"})

		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, instanceUID, actual.OwnerReferences[0].UID)

	})

	t.Run("should cleanup deployments", func(t *testing.T) {

		deploy := deployment("dummy")
		createObjectIfNotExists(t, "dummy", &deploy)

		err := deleteDeployments(context.Background(), param, []v1.Deployment{expectedDeploy})
		assert.NoError(t, err)

		actual := v1.Deployment{}
		exists, _ := populateObjectIfExists(t, &actual, types.NamespacedName{Namespace: "default", Name: "dummy"})

		assert.False(t, exists)

	})
}

func deployment(name string) v1.Deployment {
	labels := collector.Labels(params().Instance)
	labels["app.kubernetes.io/name"] = name
	return v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
				"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params().Instance.Namespace, params().Instance.Name),
			},
		},
		Spec: v1.DeploymentSpec{
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
		},
	}
}
