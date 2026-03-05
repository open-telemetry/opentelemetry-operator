// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"errors"
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
	name           string
	defaultRecAddr string

	addrMappings map[string]string
	portMappings map[string]*corev1.ServicePort
}

func (m *MultiPortReceiver) Ports(logger logr.Logger, name string, config any) ([]corev1.ServicePort, error) {
	multiProtoEndpointCfg := &MultiProtocolEndpointConfig{}
	if err := mapstructure.Decode(config, multiProtoEndpointCfg); err != nil {
		return nil, err
	}
	var ports []corev1.ServicePort
	for protocol, ec := range multiProtoEndpointCfg.Protocols {
		defaultSvc, ok := m.portMappings[protocol]
		if !ok {
			return nil, fmt.Errorf("unknown protocol set: %s", protocol)
		}
		port := defaultSvc.Port
		if ec != nil {
			port = ec.GetPortNumOrDefault(logger, port)
		}
		defaultSvc.Name = naming.PortName(fmt.Sprintf("%s-%s", name, protocol), port)
		ports = append(ports, ConstructServicePort(defaultSvc, port))
	}
	return ports, nil
}

func (m *MultiPortReceiver) ParserType() string {
	return ComponentType(m.name)
}

func (m *MultiPortReceiver) ParserName() string {
	return fmt.Sprintf("__%s", m.name)
}

func (m *MultiPortReceiver) GetDefaultConfig(logger logr.Logger, config any, opts ...DefaultOption) (any, error) {
	// Apply options to build the DefaultConfig
	defaultCfg := &DefaultConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(defaultCfg)
		}
	}

	multiProtoEndpointCfg := &MultiProtocolEndpointConfig{}
	if err := mapstructure.Decode(config, multiProtoEndpointCfg); err != nil {
		return nil, err
	}
	defaultedConfig := map[string]any{}
	for protocol, ec := range multiProtoEndpointCfg.Protocols {
		defaultSvc, ok := m.portMappings[protocol]
		if !ok {
			return nil, fmt.Errorf("unknown protocol set: %s", protocol)
		}
		port := defaultSvc.Port
		if ec != nil {
			port = ec.GetPortNumOrDefault(logger, port)
		}
		addr := m.defaultRecAddr
		if defaultAddr, ok := m.addrMappings[protocol]; ok {
			addr = defaultAddr
		}
		conf, err := AddressDefaulter(logger, defaultCfg, addr, port, ec)
		if err != nil {
			return nil, err
		}
		defaultedConfig[protocol] = conf
	}
	return map[string]any{
		"protocols": defaultedConfig,
	}, nil
}

func (*MultiPortReceiver) GetLivenessProbe(logr.Logger, any) (*corev1.Probe, error) {
	return nil, nil
}

func (*MultiPortReceiver) GetReadinessProbe(logr.Logger, any) (*corev1.Probe, error) {
	return nil, nil
}

func (*MultiPortReceiver) GetStartupProbe(logr.Logger, any) (*corev1.Probe, error) {
	return nil, nil
}

func (*MultiPortReceiver) GetRBACRules(logr.Logger, any) ([]rbacv1.PolicyRule, error) {
	return nil, nil
}

func (*MultiPortReceiver) GetEnvironmentVariables(logr.Logger, any) ([]corev1.EnvVar, error) {
	return nil, nil
}

type MultiPortBuilder[ComponentConfigType any] []Builder[ComponentConfigType]

func NewMultiPortReceiverBuilder(name string) MultiPortBuilder[*MultiProtocolEndpointConfig] {
	return append(MultiPortBuilder[*MultiProtocolEndpointConfig]{}, NewBuilder[*MultiProtocolEndpointConfig]().WithName(name).WithDefaultRecAddress(DefaultRecAddress))
}

func NewProtocolBuilder(name string, port int32) Builder[*MultiProtocolEndpointConfig] {
	return NewBuilder[*MultiProtocolEndpointConfig]().WithName(name).WithPort(port).WithDefaultRecAddress(DefaultRecAddress)
}

func (mp MultiPortBuilder[ComponentConfigType]) AddPortMapping(builder Builder[ComponentConfigType]) MultiPortBuilder[ComponentConfigType] {
	return append(mp, builder)
}

func (mp MultiPortBuilder[ComponentConfigType]) Build() (*MultiPortReceiver, error) {
	if len(mp) < 1 {
		return nil, errors.New("must provide at least one port mapping")
	}

	mb := mp[0].MustBuild()
	multiReceiver := &MultiPortReceiver{
		name:           mb.name,
		defaultRecAddr: mb.settings.defaultRecAddr,
		addrMappings:   map[string]string{},
		portMappings:   map[string]*corev1.ServicePort{},
	}
	for _, bu := range mp[1:] {
		built, err := bu.Build()
		if err != nil {
			return nil, err
		}
		if built.settings != nil {
			multiReceiver.portMappings[built.name] = built.settings.GetServicePort()
			multiReceiver.addrMappings[built.name] = built.settings.defaultRecAddr
		}
	}
	return multiReceiver, nil
}

func (mp MultiPortBuilder[ComponentConfigType]) MustBuild() *MultiPortReceiver {
	p, err := mp.Build()
	if err != nil {
		panic(err)
	}
	return p
}
