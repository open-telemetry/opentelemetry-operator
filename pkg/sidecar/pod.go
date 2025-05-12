// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package sidecar contains operations related to sidecar manipulation (Add, update, remove).
package sidecar

import (
	"fmt"
	"slices"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

const (
	injectedLabel = "sidecar.opentelemetry.io/injected"
	confEnvVar    = "OTEL_CONFIG"
)

// add a new sidecar container to the given pod, based on the given OpenTelemetryCollector.
func add(cfg config.Config, logger logr.Logger, otelcol v1beta1.OpenTelemetryCollector, pod corev1.Pod, attributes []corev1.EnvVar) (corev1.Pod, error) {
	otelColCfg, err := collector.ReplaceConfig(otelcol, nil)
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

	if featuregate.EnableNativeSidecarContainers.IsEnabled() {
		policy := corev1.ContainerRestartPolicyAlways
		container.RestartPolicy = &policy
		// NOTE: Use ReadinessProbe as startup probe.
		// See https://github.com/open-telemetry/opentelemetry-operator/pull/2801#discussion_r1547571121
		container.StartupProbe = container.ReadinessProbe
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, container)
	} else {
		pod.Spec.Containers = append(pod.Spec.Containers, container)
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, otelcol.Spec.Volumes...)

	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[injectedLabel] = naming.Truncate("%s.%s", 63, otelcol.Namespace, otelcol.Name)

	return pod, nil
}

func isOtelColContainer(c corev1.Container) bool { return c.Name == naming.Container() }

// remove the sidecar container from the given pod.
func remove(pod corev1.Pod) corev1.Pod {
	if !existsIn(pod) {
		return pod
	}

	pod.Spec.Containers = slices.DeleteFunc(pod.Spec.Containers, isOtelColContainer)

	if featuregate.EnableNativeSidecarContainers.IsEnabled() {
		// NOTE: we also remove init containers (native sidecars) since k8s 1.28.
		// This should have no side effects.
		pod.Spec.InitContainers = slices.DeleteFunc(pod.Spec.InitContainers, isOtelColContainer)
	}
	return pod
}

// existsIn checks whether a sidecar container exists in the given pod.
func existsIn(pod corev1.Pod) bool {
	if slices.ContainsFunc(pod.Spec.Containers, isOtelColContainer) {
		return true
	}

	if featuregate.EnableNativeSidecarContainers.IsEnabled() {
		// NOTE: we also check init containers (native sidecars) since k8s 1.28.
		// This should have no side effects.
		if slices.ContainsFunc(pod.Spec.InitContainers, isOtelColContainer) {
			return true
		}
	}
	return false
}
