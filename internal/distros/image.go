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

package distros

import "k8s.io/utils/strings/slices"

type OtelcolConfig struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	Receivers map[string]any `json:"receivers" yaml:"receivers"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Exporters map[string]any `json:"exporters" yaml:"exporters"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Processors map[string]any `json:"processors,omitempty" yaml:"processors,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Connectors map[string]any `json:"connectors,omitempty" yaml:"connectors,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Extensions map[string]any `json:"extensions,omitempty" yaml:"extensions,omitempty"`
}

func IsValidConfigK8sDistro(config OtelcolConfig) bool {
	for receiver := range config.Receivers {
		if !slices.Contains(K8sReceivers, receiver) {
			return false
		}
	}
	for exporter := range config.Exporters {
		if !slices.Contains(K8sExporters, exporter) {
			return false
		}
	}
	for processor := range config.Processors {
		if !slices.Contains(K8sProcessors, processor) {
			return false
		}
	}
	for connector := range config.Connectors {
		if !slices.Contains(K8sConnectors, connector) {
			return false
		}
	}
	for extension := range config.Extensions {
		if !slices.Contains(K8sConnectors, extension) {
			return false
		}
	}
	return true
}

func IsValidConfigCoreDistro(config OtelcolConfig) bool {
	for receiver := range config.Receivers {
		if !slices.Contains(CoreReceivers, receiver) {
			return false
		}
	}
	for exporter := range config.Exporters {
		if !slices.Contains(CoreExporters, exporter) {
			return false
		}
	}
	for processor := range config.Processors {
		if !slices.Contains(CoreProcessors, processor) {
			return false
		}
	}
	for connector := range config.Connectors {
		if !slices.Contains(CoreConnectors, connector) {
			return false
		}
	}
	for extension := range config.Extensions {
		if !slices.Contains(CoreConnectors, extension) {
			return false
		}
	}
	return true
}
