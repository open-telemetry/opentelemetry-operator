// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"fmt"
	"net"

	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
)

const (
	defaultHealthcheckV2Path = "/health/status"
	defaultHealthcheckV2Port = 13133
)

// healthcheckV2HTTPConfig represents the HTTP configuration for healthcheck v2.
type healthcheckV2HTTPConfig struct {
	Endpoint string `mapstructure:"endpoint,omitempty"`
	Status   struct {
		Enabled *bool  `mapstructure:"enabled,omitempty"`
		Path    string `mapstructure:"path,omitempty"`
	} `mapstructure:"status,omitempty"`
}

// healthcheckV2Config represents the configuration for the healthcheck v2 extension.
// See: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/healthcheckv2extension
type healthcheckV2Config struct {
	UseV2 bool                    `mapstructure:"use_v2,omitempty"`
	HTTP  healthcheckV2HTTPConfig `mapstructure:"http,omitempty"`
	GRPC  struct {
		Endpoint string `mapstructure:"endpoint,omitempty"`
	} `mapstructure:"grpc,omitempty"`
}

// GetPortNumOrDefault returns the port number from HTTP endpoint or a default value.
func (c *healthcheckV2Config) GetPortNumOrDefault(logger logr.Logger, defaultPort int32) int32 {
	if c.HTTP.Endpoint != "" {
		port, err := components.PortFromEndpoint(c.HTTP.Endpoint)
		if err == nil {
			return port
		}
	}
	logger.V(3).Info("no port set for healthcheckv2, using default", "default", defaultPort)
	return defaultPort
}

func healthCheckV2AddressDefaulter(logger logr.Logger, defaultRecAddr string, port int32, config healthcheckV2Config) (map[string]interface{}, error) {
	if config.HTTP.Endpoint == "" {
		config.HTTP.Endpoint = fmt.Sprintf("%s:%d", defaultRecAddr, port)
	} else {
		h, p, err := net.SplitHostPort(config.HTTP.Endpoint)
		if err == nil && h == "" && p != "" {
			config.HTTP.Endpoint = fmt.Sprintf("%s:%s", defaultRecAddr, p)
		}
	}

	if config.HTTP.Status.Path == "" {
		config.HTTP.Status.Path = defaultHealthcheckV2Path
	}

	res := make(map[string]interface{})
	err := mapstructure.Decode(config, &res)
	return res, err
}

// healthCheckV2Probe returns the probe configuration for the healthcheck v2 extension.
func healthCheckV2Probe(logger logr.Logger, config healthcheckV2Config) (*corev1.Probe, error) {
	path := config.HTTP.Status.Path
	if len(path) == 0 {
		path = defaultHealthcheckV2Path
	}

	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: path,
				Port: intstr.FromInt32(config.GetPortNumOrDefault(logger, defaultHealthcheckV2Port)),
			},
		},
	}, nil
}

// healthCheckV2PortParser parses the ports for healthcheck v2 extension.
func healthCheckV2PortParser(logger logr.Logger, name string, defaultPort *corev1.ServicePort, config healthcheckV2Config) ([]corev1.ServicePort, error) {
	singleEndpointConfig := &components.SingleEndpointConfig{
		Endpoint: config.HTTP.Endpoint,
	}
	return components.ParseSingleEndpointSilent(logger, name, defaultPort, singleEndpointConfig)
}
