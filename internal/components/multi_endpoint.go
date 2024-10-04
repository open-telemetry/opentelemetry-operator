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

package components

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

var _ Parser = &MultiPortReceiver{}

// MultiProtocolEndpointConfig represents the minimal struct for a given YAML configuration input containing a map to
// a struct with either endpoint or listen_address.
type MultiProtocolEndpointConfig struct {
	Protocols map[string]*SingleEndpointConfig `mapstructure:"protocols"`
}

// MultiPortOption allows the setting of options for a MultiPortReceiver.
type MultiPortOption func(parser *MultiPortReceiver)

// MultiPortReceiver is a special parser for components with endpoints for each protocol.
type MultiPortReceiver struct {
	name string

	addrMappings map[string]string
	portMappings map[string]*corev1.ServicePort
}

func (m *MultiPortReceiver) Ports(logger logr.Logger, name string, config interface{}) ([]corev1.ServicePort, error) {
	multiProtoEndpointCfg := &MultiProtocolEndpointConfig{}
	if err := mapstructure.Decode(config, multiProtoEndpointCfg); err != nil {
		return nil, err
	}
	var ports []corev1.ServicePort
	for protocol, ec := range multiProtoEndpointCfg.Protocols {
		if defaultSvc, ok := m.portMappings[protocol]; ok {
			port := defaultSvc.Port
			if ec != nil {
				port = ec.GetPortNumOrDefault(logger, port)
			}
			defaultSvc.Name = naming.PortName(fmt.Sprintf("%s-%s", name, protocol), port)
			ports = append(ports, ConstructServicePort(defaultSvc, port))
		} else {
			return nil, fmt.Errorf("unknown protocol set: %s", protocol)
		}
	}
	return ports, nil
}

func (m *MultiPortReceiver) ParserType() string {
	return ComponentType(m.name)
}

func (m *MultiPortReceiver) ParserName() string {
	return fmt.Sprintf("__%s", m.name)
}

func (m *MultiPortReceiver) GetDefaultConfig(logger logr.Logger, config interface{}) (interface{}, error) {
	multiProtoEndpointCfg := &MultiProtocolEndpointConfig{}
	if err := mapstructure.Decode(config, multiProtoEndpointCfg); err != nil {
		return nil, err
	}
	tmp := make(map[string]*SingleEndpointConfig, len(multiProtoEndpointCfg.Protocols))
	for protocol, ec := range multiProtoEndpointCfg.Protocols {
		var port int32
		if defaultSvc, ok := m.portMappings[protocol]; ok {
			port = defaultSvc.Port
			if ec != nil {
				port = ec.GetPortNumOrDefault(logger, port)
			}
		}
		var addr string
		if defaultAddr, ok := m.addrMappings[protocol]; ok {
			addr = defaultAddr
		}
		res, err := AddressDefaulter(logger, addr, port, ec)
		if err != nil {
			return nil, err
		}
		tmp[protocol] = res
	}

	for protocol, ec := range tmp {
		multiProtoEndpointCfg.Protocols[protocol] = ec
	}
	return config, mapstructure.Decode(multiProtoEndpointCfg, &config)

}
func (m *MultiPortReceiver) GetLivenessProbe(logger logr.Logger, config interface{}) (*corev1.Probe, error) {
	return nil, nil
}

func (m *MultiPortReceiver) GetReadinessProbe(logger logr.Logger, config interface{}) (*corev1.Probe, error) {
	return nil, nil
}

func (m *MultiPortReceiver) GetRBACRules(logr.Logger, interface{}) ([]rbacv1.PolicyRule, error) {
	return nil, nil
}

type MultiPortBuilder[ComponentConfigType any] []Builder[ComponentConfigType]

func NewMultiPortReceiverBuilder(name string) MultiPortBuilder[*MultiProtocolEndpointConfig] {
	return append(MultiPortBuilder[*MultiProtocolEndpointConfig]{}, NewBuilder[*MultiProtocolEndpointConfig]().WithName(name))
}

func NewProtocolBuilder(name string, port int32) Builder[*MultiProtocolEndpointConfig] {
	return NewBuilder[*MultiProtocolEndpointConfig]().WithName(name).WithPort(port).WithDefaultsApplier(MultiAddressDefaulter)
}

func (mp MultiPortBuilder[ComponentConfigType]) AddPortMapping(builder Builder[ComponentConfigType]) MultiPortBuilder[ComponentConfigType] {
	return append(mp, builder)
}

func (mp MultiPortBuilder[ComponentConfigType]) Build() (*MultiPortReceiver, error) {
	if len(mp) < 1 {
		return nil, fmt.Errorf("must provide at least one port mapping")
	}
	multiReceiver := &MultiPortReceiver{
		name:         mp[0].MustBuild().name,
		addrMappings: map[string]string{},
		portMappings: map[string]*corev1.ServicePort{},
	}
	for _, bu := range mp[1:] {
		built, err := bu.Build()
		if err != nil {
			return nil, err
		}
		multiReceiver.portMappings[built.name] = built.settings.GetServicePort()
		if built.settings != nil {
			multiReceiver.addrMappings[built.name] = built.settings.defaultRecAddr
		}
	}
	return multiReceiver, nil
}

func (mp MultiPortBuilder[ComponentConfigType]) MustBuild() *MultiPortReceiver {
	if p, err := mp.Build(); err != nil {
		panic(err)
	} else {
		return p
	}
}

func MultiAddressDefaulter(logger logr.Logger, defaultRecAddr string, port int32, config *MultiProtocolEndpointConfig) (*MultiProtocolEndpointConfig, error) {
	for protocol, ec := range config.Protocols {
		res, err := AddressDefaulter(logger, defaultRecAddr, port, ec)
		if err != nil {
			return nil, err
		}
		config.Protocols[protocol].Endpoint = res.Endpoint
	}
	return config, nil
}
