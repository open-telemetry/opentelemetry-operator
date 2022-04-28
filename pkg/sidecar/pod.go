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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

const (
	label = "sidecar.opentelemetry.io/injected"
)

// add a new sidecar container to the given pod, based on the given OpenTelemetryCollector.
func add(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector, pod corev1.Pod, attributes []corev1.EnvVar) (corev1.Pod, error) {
	// add the container
	volumes := collector.Volumes(cfg, otelcol)
	container := collector.Container(cfg, logger, otelcol)
	if !hasResourceAttributeEnvVar(container.Env) {
		container.Env = append(container.Env, attributes...)
	}
	pod.Spec.Containers = append(pod.Spec.Containers, container)
	pod.Spec.Volumes = append(pod.Spec.Volumes, volumes...)

	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[label] = fmt.Sprintf("%s.%s", otelcol.Namespace, otelcol.Name)

	return pod, nil
}

// remove the sidecar container from the given pod.
func remove(pod corev1.Pod) (corev1.Pod, error) {
	if !existsIn(pod) {
		return pod, nil
	}

	var containers []corev1.Container
	for _, container := range pod.Spec.Containers {
		if container.Name != naming.Container() {
			containers = append(containers, container)
		}
	}
	pod.Spec.Containers = containers
	return pod, nil
}

// existsIn checks whether a sidecar container exists in the given pod.
func existsIn(pod corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		if container.Name == naming.Container() {
			return true
		}
	}
	return false
}
