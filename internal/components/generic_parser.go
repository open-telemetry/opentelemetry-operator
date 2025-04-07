// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

var (
	_ Parser = &GenericParser[SingleEndpointConfig]{}
)

// GenericParser serves as scaffolding for custom parsing logic by isolating
// functionality to idempotent functions.
type GenericParser[T any] struct {
	name                string
	settings            *Settings[T]
	portParser          PortParser[T]
	clusterRoleRulesGen ClusterRoleRulesGenerator[T]
	roleGen             RoleGenerator[T]
	roleBindingGen      RoleBindingGenerator[T]
	envVarGen           EnvVarGenerator[T]
	livenessGen         ProbeGenerator[T]
	readinessGen        ProbeGenerator[T]
	defaultsApplier     Defaulter[T]
}

func (g *GenericParser[T]) GetDefaultConfig(logger logr.Logger, config interface{}) (interface{}, error) {
	if g.settings == nil || g.defaultsApplier == nil {
		return config, nil
	}

	if g.settings.defaultRecAddr == "" || g.settings.port == 0 {
		return config, nil
	}

	var parsed T
	if err := mapstructure.Decode(config, &parsed); err != nil {
		return nil, err
	}
	return g.defaultsApplier(logger, g.settings.defaultRecAddr, g.settings.port, parsed)
}

func (g *GenericParser[T]) GetLivenessProbe(logger logr.Logger, config interface{}) (*corev1.Probe, error) {
	if g.livenessGen == nil {
		return nil, nil
	}
	var parsed T
	if err := mapstructure.Decode(config, &parsed); err != nil {
		return nil, err
	}
	return g.livenessGen(logger, parsed)
}

func (g *GenericParser[T]) GetReadinessProbe(logger logr.Logger, config interface{}) (*corev1.Probe, error) {
	if g.readinessGen == nil {
		return nil, nil
	}
	var parsed T
	if err := mapstructure.Decode(config, &parsed); err != nil {
		return nil, err
	}
	return g.readinessGen(logger, parsed)
}

func (g *GenericParser[T]) GetClusterRoleRules(logger logr.Logger, config interface{}) ([]rbacv1.PolicyRule, error) {
	if g.clusterRoleRulesGen == nil {
		return nil, nil
	}
	var parsed T
	if err := mapstructure.Decode(config, &parsed); err != nil {
		return nil, err
	}
	return g.clusterRoleRulesGen(logger, parsed)
}

func (g *GenericParser[T]) GetRbacRoles(logger logr.Logger, otelCollectorName string, config interface{}) ([]*rbacv1.Role, error) {
	if g.roleGen == nil {
		return nil, nil
	}
	var parsed T
	if err := mapstructure.Decode(config, &parsed); err != nil {
		return nil, err
	}
	return g.roleGen(logger, parsed, g.name, otelCollectorName)
}

func (g *GenericParser[T]) GetRbacRoleBindings(logger logr.Logger, otelCollectorName string, config interface{}, serviceAccountName string, otelCollectorNamespace string) ([]*rbacv1.RoleBinding, error) {
	if g.roleBindingGen == nil {
		return nil, nil
	}
	var parsed T
	if err := mapstructure.Decode(config, &parsed); err != nil {
		return nil, err
	}

	return g.roleBindingGen(logger, parsed, g.name, serviceAccountName, otelCollectorName, otelCollectorNamespace)
}

func (g *GenericParser[T]) GetEnvironmentVariables(logger logr.Logger, config interface{}) ([]corev1.EnvVar, error) {
	if g.envVarGen == nil {
		return nil, nil
	}
	var parsed T
	if err := mapstructure.Decode(config, &parsed); err != nil {
		return nil, err
	}
	return g.envVarGen(logger, parsed)
}

func (g *GenericParser[T]) Ports(logger logr.Logger, name string, config interface{}) ([]corev1.ServicePort, error) {
	if g.portParser == nil {
		return nil, nil
	}
	var parsed T
	if err := mapstructure.Decode(config, &parsed); err != nil {
		return nil, err
	}
	return g.portParser(logger, name, g.settings.GetServicePort(), parsed)
}

func (g *GenericParser[T]) ParserType() string {
	return ComponentType(g.name)
}

func (g *GenericParser[T]) ParserName() string {
	return fmt.Sprintf("__%s", g.name)
}
