// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

var (
	_ Parser = &GenericParser[*SingleEndpointConfig]{}
)

// SingleEndpointConfig represents the minimal struct for a given YAML configuration input containing either
// endpoint or listen_address.
type SingleEndpointConfig struct {
	Endpoint      string `mapstructure:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	ListenAddress string `mapstructure:"listen_address,omitempty" yaml:"listen_address,omitempty"`
}

func (g *SingleEndpointConfig) GetPortNumOrDefault(logger logr.Logger, p int32) int32 {
	num, err := g.GetPortNum()
	if err != nil {
		logger.V(3).Info("no port set, using default: %d", p)
		return p
	}
	return num
}

// GetPortNum attempts to get the port for the given config. If it cannot, the UnsetPort and the given missingPortError
// are returned.
func (g *SingleEndpointConfig) GetPortNum() (int32, error) {
	if len(g.Endpoint) > 0 {
		return PortFromEndpoint(g.Endpoint)
	} else if len(g.ListenAddress) > 0 {
		return PortFromEndpoint(g.ListenAddress)
	}
	return UnsetPort, PortNotFoundErr
}

func ParseSingleEndpointSilent(logger logr.Logger, name string, defaultPort *corev1.ServicePort, singleEndpointConfig *SingleEndpointConfig) ([]corev1.ServicePort, error) {
	return internalParseSingleEndpoint(logger, name, true, defaultPort, singleEndpointConfig)
}

func ParseSingleEndpoint(logger logr.Logger, name string, defaultPort *corev1.ServicePort, singleEndpointConfig *SingleEndpointConfig) ([]corev1.ServicePort, error) {
	return internalParseSingleEndpoint(logger, name, false, defaultPort, singleEndpointConfig)
}

func internalParseSingleEndpoint(logger logr.Logger, name string, failSilently bool, defaultPort *corev1.ServicePort, singleEndpointConfig *SingleEndpointConfig) ([]corev1.ServicePort, error) {
	if singleEndpointConfig == nil {
		return nil, nil
	}
	if _, err := singleEndpointConfig.GetPortNum(); err != nil && defaultPort.Port == UnsetPort {
		if failSilently {
			logger.WithValues("receiver", defaultPort.Name).V(4).Info("couldn't parse the endpoint's port and no default port set", "error", err)
			err = nil
		} else {
			logger.WithValues("receiver", defaultPort.Name).Error(err, "couldn't parse the endpoint's port and no default port set")
		}
		return []corev1.ServicePort{}, err
	}
	port := singleEndpointConfig.GetPortNumOrDefault(logger, defaultPort.Port)
	svcPort := defaultPort
	svcPort.Name = naming.PortName(name, port)
	return []corev1.ServicePort{ConstructServicePort(svcPort, port)}, nil
}

func NewSinglePortParserBuilder(name string, port int32) Builder[*SingleEndpointConfig] {
	return NewBuilder[*SingleEndpointConfig]().WithPort(port).WithName(name).WithPortParser(ParseSingleEndpoint).WithDefaultsApplier(AddressDefaulter).WithDefaultRecAddress("0.0.0.0")
}

func NewSilentSinglePortParserBuilder(name string, port int32) Builder[*SingleEndpointConfig] {
	return NewBuilder[*SingleEndpointConfig]().WithPort(port).WithName(name).WithPortParser(ParseSingleEndpointSilent).WithDefaultsApplier(AddressDefaulter).WithDefaultRecAddress("0.0.0.0")
}

func AddressDefaulter(logger logr.Logger, defaultRecAddr string, port int32, config *SingleEndpointConfig) (map[string]interface{}, error) {
	if config == nil {
		config = &SingleEndpointConfig{}
	}

	if config.Endpoint == "" {
		config.Endpoint = fmt.Sprintf("%s:%d", defaultRecAddr, port)
	} else {
		v := strings.Split(config.Endpoint, ":")
		if len(v) < 2 || v[0] == "" {
			config.Endpoint = fmt.Sprintf("%s:%s", defaultRecAddr, v[len(v)-1])
		}
	}

	res := make(map[string]interface{})
	err := mapstructure.Decode(config, &res)
	return res, err
}
