// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	GrpcProtocol          = "grpc"
	HttpProtocol          = "http"
	UnsetPort       int32 = 0
	PortNotFoundErr       = errors.New("port should not be empty")
)

type PortRetriever interface {
	GetPortNum() (int32, error)
	GetPortNumOrDefault(logr.Logger, int32) int32
}

// PortParser is a function that returns a list of servicePorts given a config of type Config.
type PortParser[ComponentConfigType any] func(logger logr.Logger, name string, defaultPort *corev1.ServicePort, config ComponentConfigType) ([]corev1.ServicePort, error)

// ClusterRoleRulesGenerator is a function that generates a list of RBAC Rules given a configuration of type Config
// It's expected that type Config is the configuration used by a parser.
type ClusterRoleRulesGenerator[ComponentConfigType any] func(logger logr.Logger, config ComponentConfigType) ([]rbacv1.PolicyRule, error)

// RoleGenerator is a function that generates a list of RBACRoles given a configuration of type Config
// It's expected that type Config is the configuration used by a parser.
type RoleGenerator[ComponentConfigType any] func(logger logr.Logger, config ComponentConfigType, componentName string, otelCollectorName string) ([]*rbacv1.Role, error)

// RoleBindingGenerator is a function that generates a list of RBACRoleBindings given a configuration of type Config
// It's expected that type Config is the configuration used by a parser.
type RoleBindingGenerator[ComponentConfigType any] func(logger logr.Logger, config ComponentConfigType, componentName string, serviceAccountName string, otelCollectorName string, otelCollectorNamespace string) ([]*rbacv1.RoleBinding, error)

// ProbeGenerator is a function that generates a valid probe for a container given Config
// It's expected that type Config is the configuration used by a parser.
type ProbeGenerator[ComponentConfigType any] func(logger logr.Logger, config ComponentConfigType) (*corev1.Probe, error)

// EnvVarGenerator is a function that generates a list of environment variables for a given config.
// It's expected that type Config is the configuration used by a parser.
type EnvVarGenerator[ComponentConfigType any] func(logger logr.Logger, config ComponentConfigType) ([]corev1.EnvVar, error)

// Defaulter is a function that applies given defaults to the passed Config.
// It's expected that type Config is the configuration used by a parser.
type Defaulter[ComponentConfigType any] func(logger logr.Logger, defaultAddr string, defaultPort int32, config ComponentConfigType) (map[string]interface{}, error)

// ComponentType returns the type for a given component name.
// components have a name like:
// - mycomponent/custom
// - mycomponent
// we extract the "mycomponent" part and see if we have a parser for the component.
func ComponentType(name string) string {
	if strings.Contains(name, "/") {
		return name[:strings.Index(name, "/")]
	}
	return name
}

func PortFromEndpoint(endpoint string) (int32, error) {
	var err error
	var port int64

	r := regexp.MustCompile(":[0-9]+")

	if r.MatchString(endpoint) {
		portStr := r.FindString(endpoint)
		cleanedPortStr := strings.Replace(portStr, ":", "", -1)
		port, err = strconv.ParseInt(cleanedPortStr, 10, 32)

		if err != nil {
			return UnsetPort, err
		}
	}

	if port == 0 {
		return UnsetPort, PortNotFoundErr
	}

	return int32(port), err //nolint: gosec // disable G115, this is guaranteed to not overflow due to the bitSize in the ParseInt call
}

type ParserRetriever func(string) Parser

type Parser interface {
	// GetDefaultConfig returns a config with set default values.
	// NOTE: Config merging must be done by the caller if desired.
	GetDefaultConfig(logger logr.Logger, config interface{}) (interface{}, error)

	// Ports returns the service ports parsed based on the component's configuration where name is the component's name
	// of the form "name" or "type/name"
	Ports(logger logr.Logger, name string, config interface{}) ([]corev1.ServicePort, error)

	// GetClusterRoleRules returns the rbac rules for this component
	GetClusterRoleRules(logger logr.Logger, config interface{}) ([]rbacv1.PolicyRule, error)

	// GetRbacRoleBindings returns the rbac role bindings for this component
	GetRbacRoleBindings(logger logr.Logger, otelCollectorName string, config interface{}, serviceAccountName string, otelCollectorNamespace string) ([]*rbacv1.RoleBinding, error)

	// GetRbacRoles returns the rbac roles for this component
	GetRbacRoles(logger logr.Logger, otelCollectorName string, config interface{}) ([]*rbacv1.Role, error)

	// GetLivenessProbe returns a liveness probe set for the collector
	GetLivenessProbe(logger logr.Logger, config interface{}) (*corev1.Probe, error)

	// GetEnvironmentVariables returns a list of environment variables for the collector
	GetEnvironmentVariables(logger logr.Logger, config interface{}) ([]corev1.EnvVar, error)

	// GetReadinessProbe returns a readiness probe set for the collector
	GetReadinessProbe(logger logr.Logger, config interface{}) (*corev1.Probe, error)

	// ParserType returns the type of this parser
	ParserType() string

	// ParserName is an internal name for the parser
	ParserName() string
}

func ConstructServicePort(current *corev1.ServicePort, port int32) corev1.ServicePort {
	svc := corev1.ServicePort{
		Name:        current.Name,
		Port:        port,
		NodePort:    current.NodePort,
		AppProtocol: current.AppProtocol,
		Protocol:    current.Protocol,
	}

	if port > 0 && current.TargetPort.IntValue() > 0 {
		svc.TargetPort = intstr.FromInt32(port)
	}
	return svc
}

func GetPortsForConfig(logger logr.Logger, config map[string]interface{}, retriever ParserRetriever) ([]corev1.ServicePort, error) {
	var ports []corev1.ServicePort
	for componentName, componentDef := range config {
		parser := retriever(componentName)
		if parsedPorts, err := parser.Ports(logger, componentName, componentDef); err != nil {
			return nil, err
		} else {
			ports = append(ports, parsedPorts...)
		}
	}
	return ports, nil
}
