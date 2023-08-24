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

package config

import (
	"crypto/rand"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/protobufs"
	"gopkg.in/yaml.v2"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/logger"
)

const (
	agentType             = "io.opentelemetry.operator-opamp-bridge"
	defaultConfigFilePath = "/conf/remoteconfiguration.yaml"
)

var (
	agentVersion = os.Getenv("OPAMP_VERSION")
	hostname, _  = os.Hostname()
)

type Config struct {
	Endpoint     string   `yaml:"endpoint"`
	Protocol     string   `yaml:"protocol"`
	Capabilities []string `yaml:"capabilities"`

	// ComponentsAllowed is a list of allowed OpenTelemetry components for each pipeline type (receiver, processor, etc.)
	ComponentsAllowed map[string][]string `yaml:"components_allowed,omitempty"`
}

func (c *Config) CreateClient(logger *logger.Logger) client.OpAMPClient {
	if c.Protocol == "http" {
		return client.NewHTTP(logger)
	}
	return client.NewWebSocket(logger)
}

func (c *Config) GetComponentsAllowed() map[string]map[string]bool {
	m := make(map[string]map[string]bool)
	for component, componentSet := range c.ComponentsAllowed {
		if _, ok := m[component]; !ok {
			m[component] = make(map[string]bool)
		}
		for _, s := range componentSet {
			m[component][s] = true
		}
	}
	return m
}

func (c *Config) GetCapabilities() protobufs.AgentCapabilities {
	var capabilities int32
	for _, capability := range c.Capabilities {
		// This is a helper so that we don't force consumers to prefix every agent capability
		formatted := fmt.Sprintf("AgentCapabilities_%s", capability)
		if v, ok := protobufs.AgentCapabilities_value[formatted]; ok {
			capabilities = v | capabilities
		}
	}
	return protobufs.AgentCapabilities(capabilities)
}

func (c *Config) GetAgentType() string {
	return agentType
}

func (c *Config) GetAgentVersion() string {
	return agentVersion
}

func (c *Config) GetDescription() *protobufs.AgentDescription {
	return &protobufs.AgentDescription{
		IdentifyingAttributes: []*protobufs.KeyValue{
			keyValuePair("service.name", c.GetAgentType()),
			keyValuePair("service.version", c.GetAgentVersion()),
		},
		NonIdentifyingAttributes: []*protobufs.KeyValue{
			keyValuePair("os.family", runtime.GOOS),
			keyValuePair("host.name", hostname),
		},
	}
}

func keyValuePair(key string, value string) *protobufs.KeyValue {
	return &protobufs.KeyValue{
		Key: key,
		Value: &protobufs.AnyValue{
			Value: &protobufs.AnyValue_StringValue{
				StringValue: value,
			},
		},
	}
}

func (c *Config) GetNewInstanceId() ulid.ULID {
	entropy := ulid.Monotonic(rand.Reader, 0)
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy)
}

func (c *Config) RemoteConfigEnabled() bool {
	capabilities := c.GetCapabilities()
	return capabilities&protobufs.AgentCapabilities_AgentCapabilities_AcceptsRemoteConfig != 0
}

func Load(file string) (Config, error) {
	var cfg Config
	if err := unmarshal(&cfg, file); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func unmarshal(cfg *Config, configFile string) error {
	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	if err = yaml.UnmarshalStrict(yamlFile, cfg); err != nil {
		return fmt.Errorf("error unmarshaling YAML: %w", err)
	}
	return nil
}
