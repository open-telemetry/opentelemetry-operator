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

package targetallocator

import (
	"strings"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

const (
	targetAllocatorFilename = "targetallocator.yaml"
)

func ConfigMap(params manifests.Params) (*corev1.ConfigMap, error) {
	otelcol := params.OtelCol

	name := naming.TAConfigMap(otelcol.Name)
	version := strings.Split(otelcol.Spec.Image, ":")
	labels := Labels(otelcol, name)
	if len(version) > 1 {
		labels["app.kubernetes.io/version"] = version[len(version)-1]
	} else {
		labels["app.kubernetes.io/version"] = "latest"
	}

	// Collector supports environment variable substitution, but the TA does not.
	// TA ConfigMap should have a single "$", as it does not support env var substitution
	prometheusReceiverConfig, err := adapters.UnescapeDollarSignsInPromConfig(otelcol.Spec.Config)
	if err != nil {
		return &corev1.ConfigMap{}, err
	}

	taConfig := make(map[interface{}]interface{})
	prometheusCRConfig := make(map[interface{}]interface{})
	taConfig["label_selector"] = collector.SelectorLabels(otelcol)
	// We only take the "config" from the returned object, if it's present
	if prometheusConfig, ok := prometheusReceiverConfig["config"]; ok {
		taConfig["config"] = prometheusConfig
	}

	if len(otelcol.Spec.TargetAllocator.AllocationStrategy) > 0 {
		taConfig["allocation_strategy"] = otelcol.Spec.TargetAllocator.AllocationStrategy
	} else {
		taConfig["allocation_strategy"] = v1alpha1.OpenTelemetryTargetAllocatorAllocationStrategyLeastWeighted
	}

	if len(otelcol.Spec.TargetAllocator.FilterStrategy) > 0 {
		taConfig["filter_strategy"] = otelcol.Spec.TargetAllocator.FilterStrategy
	}

	if otelcol.Spec.TargetAllocator.PrometheusCR.ScrapeInterval.Size() > 0 {
		prometheusCRConfig["scrape_interval"] = otelcol.Spec.TargetAllocator.PrometheusCR.ScrapeInterval.Duration
	}

	if otelcol.Spec.TargetAllocator.PrometheusCR.ServiceMonitorSelector != nil {
		taConfig["service_monitor_selector"] = &otelcol.Spec.TargetAllocator.PrometheusCR.ServiceMonitorSelector
	}

	if otelcol.Spec.TargetAllocator.PrometheusCR.PodMonitorSelector != nil {
		taConfig["pod_monitor_selector"] = &otelcol.Spec.TargetAllocator.PrometheusCR.PodMonitorSelector
	}

	if len(prometheusCRConfig) > 0 {
		taConfig["prometheus_cr"] = prometheusCRConfig
	}

	taConfigYAML, err := yaml.Marshal(taConfig)
	if err != nil {
		return &corev1.ConfigMap{}, err
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   otelcol.Namespace,
			Labels:      labels,
			Annotations: otelcol.Annotations,
		},
		Data: map[string]string{
			targetAllocatorFilename: string(taConfigYAML),
		},
	}, nil
}
