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
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

const (
	injectedLabel = "sidecar.opentelemetry.io/injected"
	confEnvVar    = "OTEL_CONFIG"
)

// add a new sidecar container to the given pod, based on the given OpenTelemetryCollector.
func add(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector, pod corev1.Pod, attributes []corev1.EnvVar) (corev1.Pod, error) {
	otelColCfg, err := collector.ReplaceConfig(otelcol)
	if err != nil {
		return pod, err
	}

	container := collector.Container(cfg, logger, otelcol, false)
	container.Args = append(container.Args, fmt.Sprintf("--config=env:%s", confEnvVar))

	container.Env = append(container.Env, corev1.EnvVar{Name: confEnvVar, Value: otelColCfg})
	if !hasResourceAttributeEnvVar(container.Env) {
		container.Env = append(container.Env, attributes...)
	}
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, otelcol.Spec.InitContainers...)
	pod.Spec.Containers = append(pod.Spec.Containers, container)
	pod.Spec.Volumes = append(pod.Spec.Volumes, otelcol.Spec.Volumes...)

	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[injectedLabel] = naming.Truncate("%s.%s", 63, otelcol.Namespace, otelcol.Name)

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
