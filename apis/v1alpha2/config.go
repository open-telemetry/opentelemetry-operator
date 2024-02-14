// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"gopkg.in/yaml.v3"
)

// AnyConfig represent parts of the config.
type AnyConfig struct {
	Object map[string]interface{} `json:"-" yaml:",inline"`
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (c *AnyConfig) DeepCopyInto(out *AnyConfig) {
	*out = *c
	if c.Object != nil {
		in, out := &c.Object, &out.Object
		*out = make(map[string]interface{}, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AnyConfig.
func (c *AnyConfig) DeepCopy() *AnyConfig {
	if c == nil {
		return nil
	}
	out := new(AnyConfig)
	c.DeepCopyInto(out)
	return out
}

var _ json.Marshaler = &AnyConfig{}
var _ json.Unmarshaler = &AnyConfig{}

// UnmarshalJSON implements an alternative parser for this field.
func (c *AnyConfig) UnmarshalJSON(b []byte) error {
	vals := map[string]interface{}{}
	if err := json.Unmarshal(b, &vals); err != nil {
		return err
	}
	c.Object = vals
	return nil
}

// MarshalJSON specifies how to convert this object into JSON.
func (c *AnyConfig) MarshalJSON() ([]byte, error) {
	if c == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(c.Object)
}

// Config encapsulates collector config.
type Config struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	Receivers AnyConfig `json:"receivers" yaml:"receivers"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Exporters AnyConfig `json:"exporters" yaml:"exporters"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Processors *AnyConfig `json:"processors,omitempty" yaml:"processors,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Connectors *AnyConfig `json:"connectors,omitempty" yaml:"connectors,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Extensions *AnyConfig `json:"extensions,omitempty" yaml:"extensions,omitempty"`
	Service    Service    `json:"service" yaml:"service"`
}

// Yaml encodes the current object and returns it as a string.
func (c Config) Yaml() (string, error) {
	var buf bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&buf)
	yamlEncoder.SetIndent(2)
	if err := yamlEncoder.Encode(&c); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// MetricsConfig comes from the collector
type MetricsConfig struct {
	// Level is the level of telemetry metrics, the possible values are:
	//  - "none" indicates that no telemetry data should be collected;
	//  - "basic" is the recommended and covers the basics of the service telemetry.
	//  - "normal" adds some other indicators on top of basic.
	//  - "detailed" adds dimensions and views to the previous levels.
	Level string `json:"level,omitempty" yaml:"level,omitempty"`

	// Address is the [address]:port that metrics exposition should be bound to.
	Address string `json:"address,omitempty" yaml:"address,omitempty"`
}

type Telemetry struct {
	Metrics MetricsConfig `json:"metrics,omitempty" yaml:"metrics,omitempty"`

	// Resource specifies user-defined attributes to include with all emitted telemetry.
	// Note that some attributes are added automatically (e.g. service.version) even
	// if they are not specified here. In order to suppress such attributes the
	// attribute must be specified in this map with null YAML value (nil string pointer).
	Resource map[string]*string `json:"resource,omitempty" yaml:"resource,omitempty"`
}

type Service struct {
	Extensions []string `json:"extensions,omitempty" yaml:"extensions,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Telemetry *Telemetry `json:"telemetry,omitempty" yaml:"telemetry,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Pipelines AnyConfig `json:"pipelines" yaml:"pipelines"`
}

// Returns null objects in the config.
func (c Config) nullObjects() []string {
	var nullKeys []string
	if nulls := hasNullValue(c.Receivers.Object); len(nulls) > 0 {
		nullKeys = append(nullKeys, addPrefix("receivers.", nulls)...)
	}
	if nulls := hasNullValue(c.Exporters.Object); len(nulls) > 0 {
		nullKeys = append(nullKeys, addPrefix("exporters.", nulls)...)
	}
	if c.Processors != nil {
		if nulls := hasNullValue(c.Processors.Object); len(nulls) > 0 {
			nullKeys = append(nullKeys, addPrefix("processors.", nulls)...)
		}
	}
	if c.Extensions != nil {
		if nulls := hasNullValue(c.Extensions.Object); len(nulls) > 0 {
			nullKeys = append(nullKeys, addPrefix("extensions.", nulls)...)
		}
	}
	if c.Connectors != nil {
		if nulls := hasNullValue(c.Connectors.Object); len(nulls) > 0 {
			nullKeys = append(nullKeys, addPrefix("connectors.", nulls)...)
		}
	}
	// Make the return deterministic. The config uses maps therefore processing order is non-deterministic.
	sort.Strings(nullKeys)
	return nullKeys
}

func hasNullValue(cfg map[string]interface{}) []string {
	var nullKeys []string
	for k, v := range cfg {
		if v == nil {
			nullKeys = append(nullKeys, fmt.Sprintf("%s:", k))
		}
		if reflect.ValueOf(v).Kind() == reflect.Map {
			var nulls []string
			val, ok := v.(map[string]interface{})
			if ok {
				nulls = hasNullValue(val)
			}
			if len(nulls) > 0 {
				prefixed := addPrefix(k+".", nulls)
				nullKeys = append(nullKeys, prefixed...)
			}
		}
	}
	return nullKeys
}

func addPrefix(prefix string, arr []string) []string {
	var prefixed []string
	for _, v := range arr {
		prefixed = append(prefixed, fmt.Sprintf("%s%s", prefix, v))
	}
	return prefixed
}
