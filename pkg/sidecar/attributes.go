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

// Package sidecar contains operations related to sidecar manipulation (Add, update, remove).
package sidecar

import (
	"fmt"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const resourceAttributesEnvName = "OTEL_RESOURCE_ATTRIBUTES"

type podReferences struct {
	replicaset *appsv1.ReplicaSet
	deployment *appsv1.Deployment
}

// add resource attributes environment variables. and OTEL_RESOURCE_ATTRIBUTES if not exists.
func getAttributesEnv(ns corev1.Namespace, podReferences podReferences) []corev1.EnvVar {

	var envvars []corev1.EnvVar

	attributes := map[string]string{
		"k8s.pod.name":       "$(POD_NAME)",
		"k8s.pod.uid":        "$(POD_UID)",
		"k8s.node.name":      "$(NODE_NAME)",
		"k8s.namespace.name": ns.Name,
	}

	if podReferences.deployment != nil {
		attributes["k8s.deployment.uid"] = string(podReferences.deployment.UID)
		attributes["k8s.deployment.name"] = string(podReferences.deployment.Name)
	}

	if podReferences.replicaset != nil {
		attributes["k8s.replicaset.uid"] = string(podReferences.replicaset.UID)
		attributes["k8s.replicaset.name"] = string(podReferences.replicaset.Name)
	}

	envvars = append(envvars, corev1.EnvVar{
		Name: "NODE_NAME",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "spec.nodeName",
			},
		},
	})

	envvars = append(envvars, corev1.EnvVar{
		Name: "POD_UID",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.uid",
			},
		},
	})
	envvars = append(envvars, corev1.EnvVar{
		Name:  resourceAttributesEnvName,
		Value: mapToValue(attributes),
	})

	return envvars
}

func mapToValue(attribues map[string]string) string {
	var parts []string

	// Sort it to make it predictable
	keys := make([]string, 0, len(attribues))
	for k := range attribues {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, attribues[key]))
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
