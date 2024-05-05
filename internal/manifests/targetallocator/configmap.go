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
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

const (
	targetAllocatorFilename = "targetallocator.yaml"
)

func ConfigMap(params manifests.Params) (*corev1.ConfigMap, error) {
	instance := params.TargetAllocator
	name := naming.TAConfigMap(instance.Name)
	labels := manifestutils.Labels(instance.ObjectMeta, name, params.TargetAllocator.Spec.Image, ComponentOpenTelemetryTargetAllocator, nil)
	taSpec := instance.Spec

	taConfig := make(map[interface{}]interface{})
	prometheusCRConfig := make(map[interface{}]interface{})
	taConfig["collector_selector"] = taSpec.CollectorSelector

	// Add scrape configs if present
	if instance.Spec.ScrapeConfigs != nil && len(instance.Spec.ScrapeConfigs) > 0 {
		taConfig["config"] = map[string]interface{}{
			"scrape_configs": instance.Spec.ScrapeConfigs,
		}
	}

	if len(taSpec.AllocationStrategy) > 0 {
		taConfig["allocation_strategy"] = taSpec.AllocationStrategy
	} else {
		taConfig["allocation_strategy"] = v1beta1.TargetAllocatorAllocationStrategyConsistentHashing
	}
	taConfig["filter_strategy"] = taSpec.FilterStrategy

	if taSpec.PrometheusCR.ScrapeInterval.Size() > 0 {
		prometheusCRConfig["scrape_interval"] = taSpec.PrometheusCR.ScrapeInterval.Duration
	}

	prometheusCRConfig["service_monitor_selector"] = taSpec.PrometheusCR.ServiceMonitorSelector

	prometheusCRConfig["pod_monitor_selector"] = taSpec.PrometheusCR.PodMonitorSelector

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
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: instance.Annotations,
		},
		Data: map[string]string{
			targetAllocatorFilename: string(taConfigYAML),
		},
	}, nil
}
