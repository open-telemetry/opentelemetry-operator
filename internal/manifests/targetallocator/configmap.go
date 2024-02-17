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
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

const (
	targetAllocatorFilename = "targetallocator.yaml"
)

func ConfigMap(params manifests.Params) (*corev1.ConfigMap, error) {
	name := naming.TAConfigMap(params.OtelCol.Name)
	labels := Labels(params.OtelCol, name)
	// TODO: https://github.com/open-telemetry/opentelemetry-operator/issues/2603
	cfgStr, err := params.OtelCol.Spec.Config.Yaml()
	if err != nil {
		return &corev1.ConfigMap{}, err
	}

	// Collector supports environment variable substitution, but the TA does not.
	// TA ConfigMap should have a single "$", as it does not support env var substitution
	prometheusReceiverConfig, err := adapters.UnescapeDollarSignsInPromConfig(cfgStr)
	if err != nil {
		return &corev1.ConfigMap{}, err
	}

	taConfig := make(map[interface{}]interface{})
	prometheusCRConfig := make(map[interface{}]interface{})
	collectorSelectorLabels := manifestutils.SelectorLabels(params.OtelCol.ObjectMeta, collector.ComponentOpenTelemetryCollector)
	taConfig["collector_selector"] = map[string]any{
		"matchlabels": collectorSelectorLabels,
	}
	// The below instruction is here for compatibility with the previous target allocator version
	// TODO: Drop it after 3 more versions
	taConfig["label_selector"] = collectorSelectorLabels
	// We only take the "config" from the returned object, if it's present
	if prometheusConfig, ok := prometheusReceiverConfig["config"]; ok {
		taConfig["config"] = prometheusConfig
	}

	if len(params.OtelCol.Spec.TargetAllocator.AllocationStrategy) > 0 {
		taConfig["allocation_strategy"] = params.OtelCol.Spec.TargetAllocator.AllocationStrategy
	} else {
		taConfig["allocation_strategy"] = v1beta1.TargetAllocatorAllocationStrategyConsistentHashing
	}

	taConfig["filter_strategy"] = params.OtelCol.Spec.TargetAllocator.FilterStrategy

	if params.OtelCol.Spec.TargetAllocator.PrometheusCR.ScrapeInterval.Size() > 0 {
		prometheusCRConfig["scrape_interval"] = params.OtelCol.Spec.TargetAllocator.PrometheusCR.ScrapeInterval.Duration
	}

	prometheusCRConfig["service_monitor_selector"] = params.OtelCol.Spec.TargetAllocator.PrometheusCR.ServiceMonitorSelector

	// The below instruction is here for compatibility with the previous target allocator version
	// TODO: Drop it after 3 more versions
	if params.OtelCol.Spec.TargetAllocator.PrometheusCR.ServiceMonitorSelector != nil {
		taConfig["service_monitor_selector"] = &params.OtelCol.Spec.TargetAllocator.PrometheusCR.ServiceMonitorSelector.MatchLabels
	}

	prometheusCRConfig["pod_monitor_selector"] = params.OtelCol.Spec.TargetAllocator.PrometheusCR.PodMonitorSelector

	// The below instruction is here for compatibility with the previous target allocator version
	// TODO: Drop it after 3 more versions
	if params.OtelCol.Spec.TargetAllocator.PrometheusCR.PodMonitorSelector != nil {
		taConfig["pod_monitor_selector"] = &params.OtelCol.Spec.TargetAllocator.PrometheusCR.PodMonitorSelector.MatchLabels
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
			Namespace:   params.OtelCol.Namespace,
			Labels:      labels,
			Annotations: params.OtelCol.Annotations,
		},
		Data: map[string]string{
			targetAllocatorFilename: string(taConfigYAML),
		},
	}, nil
}
