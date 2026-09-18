// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package adapters is for data conversion.
package adapters

import (
	"encoding/json"
	"errors"
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

// ErrInvalidYAML represents an error in the format of the configuration file.
var ErrInvalidYAML = errors.New("couldn't parse the opentelemetry-collector configuration")

// ConfigFromString extracts a configuration map from the given string.
// If the given string isn't a valid YAML, ErrInvalidYAML is returned.
func ConfigFromString(configStr string) (map[any]any, error) {
	config := make(map[any]any)
	if err := yaml.Unmarshal([]byte(configStr), &config); err != nil {
		return nil, ErrInvalidYAML
	}

	return config, nil
}

// ConfigFromStruct converts a structured collector config into the generic map form that ConfigFromString
// produces. The conversion goes through the config's JSON representation, the form the CR is stored in, rather
// than through a YAML rendering of it, so no scalar can change type on the way. The result shares nothing with
// the input and can be modified freely; ConfigToStruct converts it back.
func ConfigFromStruct(cfg *v1beta1.Config) (map[any]any, error) {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal collector config: %w", err)
	}
	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("unmarshal collector config: %w", err)
	}
	return anyKeys(config).(map[any]any), nil
}

// ConfigToStruct converts a generic config map, as produced by ConfigFromString or ConfigFromStruct, back into a
// structured collector config through its JSON representation. All map keys must be strings.
func ConfigToStruct(config map[any]any) (*v1beta1.Config, error) {
	m, err := stringKeys(config)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal collector config: %w", err)
	}
	cfg := &v1beta1.Config{}
	if err := json.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("unmarshal collector config: %w", err)
	}
	return cfg, nil
}

// anyKeys returns a deep copy of v in which every map[string]any is a map[any]any.
func anyKeys(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[any]any, len(val))
		for k, elem := range val {
			out[k] = anyKeys(elem)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, elem := range val {
			out[i] = anyKeys(elem)
		}
		return out
	default:
		return v
	}
}

// stringKeys returns a deep copy of v in which every map[any]any is a map[string]any, failing on a key that is
// not a string.
func stringKeys(v any) (any, error) {
	switch val := v.(type) {
	case map[any]any:
		out := make(map[string]any, len(val))
		for k, elem := range val {
			key, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("map key %v is a %T, not a string", k, k)
			}
			converted, err := stringKeys(elem)
			if err != nil {
				return nil, err
			}
			out[key] = converted
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, elem := range val {
			converted, err := stringKeys(elem)
			if err != nil {
				return nil, err
			}
			out[k] = converted
		}
		return out, nil
	case []any:
		out := make([]any, len(val))
		for i, elem := range val {
			converted, err := stringKeys(elem)
			if err != nil {
				return nil, err
			}
			out[i] = converted
		}
		return out, nil
	default:
		return v, nil
	}
}
