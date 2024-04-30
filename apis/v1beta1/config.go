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

package v1beta1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type ComponentType int

const (
	ComponentTypeReceiver ComponentType = iota
	ComponentTypeExporter
	ComponentTypeProcessor
)

func (c ComponentType) String() string {
	return [...]string{"receiver", "exporter", "processor"}[c]
}

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

type ComponentDefinitions map[string]*AnyConfig

func (cd ComponentDefinitions) hasNullValues(prefix string) []string {
	var nullKeys []string
	for key, config := range cd {
		prefixWithKey := fmt.Sprintf("%s.%s", prefix, key)
		if config == nil {
			nullKeys = append(nullKeys, prefixWithKey+":")
			continue
		}
		if nulls := hasNullValue(config.Object); len(nulls) > 0 {
			nullKeys = append(nullKeys, addPrefix(prefixWithKey, nulls)...)
		}
	}
	return nullKeys
}

// Pipeline is a struct of component type to a list of component IDs.
type Pipeline struct {
	Exporters  []string `json:"exporters" yaml:"exporters"`
	Processors []string `json:"processors" yaml:"processors"`
	Receivers  []string `json:"receivers" yaml:"receivers"`
}

// GetEnabledComponents constructs a list of enabled components by component type.
func (c *Config) GetEnabledComponents() map[ComponentType]map[string]interface{} {
	toReturn := map[ComponentType]map[string]interface{}{
		ComponentTypeReceiver:  {},
		ComponentTypeProcessor: {},
		ComponentTypeExporter:  {},
	}
	for _, pipeline := range c.Service.Pipelines {
		if pipeline == nil {
			continue
		}
		for _, componentId := range pipeline.Receivers {
			toReturn[ComponentTypeReceiver][componentId] = struct{}{}
		}
		for _, componentId := range pipeline.Exporters {
			toReturn[ComponentTypeExporter][componentId] = struct{}{}
		}
		for _, componentId := range pipeline.Processors {
			toReturn[ComponentTypeProcessor][componentId] = struct{}{}
		}
	}
	return toReturn
}

// Config encapsulates collector config.
type Config struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	Receivers ComponentDefinitions `json:"receivers" yaml:"receivers"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Exporters ComponentDefinitions `json:"exporters" yaml:"exporters"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Processors ComponentDefinitions `json:"processors,omitempty" yaml:"processors,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Connectors ComponentDefinitions `json:"connectors,omitempty" yaml:"connectors,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Extensions ComponentDefinitions `json:"extensions,omitempty" yaml:"extensions,omitempty"`
	Service    Service              `json:"service" yaml:"service"`
}

// Yaml encodes the current object and returns it as a string.
func (c *Config) Yaml() (string, error) {
	var buf bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&buf)
	yamlEncoder.SetIndent(2)
	if err := yamlEncoder.Encode(&c); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Returns null objects in the config.
func (c *Config) nullObjects() []string {
	var nullKeys []string
	nullKeys = append(nullKeys, c.Receivers.hasNullValues("receivers")...)
	nullKeys = append(nullKeys, c.Exporters.hasNullValues("exporters")...)
	nullKeys = append(nullKeys, c.Processors.hasNullValues("processors")...)
	nullKeys = append(nullKeys, c.Extensions.hasNullValues("extensions")...)
	nullKeys = append(nullKeys, c.Connectors.hasNullValues("connectors")...)
	// Make the return deterministic. The config uses maps therefore processing order is non-deterministic.
	sort.Strings(nullKeys)
	return nullKeys
}

type Service struct {
	Extensions *[]string `json:"extensions,omitempty" yaml:"extensions,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Telemetry *AnyConfig `json:"telemetry,omitempty" yaml:"telemetry,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Pipelines map[string]*Pipeline `json:"pipelines" yaml:"pipelines"`
}

// MetricsPort gets the port number for the metrics endpoint from the collector config if it has been set.
func (s *Service) MetricsPort() (int32, error) {
	if s.GetTelemetry() == nil {
		// telemetry isn't set, use the default
		return 8888, nil
	}
	_, port, netErr := net.SplitHostPort(s.GetTelemetry().Metrics.Address)
	if netErr != nil && strings.Contains(netErr.Error(), "missing port in address") {
		return 8888, nil
	} else if netErr != nil {
		return 0, netErr
	}
	i64, err := strconv.ParseInt(port, 10, 32)
	if err != nil {
		return 0, err
	}

	return int32(i64), nil
}

// MetricsConfig comes from the collector.
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

// Telemetry is an intermediary type that allows for easy access to the collector's telemetry settings.
type Telemetry struct {
	Metrics MetricsConfig `json:"metrics,omitempty" yaml:"metrics,omitempty"`

	// Resource specifies user-defined attributes to include with all emitted telemetry.
	// Note that some attributes are added automatically (e.g. service.version) even
	// if they are not specified here. In order to suppress such attributes the
	// attribute must be specified in this map with null YAML value (nil string pointer).
	Resource map[string]*string `json:"resource,omitempty" yaml:"resource,omitempty"`
}

// GetTelemetry serves as a helper function to access the fields we care about in the underlying telemetry struct.
// This exists to avoid needing to worry extra fields in the telemetry struct.
func (s *Service) GetTelemetry() *Telemetry {
	if s.Telemetry == nil {
		return nil
	}
	// Convert map to JSON bytes
	jsonData, err := json.Marshal(s.Telemetry)
	if err != nil {
		return nil
	}
	t := &Telemetry{}
	// Unmarshal JSON into the provided struct
	if err := json.Unmarshal(jsonData, t); err != nil {
		return nil
	}
	return t
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
				prefixed := addPrefix(k, nulls)
				nullKeys = append(nullKeys, prefixed...)
			}
		}
	}
	return nullKeys
}

func addPrefix(prefix string, arr []string) []string {
	var prefixed []string
	for _, v := range arr {
		prefixed = append(prefixed, fmt.Sprintf("%s.%s", prefix, v))
	}
	return prefixed
}
