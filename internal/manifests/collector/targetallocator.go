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

package collector

import (
	"github.com/mitchellh/mapstructure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator/adapters"
)

// TargetAllocator builds the TargetAllocator CR for the given instance.
func TargetAllocator(params manifests.Params) (*v1alpha1.TargetAllocator, error) {

	taSpec := params.OtelCol.Spec.TargetAllocator
	if !taSpec.Enabled {
		return nil, nil
	}

	configStr, err := params.OtelCol.Spec.Config.Yaml()
	if err != nil {
		return nil, err
	}
	scrapeConfigs, err := getScrapeConfigs(configStr)
	if err != nil {
		return nil, err
	}
	globalConfig, err := getGlobalConfig(params.OtelCol.Spec.Config)
	if err != nil {
		return nil, err
	}

	return &v1alpha1.TargetAllocator{
		ObjectMeta: metav1.ObjectMeta{
			Name:        params.OtelCol.Name,
			Namespace:   params.OtelCol.Namespace,
			Annotations: params.OtelCol.Annotations,
			Labels:      params.OtelCol.Labels,
		},
		Spec: v1alpha1.TargetAllocatorSpec{
			OpenTelemetryCommonFields: v1beta1.OpenTelemetryCommonFields{
				Replicas:                  taSpec.Replicas,
				NodeSelector:              taSpec.NodeSelector,
				Resources:                 taSpec.Resources,
				ServiceAccount:            taSpec.ServiceAccount,
				Image:                     taSpec.Image,
				Affinity:                  taSpec.Affinity,
				SecurityContext:           taSpec.SecurityContext,
				PodSecurityContext:        taSpec.PodSecurityContext,
				TopologySpreadConstraints: taSpec.TopologySpreadConstraints,
				Tolerations:               taSpec.Tolerations,
				Env:                       taSpec.Env,
				PodAnnotations:            params.OtelCol.Spec.PodAnnotations,
				PodDisruptionBudget:       taSpec.PodDisruptionBudget,
			},
			AllocationStrategy: taSpec.AllocationStrategy,
			FilterStrategy:     taSpec.FilterStrategy,
			ScrapeConfigs:      scrapeConfigs,
			GlobalConfig:       globalConfig,
			PrometheusCR:       taSpec.PrometheusCR,
			Observability:      taSpec.Observability,
		},
	}, nil
}

func getGlobalConfig(otelConfig v1beta1.Config) (v1beta1.AnyConfig, error) {
	// TODO: Eventually we should figure out a way to pull this in to the main specification for the TA
	type promReceiverConfig struct {
		Prometheus struct {
			Config struct {
				Global map[string]interface{} `mapstructure:"global"`
			} `mapstructure:"config"`
		} `mapstructure:"prometheus"`
	}
	decodedConfig := &promReceiverConfig{}
	if err := mapstructure.Decode(otelConfig.Receivers.Object, decodedConfig); err != nil {
		return v1beta1.AnyConfig{}, err
	}
	return v1beta1.AnyConfig{
		Object: decodedConfig.Prometheus.Config.Global,
	}, nil
}

func getScrapeConfigs(otelcolConfig string) ([]v1beta1.AnyConfig, error) {
	// Collector supports environment variable substitution, but the TA does not.
	// TA Scrape Configs should have a single "$", as it does not support env var substitution
	prometheusReceiverConfig, err := adapters.UnescapeDollarSignsInPromConfig(otelcolConfig)
	if err != nil {
		return nil, err
	}

	scrapeConfigs, err := adapters.GetScrapeConfigsFromPromConfig(prometheusReceiverConfig)
	if err != nil {
		return nil, err
	}

	v1beta1scrapeConfigs := make([]v1beta1.AnyConfig, len(scrapeConfigs))

	for i, config := range scrapeConfigs {
		v1beta1scrapeConfigs[i] = v1beta1.AnyConfig{Object: config}
	}

	return v1beta1scrapeConfigs, nil
}
