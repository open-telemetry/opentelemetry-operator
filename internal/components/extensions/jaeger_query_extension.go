// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

const (
	name         = "jaeger_query"
	grpcPortName = "jq-grpc"
	httpPort     = 16686
)

var (
	_ components.Parser = &components.GenericParser[*JaegerQueryExtensionConfig]{}
)

type JaegerQueryExtensionConfig struct {
	HTTP jaegerHTTPAddress `mapstructure:"http,omitempty" yaml:"http,omitempty"`
	GRPC jaegerGRPCAddress `mapstructure:"grpc,omitempty" yaml:"grpc,omitempty"`
}

type jaegerHTTPAddress struct {
	Endpoint string `mapstructure:"endpoint,omitempty" yaml:"endpoint,omitempty"`
}

type jaegerGRPCAddress struct {
	Endpoint string `mapstructure:"endpoint,omitempty" yaml:"endpoint,omitempty"`
}

func (g *JaegerQueryExtensionConfig) GetHTTPPortNumOrDefault(logger logr.Logger, p int32) int32 {
	num, err := g.GetHTTPPortNum()
	if err != nil {
		logger.V(3).Info("no port set, using default: %d", p)
		return p
	}
	return num
}

// GetHTTPPortNum attempts to get the port for the given config. If it cannot, the UnsetPort and the given missingPortError
// are returned.
func (g *JaegerQueryExtensionConfig) GetHTTPPortNum() (int32, error) {
	if len(g.HTTP.Endpoint) > 0 {
		return components.PortFromEndpoint(g.HTTP.Endpoint)
	}
	return components.UnsetPort, components.PortNotFoundErr
}

func (g *JaegerQueryExtensionConfig) GetGRPCPortNum() (int32, error) {
	if len(g.GRPC.Endpoint) > 0 {
		return components.PortFromEndpoint(g.GRPC.Endpoint)
	}
	return components.UnsetPort, components.PortNotFoundErr
}

func ParseJaegerQueryExtensionConfig(logger logr.Logger, name string, defaultPort *corev1.ServicePort, cfg *JaegerQueryExtensionConfig) ([]corev1.ServicePort, error) {
	if cfg == nil {
		return nil, nil
	}

	httpPortNum := cfg.GetHTTPPortNumOrDefault(logger, defaultPort.Port)
	grpcPortNum := components.UnsetPort
	if p, err := cfg.GetGRPCPortNum(); err == nil {
		grpcPortNum = p
	}

	if httpPortNum == components.UnsetPort && grpcPortNum == components.UnsetPort {
		logger.WithValues("receiver", defaultPort.Name).Error(components.PortNotFoundErr, "couldn't parse the endpoint's port and no default port set")
		return []corev1.ServicePort{}, components.PortNotFoundErr
	}

	var ports []corev1.ServicePort

	// - Preserve HTTP port name as "jaeger-query" for backward compatibility.
	// - Use a short, stable name for gRPC ("jq-grpc") to avoid the 15-char Kubernetes limit.
	if httpPortNum != components.UnsetPort {
		httpSvcPort := *defaultPort
		httpSvcPort.Name = naming.PortName(name, httpPortNum)
		ports = append(ports, components.ConstructServicePort(&httpSvcPort, httpPortNum))
	}

	if grpcPortNum != components.UnsetPort {
		grpcSvcPort := *defaultPort
		grpcSvcPort.Name = naming.PortName(grpcPortName, grpcPortNum)
		ports = append(ports, components.ConstructServicePort(&grpcSvcPort, grpcPortNum))
	}

	return ports, nil
}

func NewJaegerQueryExtensionParserBuilder() components.Builder[*JaegerQueryExtensionConfig] {
	return components.NewBuilder[*JaegerQueryExtensionConfig]().WithPort(httpPort).WithName(name).WithPortParser(ParseJaegerQueryExtensionConfig).WithDefaultsApplier(endpointDefaulter).WithDefaultRecAddress(components.DefaultRecAddress).WithTargetPort(httpPort)
}

func endpointDefaulter(_ logr.Logger, defaultRecAddr string, port int32, config *JaegerQueryExtensionConfig) (map[string]interface{}, error) {
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

	// Apply address defaulting for gRPC only if an endpoint is provided
	if config.GRPC.Endpoint != "" {
		v := strings.Split(config.GRPC.Endpoint, ":")
		if len(v) < 2 || v[0] == "" {
			config.GRPC.Endpoint = fmt.Sprintf("%s:%s", defaultRecAddr, v[len(v)-1])
		}
	}

	res := make(map[string]interface{})
	err := mapstructure.Decode(config, &res)
	// Avoid emitting empty grpc map when not configured
	if config.GRPC.Endpoint == "" {
		if m, ok := res["grpc"].(map[string]interface{}); ok && len(m) == 0 {
			delete(res, "grpc")
		}
	}
	return res, err
}
