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

// Package sidecar contains operations related to sidecar manipulation (add, update, remove).
package sidecar

import (
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/signalfx/splunk-otel-operator/api/v1alpha1"
	"github.com/signalfx/splunk-otel-operator/internal/config"
	"github.com/signalfx/splunk-otel-operator/pkg/collector"
	"github.com/signalfx/splunk-otel-operator/pkg/naming"
)

const (
	label = "sidecar.splunk.com/injected"
)

// Add a new sidecar container to the given pod, based on the given SplunkOtelAgent.
func Add(cfg config.Config, logger logr.Logger, otelcol v1alpha1.SplunkOtelAgent, pod corev1.Pod) (corev1.Pod, error) {
	// add the container
	volumes := collector.Volumes(cfg, otelcol)
	container := collector.Container(cfg, logger, otelcol)
	pod.Spec.Containers = append(pod.Spec.Containers, container)
	pod.Spec.Volumes = append(pod.Spec.Volumes, volumes...)

	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[label] = fmt.Sprintf("%s.%s", otelcol.Namespace, otelcol.Name)

	return pod, nil
}

// Remove the sidecar container from the given pod.
func Remove(pod corev1.Pod) (corev1.Pod, error) {
	if !ExistsIn(pod) {
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

// ExistsIn checks whether a sidecar container exists in the given pod.
func ExistsIn(pod corev1.Pod) bool {
	for _, container := range pod.Spec.Containers {
		if container.Name == naming.Container() {
			return true
		}
	}
	return false
}
