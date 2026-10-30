// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

// loadDefaultCollectorConfig reads DefaultCollectorConfigFile and stores the parsed configuration in
// Internal.DefaultCollectorConfig. It's a no-op when no file is configured. The file is parsed into the
// generic map form and then into v1beta1.Config through its JSON representation, the form collectors are
// stored in, so no scalar changes type on the way.
func (c *Config) loadDefaultCollectorConfig() error {
	if c.DefaultCollectorConfigFile == "" {
		return nil
	}

	data, err := os.ReadFile(c.DefaultCollectorConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", c.DefaultCollectorConfigFile, err)
	}

	var generic map[string]any
	if err = yaml.Unmarshal(data, &generic); err != nil {
		return fmt.Errorf("failed to parse file %q: %w", c.DefaultCollectorConfigFile, err)
	}

	raw, err := json.Marshal(generic)
	if err != nil {
		return fmt.Errorf("failed to encode config from file %q: %w", c.DefaultCollectorConfigFile, err)
	}

	parsed := &v1beta1.Config{}
	if err = json.Unmarshal(raw, parsed); err != nil {
		return fmt.Errorf("failed to decode config from file %q: %w", c.DefaultCollectorConfigFile, err)
	}

	c.Internal.DefaultCollectorConfig = parsed
	return nil
}
