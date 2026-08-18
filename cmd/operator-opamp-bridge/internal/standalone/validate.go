// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/webhook"
)

// validateCollectorConfigEntry checks that a standalone config value is valid collector config.
// It returns an error if the body is not valid Otel config. This helps report bad configs back to the opamp server.
func validateCollectorConfigEntry(body string) error {
	if strings.TrimSpace(body) == "" {
		return errors.New("config value is empty")
	}

	var cfg v1beta1.Config
	if err := yaml.Unmarshal([]byte(body), &cfg); err != nil {
		return fmt.Errorf("failed to parse collector config: %w", err)
	}

	collector := &v1beta1.OpenTelemetryCollector{
		Spec: v1beta1.OpenTelemetryCollectorSpec{
			Mode:   v1beta1.ModeDeployment,
			Config: cfg,
		},
	}
	if _, err := (webhook.CollectorWebhook{}).Validate(context.Background(), collector); err != nil {
		return fmt.Errorf("collector config failed operator validation: %w", err)
	}
	if err := validateCollectorConfigReferences(cfg); err != nil {
		return fmt.Errorf("collector config failed references validation: %w", err)
	}

	return nil
}

func validateCollectorConfigReferences(cfg v1beta1.Config) error {
	if len(cfg.Receivers.Object) == 0 {
		return errors.New("no receivers configured")
	}
	if len(cfg.Exporters.Object) == 0 {
		return errors.New("no exporters configured")
	}
	if len(cfg.Service.Pipelines) == 0 {
		return errors.New("no pipelines configured")
	}

	for pipelineName, pipeline := range cfg.Service.Pipelines {
		if pipeline == nil {
			return fmt.Errorf("pipeline %s is empty", pipelineName)
		}
		for _, receiver := range pipeline.Receivers {
			if !componentExists(receiver, cfg.Receivers.Object, cfg.Connectors) {
				return fmt.Errorf("pipeline %s references non-existent receiver %s", pipelineName, receiver)
			}
		}
		for _, processor := range pipeline.Processors {
			if cfg.Processors == nil || !componentExists(processor, cfg.Processors.Object, nil) {
				return fmt.Errorf("pipeline %s references non-existent processor %s", pipelineName, processor)
			}
		}
		for _, exporter := range pipeline.Exporters {
			if !componentExists(exporter, cfg.Exporters.Object, cfg.Connectors) {
				return fmt.Errorf("pipeline %s references non-existent exporter %s", pipelineName, exporter)
			}
		}
	}

	for _, extension := range cfg.Service.Extensions {
		if cfg.Extensions == nil || !componentExists(extension, cfg.Extensions.Object, nil) {
			return fmt.Errorf("service references non-existent extension %s", extension)
		}
	}

	return nil
}

func componentExists(name string, components map[string]any, connectors *v1beta1.AnyConfig) bool {
	if _, ok := components[name]; ok {
		return true
	}
	if connectors != nil {
		_, ok := connectors.Object[name]
		return ok
	}
	return false
}
