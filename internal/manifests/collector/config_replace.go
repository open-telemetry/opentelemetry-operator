// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
	ta "github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/internal/otelconfig"
)

// ReplaceConfig renders the collector configuration written to the collector's ConfigMap. When a target
// allocator is in use, the Prometheus receiver is rewritten to fetch its scrape targets from it.
//
// The rendering is always a single otelconfig.RenderYAML call on the values the CR holds. The target allocator rewrite
// happens on a generic map derived from those values through their JSON representation (adapters.ConfigFromStruct
// and adapters.ConfigToStruct), never by parsing a YAML rendering of them, so no intermediate step can change a
// value's type on its way to the ConfigMap.
func ReplaceConfig(otelcol v1beta1.OpenTelemetryCollector, targetAllocator *v1alpha1.TargetAllocator, options ...ta.TAOption) (string, error) {
	collectorSpec := otelcol.Spec
	taEnabled := targetAllocator != nil
	// Check if TargetAllocator is present, if not, return the original config
	if !taEnabled {
		return otelconfig.RenderYAML(&collectorSpec.Config)
	}

	config, err := adapters.ConfigFromStruct(&collectorSpec.Config)
	if err != nil {
		return "", err
	}

	promCfgMap, getCfgPromErr := ta.PromReceiverConfig(config)
	if getCfgPromErr != nil {
		return "", getCfgPromErr
	}

	validateCfgPromErr := ta.ValidatePromConfig(promCfgMap, taEnabled)
	if validateCfgPromErr != nil {
		return "", validateCfgPromErr
	}

	// Use the interval from CRD (which has a default of 30s if not specified)
	if otelcol.Spec.TargetAllocator.CollectorTargetReloadInterval != nil {
		interval := otelcol.Spec.TargetAllocator.CollectorTargetReloadInterval.Duration.String()
		options = append(options, ta.WithCollectorTargetReloadInterval(interval))
	}

	updPromCfgMap, getCfgPromErr := ta.AddTAConfigToPromConfig(promCfgMap, naming.TAServiceFQDN(targetAllocator.Name, targetAllocator.Namespace), options...)
	if getCfgPromErr != nil {
		return "", getCfgPromErr
	}

	// type coercion checks are handled in the AddTAConfigToPromConfig method above
	config["receivers"].(map[any]any)["prometheus"] = updPromCfgMap

	updatedConfig, err := adapters.ConfigToStruct(config)
	if err != nil {
		return "", err
	}
	return otelconfig.RenderYAML(updatedConfig)
}
