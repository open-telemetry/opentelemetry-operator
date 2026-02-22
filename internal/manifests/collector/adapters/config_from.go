// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package adapters is for data conversion.
package adapters

import (
	"errors"

	"gopkg.in/yaml.v2"
)

var (
	// ErrInvalidYAML represents an error in the format of the configuration file.
	ErrInvalidYAML = errors.New("couldn't parse the opentelemetry-collector configuration")
)

// ConfigFromString extracts a configuration map from the given string.
// If the given string isn't a valid YAML, ErrInvalidYAML is returned.
func ConfigFromString(configStr string) (map[any]any, error) {
	config := make(map[any]any)
	if err := yaml.Unmarshal([]byte(configStr), &config); err != nil {
		return nil, ErrInvalidYAML
	}

	return config, nil
}
