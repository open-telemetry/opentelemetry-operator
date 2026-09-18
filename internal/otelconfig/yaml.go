// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconfig

import (
	"encoding/json"
	"fmt"
	"reflect"

	"go.yaml.in/yaml/v3"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

// RenderYAML renders the config as the YAML document written to the collector's ConfigMap and verifies, before
// returning it, that the collector will read back exactly the values the config holds.
//
// Config.Yaml keeps every value's type intact by construction - it encodes with the same library the collector
// decodes with - but the operator does not take that on trust for the documents it writes: the rendered document
// is decoded again and compared with the config's JSON representation, the form the CR is stored in. A rendering
// the collector would read differently from the config is reported as an error instead of being returned,
// whatever its cause - an encoder regression, or a config no rendering can represent faithfully (for example a
// "<<" map key, which the decoder treats as a YAML 1.1 merge key even when it is quoted,
// https://github.com/go-yaml/yaml/issues/245).
//
// Every operator code path that writes out a rendered collector config must go through this function instead of
// calling Config.Yaml directly.
func RenderYAML(c *v1beta1.Config) (string, error) {
	doc, err := c.Yaml()
	if err != nil {
		return "", err
	}
	if err := verifyYAMLEquivalence(c, []byte(doc)); err != nil {
		return "", err
	}
	return doc, nil
}

// verifyYAMLEquivalence checks that decoding doc the way the collector does (go.yaml.in/yaml/v3, into untyped
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
func verifyYAMLEquivalence(c *v1beta1.Config, doc []byte) error {
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
// rendering (got), see verifyYAMLEquivalence for the rules. path is where want sits in the config ("$" at the
// root); it grows with each recursion step and prefixes every error, so a mismatch names the exact field.
func equivalent(path string, want, got any) error {
	switch w := want.(type) {
	case nil:
		// A nil map or slice is null in JSON but {} or [] in YAML, see verifyYAMLEquivalence.
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
