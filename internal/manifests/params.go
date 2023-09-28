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

package manifests

import (
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func NewOtelConfig(instance *v1alpha1.OpenTelemetryCollector) (OtelConfig, error){
	var cfg OtelConfig
	if instance.Spec.Config != "" {
		cfg.configStr = instance.Spec.Config
		err:=yaml.Unmarshal([]byte(instance.Spec.Config), &cfg)
		if err!= nil {
			return cfg, fmt.Errorf("there was a problem parsing the config: %w", err)
		}
	} else {
		cfg.configSpec = instance.Spec.ConfigSpec
		if instance.Spec.ConfigSpec.Exporters != nil {
			var exporters map[string]interface{}
			if err := json.Unmarshal(instance.Spec.ConfigSpec.Exporters.Raw, &exporters); err != nil {
				return cfg, fmt.Errorf("there was a problem parsing the exporters config: %w", err)
			}
			cfg.Exporters = exporters
		}
		if instance.Spec.ConfigSpec.Receivers != nil {
			var receivers map[string]interface{}
			if err := json.Unmarshal(instance.Spec.ConfigSpec.Exporters.Raw, &receivers); err != nil {
				return cfg, fmt.Errorf("there was a problem parsing the receivers config: %w", err)
			}
			cfg.Exporters = receivers
		}
		if instance.Spec.ConfigSpec.Connectors != nil {
			var connectors map[string]interface{}
			if err := json.Unmarshal(instance.Spec.ConfigSpec.Exporters.Raw, &connectors); err != nil {
				return cfg, fmt.Errorf("there was a problem parsing the connectors config: %w", err)
			}
			cfg.Exporters = connectors
		}
		if instance.Spec.ConfigSpec.Processors!= nil {
			var processors map[string]interface{}
			if err := json.Unmarshal(instance.Spec.ConfigSpec.Exporters.Raw, &processors); err != nil {
				return cfg, fmt.Errorf("there was a problem parsing the processors config: %w", err)
			}
			cfg.Exporters = processors
		}
		if instance.Spec.ConfigSpec.Service!= nil {
			var service map[string]interface{}
			if err := json.Unmarshal(instance.Spec.ConfigSpec.Exporters.Raw, &service); err != nil {
				return cfg, fmt.Errorf("there was a problem parsing the service config: %w", err)
			}
			cfg.Exporters = service
		}
	}
	return cfg, nil
}

type OtelConfig struct {
	Exporters map[string]interface{}  `yaml:"exporters,omitempty"`
	Receivers map[string]interface{} `yaml:"receivers,omitempty"`
	Connectors map[string]interface{} `yaml:"connectors,omitempty"`
	Processors map[string]interface{} `yaml:"processors,omitempty"`
	Service map[string]interface{} `yaml:"service,omitempty"`
	configStr string
	configSpec v1alpha1.ConfigSpec
}

func (c *OtelConfig) String() string {
	if c.configStr != "" {
		return c.configStr
	}
	return c.configSpec.String()
}

// Params holds the reconciliation-specific parameters.
type Params struct {
	OtelConfig OtelConfig
	Client   client.Client
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Instance v1alpha1.OpenTelemetryCollector
	Config   config.Config
}
