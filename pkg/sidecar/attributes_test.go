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

package sidecar

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetAttributesEnvNoPodReferences(t *testing.T) {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-ns",
		},
	}
	references := podReferences{}
	envs := getAttributesEnv(ns, references)

	expectedEnv := []corev1.EnvVar{
		{
			Name: "NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		{
			Name: "POD_UID",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.uid",
				},
			},
		},
		{
			Name: resourceAttributesEnvName,
			Value: fmt.Sprintf("%s=my-ns,%s=$(NODE_NAME),%s=$(POD_NAME),%s=$(POD_UID)",
				semconv.K8SNamespaceNameKey, semconv.K8SNodeNameKey, semconv.K8SPodNameKey, semconv.K8SPodUIDKey),
		},
	}

	assert.Equal(t, expectedEnv, envs)
}

func TestGetAttributesEnvWithPodReferences(t *testing.T) {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-ns",
		},
	}
	references := podReferences{
		deployment: &appv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-deployment",
				UID:  "uuid-dep",
			},
		},
		replicaset: &appv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-replicaset",
				UID:  "uuid-replicaset",
			},
		},
	}
	envs := getAttributesEnv(ns, references)

	expectedEnv := []corev1.EnvVar{
		{
			Name: "NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		{
			Name: "POD_UID",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.uid",
				},
			},
		},
		{
			Name: resourceAttributesEnvName,
			Value: fmt.Sprintf("%s=my-deployment,%s=uuid-dep,%s=my-ns,%s=$(NODE_NAME),%s=$(POD_NAME),%s=$(POD_UID),%s=my-replicaset,%s=uuid-replicaset",
				semconv.K8SDeploymentNameKey,
				semconv.K8SDeploymentUIDKey,
				semconv.K8SNamespaceNameKey,
				semconv.K8SNodeNameKey,
				semconv.K8SPodNameKey,
				semconv.K8SPodUIDKey,
				semconv.K8SReplicaSetNameKey,
				semconv.K8SReplicaSetUIDKey,
			),
		},
	}

	assert.Equal(t, expectedEnv, envs)
}

func TestHasResourceAttributeEnvVar(t *testing.T) {
	for _, tt := range []struct {
		desc     string
		expected bool
		env      []corev1.EnvVar
	}{
		{
			"has-attributes", true, []corev1.EnvVar{
				{
					Name: resourceAttributesEnvName,
					Value: fmt.Sprintf("%s=my-deployment,%s=uuid-dep,%s=my-ns,%s=$(NODE_NAME),%s=$(POD_NAME),%s=$(POD_UID),%s=my-replicaset,%s=uuid-replicaset",
						semconv.K8SDeploymentNameKey,
						semconv.K8SDeploymentUIDKey,
						semconv.K8SNamespaceNameKey,
						semconv.K8SNodeNameKey,
						semconv.K8SPodNameKey,
						semconv.K8SPodUIDKey,
						semconv.K8SReplicaSetNameKey,
						semconv.K8SReplicaSetUIDKey,
					),
				},
			},
		},

		{
			"does-not-have-attributes", false, []corev1.EnvVar{
				{
					Name:  "other_env",
					Value: "other_value",
				},
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			assert.Equal(t, tt.expected, hasResourceAttributeEnvVar(tt.env))
		})
	}
}
