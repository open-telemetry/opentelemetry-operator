// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

const standaloneConfigVersion = "opentelemetry.io/opamp-bridge-standalone/v1alpha1"

type standaloneConfig struct {
	Version   string            `json:"version" yaml:"version"`
	Name      string            `json:"name" yaml:"name"`
	Namespace string            `json:"namespace" yaml:"namespace"`
	Config    map[string]string `json:"config" yaml:"config"`
}

func (c standaloneConfig) validate(name, namespace string) error {
	if c.Version != standaloneConfigVersion {
		return fmt.Errorf("unsupported standalone config version %q", c.Version)
	}
	if c.Name == "" {
		return errors.New("standalone config name is required")
	}
	if c.Namespace == "" {
		return errors.New("standalone config namespace is required")
	}
	if c.Name != name {
		return fmt.Errorf("standalone config name %q does not match target name %q", c.Name, name)
	}
	if c.Namespace != namespace {
		return fmt.Errorf("standalone config namespace %q does not match target namespace %q", c.Namespace, namespace)
	}
	if len(c.Config) == 0 {
		return errors.New("standalone config data is required")
	}
	for key := range c.Config {
		if strings.TrimSpace(key) == "" {
			return errors.New("standalone config contains an empty data key")
		}
	}
	return nil
}

func (c standaloneConfig) validateCollectorConfig() error {
	var validationErrors []string

	keys := make([]string, 0, len(c.Config))
	for key := range c.Config {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	for _, key := range keys {
		if err := validateCollectorConfigEntry(c.Config[key]); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("%s: %v", key, err))
			continue
		}
		return nil
	}

	return fmt.Errorf("no valid OpenTelemetry Collector config found in standalone config data: %s", strings.Join(validationErrors, "; "))
}

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
