// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

const (
	name = "jaeger_query"
	port = 16686
)

var _ components.Parser = &components.GenericParser[*JaegerQueryExtensionConfig]{}

type JaegerQueryExtensionConfig struct {
	HTTP jaegerHTTPAddress `mapstructure:"http,omitempty" yaml:"http,omitempty"`
	GRPC jaegerGRPCAddress `mapstructure:"grpc,omitempty" yaml:"grpc,omitempty"`
}

type jaegerHTTPAddress struct {
	Endpoint string                `mapstructure:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	TLS      *components.TLSConfig `mapstructure:"tls,omitempty" yaml:"tls,omitempty"`
}

type jaegerGRPCAddress struct {
	Endpoint string `mapstructure:"endpoint,omitempty" yaml:"endpoint,omitempty"`
}

func (g *JaegerQueryExtensionConfig) GetPortNumOrDefault(logger logr.Logger, p int32) int32 {
	num, err := g.GetPortNum()
	if err != nil {
		logger.V(3).Info("no port set, using default: %d", p)
		return p
	}
	return num
}

// GetPortNum attempts to get the port for the given config. If it cannot, the UnsetPort and the given missingPortError
// are returned.
func (g *JaegerQueryExtensionConfig) GetPortNum() (int32, error) {
	if g.HTTP.Endpoint != "" {
		return components.PortFromEndpoint(g.HTTP.Endpoint)
	}
	return components.UnsetPort, components.PortNotFoundErr
}

func ParseJaegerQueryExtensionConfig(logger logr.Logger, name string, defaultPort *corev1.ServicePort, cfg *JaegerQueryExtensionConfig) ([]corev1.ServicePort, error) {
	if cfg == nil {
		return nil, nil
	}
	if _, err := cfg.GetPortNum(); err != nil && defaultPort.Port == components.UnsetPort {
		logger.WithValues("receiver", defaultPort.Name).Error(err, "couldn't parse the endpoint's port and no default port set")
		return []corev1.ServicePort{}, err
	}
	httpPort := cfg.GetPortNumOrDefault(logger, defaultPort.Port)
	svcPort := defaultPort
	svcPort.Name = naming.PortName(name, httpPort)
	ports := []corev1.ServicePort{components.ConstructServicePort(svcPort, httpPort)}

	// Add gRPC port if explicitly configured
	if cfg.GRPC.Endpoint != "" {
		grpcPortNum, err := components.PortFromEndpoint(cfg.GRPC.Endpoint)
		if err != nil {
			logger.WithValues("extension", name).Error(err, "couldn't parse the gRPC endpoint's port")
		} else if grpcPortNum != httpPort {
			// Only add gRPC port if it differs from the HTTP port
			grpcSvcPort := &corev1.ServicePort{
				TargetPort: intstr.FromInt32(grpcPortNum),
			}
			grpcSvcPort.Name = naming.PortName(fmt.Sprintf("%s-grpc", name), grpcPortNum)
			ports = append(ports, components.ConstructServicePort(grpcSvcPort, grpcPortNum))
		}
	}

	return ports, nil
}

func NewJaegerQueryExtensionParserBuilder() components.Builder[*JaegerQueryExtensionConfig] {
	return components.NewBuilder[*JaegerQueryExtensionConfig]().WithPort(port).WithName(name).WithPortParser(ParseJaegerQueryExtensionConfig).WithDefaultsApplier(endpointDefaulter).WithDefaultRecAddress(components.DefaultRecAddress).WithTargetPort(port)
}

func endpointDefaulter(_ logr.Logger, defaultCfg *components.DefaultConfig, defaultRecAddr string, port int32, config *JaegerQueryExtensionConfig) (map[string]any, error) {
	if config == nil {
		config = &JaegerQueryExtensionConfig{}
	}

	if config.HTTP.Endpoint == "" {
		config.HTTP.Endpoint = fmt.Sprintf("%s:%d", defaultRecAddr, port)
	} else {
		v := strings.Split(config.HTTP.Endpoint, ":")
		if len(v) < 2 || v[0] == "" {
			config.HTTP.Endpoint = fmt.Sprintf("%s:%s", defaultRecAddr, v[len(v)-1])
		}
	}

	// Apply default host for gRPC endpoint if configured but missing host
	if config.GRPC.Endpoint != "" {
		v := strings.Split(config.GRPC.Endpoint, ":")
		if len(v) < 2 || v[0] == "" {
			config.GRPC.Endpoint = fmt.Sprintf("%s:%s", defaultRecAddr, v[len(v)-1])
		}
	}

	config.HTTP.TLS.ApplyTLSProfileDefaults(defaultCfg.TLSProfile)

	res := make(map[string]any)
	err := mapstructure.Decode(config, &res)
	// Remove empty gRPC config to avoid injecting unwanted configuration
	if config.GRPC.Endpoint == "" {
		delete(res, "grpc")
	}
	return res, err
}
