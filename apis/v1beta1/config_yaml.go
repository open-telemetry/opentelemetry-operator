// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Yaml renders the config as the YAML document written to the collector's ConfigMap.
//
// The collector parses that document with go.yaml.in/yaml/v3 (through confmap), so the same library is used to
// write it: its encoder quotes exactly those scalars its own decoder would otherwise resolve to a non-string type
// ("0e12", "true", ".inf", "2001-12-14", ...), which is what keeps a value's type intact from the CR through the
// ConfigMap to the collector. The document is nevertheless decoded again and compared with the config's JSON
// representation (the form the CR is stored in) by VerifyYAMLEquivalence before it is returned, so a rendering
// the collector would read differently from the CR is reported as an error rather than written out.
//
// Known limitation: a map key "<<" cannot be rendered, because the decoder treats it as a YAML 1.1 merge key
// even when it is quoted (https://github.com/go-yaml/yaml/issues/245); Yaml returns an error for such a config.
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
	if err := c.VerifyYAMLEquivalence(buf.Bytes()); err != nil {
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
		if val == math.Trunc(val) && val >= math.MinInt64 && val < math.MaxInt64 && float64(int64(val)) == val {
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

// VerifyYAMLEquivalence checks that decoding doc the way the collector does (go.yaml.in/yaml/v3, into untyped
// values) yields the same values as the config's JSON representation, i.e. the values the CR holds. It returns
// an error naming the first difference found.
//
// Equivalence is judged by what the collector's config loader observes: every scalar must have the same type
// and value, and every collection the same members. Two differences are tolerated because they are artifacts of
// the Go serialization on either side and invisible to the collector:
//
//   - Numbers are compared by value. JSON has no integer/float distinction and encoding/json decodes every
//     number as float64, whereas the YAML decoder produces int for integer literals; the collector converts
//     either to the field's numeric type.
//   - null and an empty collection are interchangeable. encoding/json renders a nil map or slice as null, the
//     YAML encoder renders it as {} or []; the collector treats a null section like an empty one.
func (c *Config) VerifyYAMLEquivalence(doc []byte) error {
	jsonDoc, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config to JSON: %w", err)
	}
	var want any
	if err := json.Unmarshal(jsonDoc, &want); err != nil {
		return fmt.Errorf("unmarshal config JSON: %w", err)
	}
	var got any
	if err := yaml.Unmarshal(doc, &got); err != nil {
		return fmt.Errorf("decode rendered YAML: %w", err)
	}
	if err := equivalent("$", want, got); err != nil {
		return fmt.Errorf("rendered YAML is not equivalent to the config: %w", err)
	}
	return nil
}

// equivalent compares a value decoded from the config's JSON (want) with the value decoded from its YAML
// rendering (got), see VerifyYAMLEquivalence for the rules. path is where want sits in the config ("$" at the
// root); it grows with each recursion step and prefixes every error, so a mismatch names the exact field.
func equivalent(path string, want, got any) error {
	switch w := want.(type) {
	case nil:
		// A nil map or slice is null in JSON but {} or [] in YAML, see VerifyYAMLEquivalence.
		if got == nil || isEmptyCollection(got) {
			return nil
		}
	case bool:
		if g, ok := got.(bool); ok && g == w {
			return nil
		}
	case string:
		if g, ok := got.(string); ok && g == w {
			return nil
		}
	case float64:
		if g, ok := numberValue(got); ok && g == w {
			return nil
		}
	case []any:
		g, ok := got.([]any)
		if !ok {
			if len(w) == 0 && got == nil {
				return nil
			}
			break
		}
		if len(g) != len(w) {
			return fmt.Errorf("%s: %d elements in the config, %d in the YAML", path, len(w), len(g))
		}
		for i := range w {
			if err := equivalent(fmt.Sprintf("%s[%d]", path, i), w[i], g[i]); err != nil {
				return err
			}
		}
		return nil
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			if len(w) == 0 && got == nil {
				return nil
			}
			break
		}
		for k, wv := range w {
			gv, ok := g[k]
			if !ok {
				return fmt.Errorf("%s: key %q is missing from the YAML", path, k)
			}
			if err := equivalent(path+"."+k, wv, gv); err != nil {
				return err
			}
		}
		for k := range g {
			if _, ok := w[k]; !ok {
				return fmt.Errorf("%s: key %q is not in the config", path, k)
			}
		}
		return nil
	default:
		return fmt.Errorf("%s: unexpected value of type %T in the config's JSON", path, want)
	}
	return fmt.Errorf("%s: the config holds %s, the YAML decodes to %s", path, describe(want), describe(got))
}

// isEmptyCollection reports whether v is a map or slice with no elements, whatever its key and element types.
func isEmptyCollection(v any) bool {
	rv := reflect.ValueOf(v)
	return (rv.Kind() == reflect.Map || rv.Kind() == reflect.Slice) && rv.Len() == 0
}

// numberValue returns the value of a scalar of any integer or float type; go.yaml.in/yaml/v3 produces int,
// int64 or uint64 for integers and float64 for other numbers.
func numberValue(v any) (float64, bool) {
	rv := reflect.ValueOf(v)
	switch {
	case rv.CanInt():
		return float64(rv.Int()), true
	case rv.CanUint():
		return float64(rv.Uint()), true
	case rv.CanFloat():
		return rv.Float(), true
	}
	return 0, false
}

func describe(v any) string {
	if v == nil {
		return "null"
	}
	return fmt.Sprintf("%T %v", v, v)
}
