// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receivers

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

// prometheusConfig represents the Prometheus receiver config relevant for port parsing.
// The api_server section opens an inbound HTTP listener whose port must be exposed
// in the Kubernetes Service and NetworkPolicy.
type prometheusConfig struct {
	ApiServer *apiServerConfig `mapstructure:"api_server,omitempty"`
}

type apiServerConfig struct {
	Enabled      bool          `mapstructure:"enabled,omitempty"`
	ServerConfig *serverConfig `mapstructure:"server_config,omitempty"`
}

type serverConfig struct {
	Endpoint string `mapstructure:"endpoint,omitempty"`
}

func (c *prometheusConfig) GetPortNum() (int32, error) {
	if c.ApiServer == nil || !c.ApiServer.Enabled {
		return components.UnsetPort, components.PortNotFoundErr
	}
	if c.ApiServer.ServerConfig != nil && c.ApiServer.ServerConfig.Endpoint != "" {
		return components.PortFromEndpoint(c.ApiServer.ServerConfig.Endpoint)
	}
	return components.UnsetPort, components.PortNotFoundErr
}

func (c *prometheusConfig) GetPortNumOrDefault(logger logr.Logger, p int32) int32 {
	num, err := c.GetPortNum()
	if err != nil {
		logger.V(3).Info("no port set, using default", "port", p)
		return p
	}
	return num
}

func parsePrometheusPort(logger logr.Logger, name string, defaultPort *corev1.ServicePort, cfg *prometheusConfig) ([]corev1.ServicePort, error) {
	if cfg == nil {
		return nil, nil
	}
	if _, err := cfg.GetPortNum(); err != nil {
		// No api_server port configured or not enabled — this is normal for prometheus receiver.
		return nil, nil
	}
	port := cfg.GetPortNumOrDefault(logger, defaultPort.Port)
	svcPort := defaultPort
	svcPort.Name = naming.PortName(name, port)
	return []corev1.ServicePort{components.ConstructServicePort(svcPort, port)}, nil
}

// NewPrometheusParser returns a parser for the prometheus receiver that extracts
// the api_server.server_config.endpoint port for Service and NetworkPolicy exposure.
func NewPrometheusParser() *components.GenericParser[*prometheusConfig] {
	return components.NewBuilder[*prometheusConfig]().
		WithName("prometheus").
		WithPort(components.UnsetPort).
		WithPortParser(parsePrometheusPort).
		MustBuild()
}
