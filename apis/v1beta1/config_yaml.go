// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"bytes"
	"math"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Yaml renders the config as the YAML document written to the collector's ConfigMap.
//
// The collector parses that document with go.yaml.in/yaml/v3 (through confmap), so the same library is used to
// write it: its encoder quotes exactly those scalars its own decoder would otherwise resolve to a non-string type
// ("0e12", "true", ".inf", "2001-12-14", ...), which is what keeps a value's type intact from the CR through the
// ConfigMap to the collector.
//
// Known limitation: a map key "<<" does not render faithfully, because the decoder treats it as a YAML 1.1
// merge key even when it is quoted (https://github.com/go-yaml/yaml/issues/245).
func (c *Config) Yaml() (string, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(c.renderable()); err != nil {
		return "", err
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderable returns a copy of the config with two adjustments for the encoder, neither of which changes what the
// collector reads. The copy shares nothing with the receiver's AnyConfig trees, which stay untouched.
//
// Every float64 holding an integer value becomes an int64. The CR is stored as JSON, and AnyConfig.UnmarshalJSON
// decodes JSON numbers with encoding/json, which represents all of them as float64; the encoder writes large
// float64 values in exponent notation, so an integer the CR spells as 1000000 would appear as 1e+06. The
// collector reads both as the same number, but the ConfigMap should spell integers the way the CR does. Only
// values that convert to int64 and back without loss are changed.
//
// Every multi-line string whose first line starts with a tab is marked to be double-quoted, working around an
// encoder bug that would otherwise render it unparseable; see renderableValue.
func (c *Config) renderable() *Config {
	out := *c
	out.Receivers = renderableAnyConfig(c.Receivers)
	out.Exporters = renderableAnyConfig(c.Exporters)
	if c.Processors != nil {
		processors := renderableAnyConfig(*c.Processors)
		out.Processors = &processors
	}
	if c.Connectors != nil {
		connectors := renderableAnyConfig(*c.Connectors)
		out.Connectors = &connectors
	}
	if c.Extensions != nil {
		extensions := renderableAnyConfig(*c.Extensions)
		out.Extensions = &extensions
	}
	if c.Service.Telemetry != nil {
		telemetry := renderableAnyConfig(*c.Service.Telemetry)
		out.Service.Telemetry = &telemetry
	}
	return &out
}

func renderableAnyConfig(c AnyConfig) AnyConfig {
	if c.Object == nil {
		return c
	}
	return AnyConfig{Object: renderableValue(c.Object).(map[string]any)}
}

// renderableValue returns a deep copy of a decoded value tree with the renderable adjustments applied.
//
// A multi-line string whose first line starts with a tab is turned into an explicitly double-quoted scalar to
// work around https://github.com/yaml/go-yaml/issues/383: the encoder writes multi-line strings as literal block
// scalars, but adds the indentation indicator the block needs only when the string starts with a space or a line
// break; with a leading tab it emits a block that its own parser rejects ("found a tab character where an
// indentation space is expected"). Only the first line matters: the block's indentation is fixed by it, after
// which a tab is content.
func renderableValue(v any) any {
	switch val := v.(type) {
	case float64:
		// As a float64 comparand, MaxInt64 rounds up to 2^63, so this admits every integral value in
		// [MinInt64, 2^63) - exactly the ones that convert to int64 and back without loss.
		if val == math.Trunc(val) && val >= math.MinInt64 && val < math.MaxInt64 {
			return int64(val)
		}
		return val
	case string:
		// Work around https://github.com/yaml/go-yaml/issues/383, see the function comment.
		if strings.HasPrefix(val, "\t") && strings.Contains(val, "\n") {
			return &yaml.Node{Kind: yaml.ScalarNode, Style: yaml.DoubleQuotedStyle, Value: val}
		}
		return val
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, elem := range val {
			out[k] = renderableValue(elem)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, elem := range val {
			out[i] = renderableValue(elem)
		}
		return out
	default:
		return v
	}
}
