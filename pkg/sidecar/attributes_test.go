// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sidecar

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

func TestGetAttributesEnvNoPodReferences(t *testing.T) {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-ns",
		},
	}
	references := podReferences{}
	envs := getResourceAttributesEnv(ns, references)

	expectedEnv := []corev1.EnvVar{
		{
			Name: constants.EnvPodName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: constants.EnvPodUID,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.uid",
				},
			},
		},
		{
			Name: constants.EnvNodeName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		{
			Name: resourceAttributesEnvName,
			Value: fmt.Sprintf("%s=my-ns,%s=$(%s),%s=$(%s),%s=$(%s)",
				semconv.K8SNamespaceNameKey,
				semconv.K8SNodeNameKey,
				constants.EnvNodeName,
				semconv.K8SPodNameKey,
				constants.EnvPodName,
				semconv.K8SPodUIDKey,
				constants.EnvPodUID,
			),
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
	envs := getResourceAttributesEnv(ns, references)

	expectedEnv := []corev1.EnvVar{
		{
			Name: constants.EnvPodName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: constants.EnvPodUID,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.uid",
				},
			},
		},
		{
			Name: constants.EnvNodeName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		{
			Name: resourceAttributesEnvName,
			Value: fmt.Sprintf("%s=my-deployment,%s=uuid-dep,%s=my-ns,%s=$(%s),%s=$(%s),%s=$(%s),%s=my-replicaset,%s=uuid-replicaset",
				semconv.K8SDeploymentNameKey,
				semconv.K8SDeploymentUIDKey,
				semconv.K8SNamespaceNameKey,
				semconv.K8SNodeNameKey,
				constants.EnvNodeName,
				semconv.K8SPodNameKey,
				constants.EnvPodName,
				semconv.K8SPodUIDKey,
				constants.EnvPodUID,
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
		env      []corev1.EnvVar
		expected bool
	}{
		{
			"has-attributes",
			[]corev1.EnvVar{
				{
					Name: resourceAttributesEnvName,
					Value: fmt.Sprintf("%s=my-deployment,%s=uuid-dep,%s=my-ns,%s=$(%s),%s=$(%s),%s=$(%s),%s=my-replicaset,%s=uuid-replicaset",
						semconv.K8SDeploymentNameKey,
						semconv.K8SDeploymentUIDKey,
						semconv.K8SNamespaceNameKey,
						semconv.K8SNodeNameKey,
						constants.EnvNodeName,
						semconv.K8SPodNameKey,
						constants.EnvPodName,
						semconv.K8SPodUIDKey,
						constants.EnvPodUID,
						semconv.K8SReplicaSetNameKey,
						semconv.K8SReplicaSetUIDKey,
					),
				},
			},
			true,
		},

		{
			"does-not-have-attributes",
			[]corev1.EnvVar{
				{
					Name:  "other_env",
					Value: "other_value",
				},
			},
			false,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			assert.Equal(t, tt.expected, hasResourceAttributeEnvVar(tt.env))
		})
	}
}
