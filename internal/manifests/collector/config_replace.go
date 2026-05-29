// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	go_yaml "github.com/goccy/go-yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector/adapters"
	ta "github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func ReplaceConfig(otelcol v1beta1.OpenTelemetryCollector, targetAllocator *v1alpha1.TargetAllocator, options ...ta.TAOption) (string, error) {
	collectorSpec := otelcol.Spec
	taEnabled := targetAllocator != nil
	cfgStr, err := collectorSpec.Config.Yaml()
	if err != nil {
		return "", err
	}
	// Check if TargetAllocator is present, if not, return the original config
	if !taEnabled {
		return cfgStr, nil
	}

	config, err := adapters.ConfigFromString(cfgStr)
	if err != nil {
		return "", err
	}

	promCfgMap, getCfgPromErr := ta.ConfigToPromConfig(cfgStr)
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

	// To avoid issues caused by Prometheus validation logic, which fails regex validation when it encounters
	// $$ in the prom config, we update the YAML file directly without marshaling and unmarshalling.
	updPromCfgMap, getCfgPromErr := ta.AddTAConfigToPromConfig(promCfgMap, naming.TAService(targetAllocator.Name), options...)
	if getCfgPromErr != nil {
		return "", getCfgPromErr
	}

	// type coercion checks are handled in the AddTAConfigToPromConfig method above
	config["receivers"].(map[any]any)["prometheus"] = updPromCfgMap

	out, updCfgMarshalErr := go_yaml.MarshalWithOptions(config, go_yaml.Indent(4), go_yaml.IndentSequence(true), go_yaml.AutoInt())
	if updCfgMarshalErr != nil {
		return "", updCfgMarshalErr
	}

	return string(out), nil
}
