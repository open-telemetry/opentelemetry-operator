// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package sidecar contains operations related to sidecar manipulation (Add, update, remove).
package sidecar

import (
	"fmt"
	"sort"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
)

const resourceAttributesEnvName = "OTEL_RESOURCE_ATTRIBUTES"

type podReferences struct {
	replicaset *appsv1.ReplicaSet
	deployment *appsv1.Deployment
}

// getResourceAttributesEnv returns a list of environment variables. The list contains OTEL_RESOURCE_ATTRIBUTES and additional environment variables that use Kubernetes downward API to read pod specification.
// see: https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/
func getResourceAttributesEnv(ns corev1.Namespace, podReferences podReferences) []corev1.EnvVar {

	var envvars []corev1.EnvVar

	attributes := map[attribute.Key]string{
		semconv.K8SPodNameKey:       fmt.Sprintf("$(%s)", constants.EnvPodName),
		semconv.K8SPodUIDKey:        fmt.Sprintf("$(%s)", constants.EnvPodUID),
		semconv.K8SNodeNameKey:      fmt.Sprintf("$(%s)", constants.EnvNodeName),
		semconv.K8SNamespaceNameKey: ns.Name,
	}

	if podReferences.deployment != nil {
		attributes[semconv.K8SDeploymentUIDKey] = string(podReferences.deployment.UID)
		attributes[semconv.K8SDeploymentNameKey] = string(podReferences.deployment.Name)
	}

	if podReferences.replicaset != nil {
		attributes[semconv.K8SReplicaSetUIDKey] = string(podReferences.replicaset.UID)
		attributes[semconv.K8SReplicaSetNameKey] = string(podReferences.replicaset.Name)
	}

	envvars = append(envvars, corev1.EnvVar{
		Name: constants.EnvPodName,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	})

	envvars = append(envvars, corev1.EnvVar{
		Name: constants.EnvPodUID,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.uid",
			},
		},
	})

	envvars = append(envvars, corev1.EnvVar{
		Name: constants.EnvNodeName,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "spec.nodeName",
			},
		},
	})

	envvars = append(envvars, corev1.EnvVar{
		Name:  resourceAttributesEnvName,
		Value: mapToValue(attributes),
	})

	return envvars
}

func mapToValue(attributesMap map[attribute.Key]string) string {
	var parts []string

	// Sort it to make it predictable
	keys := make([]string, 0, len(attributesMap))
	for k := range attributesMap {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)

	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, attributesMap[attribute.Key(key)]))
	}
	return strings.Join(parts, ",")
}

// check if container doesn't have already the OTEL_RESOURCE_ATTRIBUTES, we don't want to override it if it's already specified.
func hasResourceAttributeEnvVar(envvars []corev1.EnvVar) bool {
	for _, env := range envvars {
		if env.Name == resourceAttributesEnvName {
			return true
		}
	}
	return false
}
