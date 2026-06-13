// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"errors"
	"fmt"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

func validateCollectorConfigEntry(body string) error {
	if strings.TrimSpace(body) == "" {
		return errors.New("config value is empty")
	}

	var cfg v1beta1.Config
	if err := yaml.Unmarshal([]byte(body), &cfg); err != nil {
		return fmt.Errorf("failed to parse collector config: %w", err)
	}

	if len(cfg.Receivers.Object) == 0 {
		return errors.New("collector config must define at least one receiver")
	}
	if len(cfg.Exporters.Object) == 0 {
		return errors.New("collector config must define at least one exporter")
	}
	if len(cfg.Service.Pipelines) == 0 {
		return errors.New("collector config must define at least one service pipeline")
	}

	for pipelineName, pipeline := range cfg.Service.Pipelines {
		if pipeline == nil {
			return fmt.Errorf("pipeline %s is empty", pipelineName)
		}
	}

	return nil
}
