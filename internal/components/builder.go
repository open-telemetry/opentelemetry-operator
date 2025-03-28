// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package components

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

type ParserOption[ComponentConfigType any] func(*Settings[ComponentConfigType])

type Settings[ComponentConfigType any] struct {
	protocol            corev1.Protocol
	appProtocol         *string
	targetPort          intstr.IntOrString
	nodePort            int32
	name                string
	port                int32
	defaultRecAddr      string
	portParser          PortParser[ComponentConfigType]
	clusterRoleRulesGen ClusterRoleRulesGenerator[ComponentConfigType]
	roleGen             RoleGenerator[ComponentConfigType]
	roleBindingGen      RoleBindingGenerator[ComponentConfigType]
	livenessGen         ProbeGenerator[ComponentConfigType]
	readinessGen        ProbeGenerator[ComponentConfigType]
	defaultsApplier     Defaulter[ComponentConfigType]
	envVarGen           EnvVarGenerator[ComponentConfigType]
}

func NewEmptySettings[ComponentConfigType any]() *Settings[ComponentConfigType] {
	return &Settings[ComponentConfigType]{}
}

func (o *Settings[ComponentConfigType]) Apply(opts ...ParserOption[ComponentConfigType]) {
	for _, opt := range opts {
		opt(o)
	}
}

func (o *Settings[ComponentConfigType]) GetServicePort() *corev1.ServicePort {
	return &corev1.ServicePort{
		Name:        naming.PortName(o.name, o.port),
		Port:        o.port,
		Protocol:    o.protocol,
		AppProtocol: o.appProtocol,
		TargetPort:  o.targetPort,
		NodePort:    o.nodePort,
	}
}

type Builder[ComponentConfigType any] []ParserOption[ComponentConfigType]

func NewBuilder[ComponentConfigType any]() Builder[ComponentConfigType] {
	return []ParserOption[ComponentConfigType]{}
}

func (b Builder[ComponentConfigType]) WithProtocol(protocol corev1.Protocol) Builder[ComponentConfigType] {
	return append(b, func(o *Settings[ComponentConfigType]) {
		o.protocol = protocol
	})
}
func (b Builder[ComponentConfigType]) WithAppProtocol(appProtocol *string) Builder[ComponentConfigType] {
	return append(b, func(o *Settings[ComponentConfigType]) {
		o.appProtocol = appProtocol
	})
}
func (b Builder[ComponentConfigType]) WithDefaultRecAddress(defaultRecAddr string) Builder[ComponentConfigType] {
	return append(b, func(o *Settings[ComponentConfigType]) {
		o.defaultRecAddr = defaultRecAddr
	})
}
func (b Builder[ComponentConfigType]) WithTargetPort(targetPort int32) Builder[ComponentConfigType] {
	return append(b, func(o *Settings[ComponentConfigType]) {
		o.targetPort = intstr.FromInt32(targetPort)
	})
}
func (b Builder[ComponentConfigType]) WithNodePort(nodePort int32) Builder[ComponentConfigType] {
	return append(b, func(o *Settings[ComponentConfigType]) {
		o.nodePort = nodePort
	})
}
func (b Builder[ComponentConfigType]) WithName(name string) Builder[ComponentConfigType] {
	return append(b, func(o *Settings[ComponentConfigType]) {
		o.name = name
	})
}
func (b Builder[ComponentConfigType]) WithPort(port int32) Builder[ComponentConfigType] {
	return append(b, func(o *Settings[ComponentConfigType]) {
		o.port = port
	})
}
func (b Builder[ComponentConfigType]) WithPortParser(portParser PortParser[ComponentConfigType]) Builder[ComponentConfigType] {
	return append(b, func(o *Settings[ComponentConfigType]) {
		o.portParser = portParser
	})
}
func (b Builder[ComponentConfigType]) WithClusterRoleRulesGen(clusterRoleRulesGen ClusterRoleRulesGenerator[ComponentConfigType]) Builder[ComponentConfigType] {
	return append(b, func(o *Settings[ComponentConfigType]) {
		o.clusterRoleRulesGen = clusterRoleRulesGen
	})
}
func (b Builder[ComponentConfigType]) WithRoleGen(roleGen RoleGenerator[ComponentConfigType]) Builder[ComponentConfigType] {
	return append(b, func(o *Settings[ComponentConfigType]) {
		o.roleGen = roleGen
	})
}
func (b Builder[ComponentConfigType]) WithRoleBindingGen(roleBindingGen RoleBindingGenerator[ComponentConfigType]) Builder[ComponentConfigType] {
	return append(b, func(o *Settings[ComponentConfigType]) {
		o.roleBindingGen = roleBindingGen
	})
}

func (b Builder[ComponentConfigType]) WithLivenessGen(livenessGen ProbeGenerator[ComponentConfigType]) Builder[ComponentConfigType] {
	return append(b, func(o *Settings[ComponentConfigType]) {
		o.livenessGen = livenessGen
	})
}

func (b Builder[ComponentConfigType]) WithReadinessGen(readinessGen ProbeGenerator[ComponentConfigType]) Builder[ComponentConfigType] {
	return append(b, func(o *Settings[ComponentConfigType]) {
		o.readinessGen = readinessGen
	})
}
func (b Builder[ComponentConfigType]) WithEnvVarGen(envVarGen EnvVarGenerator[ComponentConfigType]) Builder[ComponentConfigType] {
	return append(b, func(o *Settings[ComponentConfigType]) {
		o.envVarGen = envVarGen
	})
}
func (b Builder[ComponentConfigType]) WithDefaultsApplier(defaultsApplier Defaulter[ComponentConfigType]) Builder[ComponentConfigType] {
	return append(b, func(o *Settings[ComponentConfigType]) {
		o.defaultsApplier = defaultsApplier
	})
}

func (b Builder[ComponentConfigType]) Build() (*GenericParser[ComponentConfigType], error) {
	o := NewEmptySettings[ComponentConfigType]()
	o.Apply(b...)
	if len(o.name) == 0 {
		return nil, fmt.Errorf("invalid settings struct, no name specified")
	}
	return &GenericParser[ComponentConfigType]{
		name:                o.name,
		portParser:          o.portParser,
		clusterRoleRulesGen: o.clusterRoleRulesGen,
		roleGen:             o.roleGen,
		roleBindingGen:      o.roleBindingGen,
		envVarGen:           o.envVarGen,
		livenessGen:         o.livenessGen,
		readinessGen:        o.readinessGen,
		defaultsApplier:     o.defaultsApplier,
		settings:            o,
	}, nil
}

func (b Builder[ComponentConfigType]) MustBuild() *GenericParser[ComponentConfigType] {
	if p, err := b.Build(); err != nil {
		panic(err)
	} else {
		return p
	}
}
