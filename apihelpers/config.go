// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apihelpers

import (
	"slices"
	"strings"

	"dario.cat/mergo"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/components"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/exporters"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/extensions"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/processors"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/receivers"
)

func getRbacRulesForComponentKinds(c *v1beta1.Config, logger logr.Logger, componentKinds ...v1beta1.ComponentKind) ([]rbacv1.PolicyRule, error) {
	var rules []rbacv1.PolicyRule
	enabledComponents := GetEnabledComponents(c)
	for _, componentKind := range componentKinds {
		var retriever components.ParserRetriever
		var cfg v1beta1.AnyConfig
		switch componentKind {
		case v1beta1.KindReceiver:
			retriever = receivers.ReceiverFor
			cfg = c.Receivers
		case v1beta1.KindExporter:
			retriever = exporters.ParserFor
			cfg = c.Exporters
		case v1beta1.KindProcessor:
			retriever = processors.ProcessorFor
			if c.Processors == nil {
				cfg = v1beta1.AnyConfig{}
			} else {
				cfg = *c.Processors
			}
		case v1beta1.KindExtension:
			retriever = extensions.ParserFor
			if c.Extensions == nil {
				cfg = v1beta1.AnyConfig{}
			} else {
				cfg = *c.Extensions
			}
		default:
			logger.V(1).Info("unknown component kind", "kind", componentKind)
			continue
		}
		for componentName := range enabledComponents[componentKind] {
			// TODO: Clean up the naming here and make it simpler to use a retriever.
			parser := retriever(componentName)
			parsedRules, err := parser.GetRBACRules(logger, cfg.Object[componentName])
			if err != nil {
				return nil, err
			}
			rules = append(rules, parsedRules...)
		}
	}
	return rules, nil
}

func getPortsForComponentKinds(c *v1beta1.Config, logger logr.Logger, componentKinds ...v1beta1.ComponentKind) ([]corev1.ServicePort, error) {
	var ports []corev1.ServicePort
	enabledComponents := GetEnabledComponents(c)
	for _, componentKind := range componentKinds {
		var retriever components.ParserRetriever
		var cfg v1beta1.AnyConfig
		switch componentKind {
		case v1beta1.KindReceiver:
			retriever = receivers.ReceiverFor
			cfg = c.Receivers
		case v1beta1.KindExporter:
			retriever = exporters.ParserFor
			cfg = c.Exporters
		case v1beta1.KindProcessor:
			continue
		case v1beta1.KindExtension:
			retriever = extensions.ParserFor
			if c.Extensions == nil {
				cfg = v1beta1.AnyConfig{}
			} else {
				cfg = *c.Extensions
			}
		}
		for componentName := range enabledComponents[componentKind] {
			// TODO: Clean up the naming here and make it simpler to use a retriever.
			parser := retriever(componentName)
			parsedPorts, err := parser.Ports(logger, componentName, cfg.Object[componentName])
			if err != nil {
				return nil, err
			}
			ports = append(ports, parsedPorts...)
		}
	}

	slices.SortFunc(ports, func(i, j corev1.ServicePort) int {
		return strings.Compare(i.Name, j.Name)
	})

	return ports, nil
}

func getEnvironmentVariablesForComponentKinds(c *v1beta1.Config, logger logr.Logger, componentKinds ...v1beta1.ComponentKind) ([]corev1.EnvVar, error) {
	envVars := []corev1.EnvVar{}
	enabledComponents := GetEnabledComponents(c)
	for _, componentKind := range componentKinds {
		var retriever components.ParserRetriever
		var cfg v1beta1.AnyConfig

		switch componentKind {
		case v1beta1.KindReceiver:
			retriever = receivers.ReceiverFor
			cfg = c.Receivers
		case v1beta1.KindExporter, v1beta1.KindProcessor, v1beta1.KindExtension:
			continue
		}
		for componentName := range enabledComponents[componentKind] {
			parser := retriever(componentName)
			parsedEnvVars, err := parser.GetEnvironmentVariables(logger, cfg.Object[componentName])
			if err != nil {
				return nil, err
			}
			envVars = append(envVars, parsedEnvVars...)
		}
	}

	slices.SortFunc(envVars, func(i, j corev1.EnvVar) int {
		return strings.Compare(i.Name, j.Name)
	})

	return envVars, nil
}

func applyDefaultForComponentKinds(c *v1beta1.Config, logger logr.Logger, parserOpts []components.DefaultOption, componentKinds ...v1beta1.ComponentKind) ([]v1beta1.EventInfo, error) {
	events, err := applyServiceDefaults(&c.Service, logger)
	if err != nil {
		return events, err
	}
	enabledComponents := GetEnabledComponents(c)
	for _, componentKind := range componentKinds {
		var retriever components.ParserRetriever
		var cfg v1beta1.AnyConfig
		switch componentKind {
		case v1beta1.KindReceiver:
			retriever = receivers.ReceiverFor
			cfg = c.Receivers
		case v1beta1.KindExporter, v1beta1.KindProcessor:
			retriever = exporters.ParserFor
			cfg = c.Exporters
		case v1beta1.KindExtension:
			if c.Extensions == nil {
				continue
			}
			retriever = extensions.ParserFor
			cfg = *c.Extensions
		}
		for componentName := range enabledComponents[componentKind] {
			parser := retriever(componentName)
			componentConf := cfg.Object[componentName]
			newCfg, err := parser.GetDefaultConfig(logger, componentConf, parserOpts...)
			if err != nil {
				return events, err
			}

			// We need to ensure we don't remove any fields in defaulting.
			mappedCfg, ok := newCfg.(map[string]any)
			if !ok || mappedCfg == nil {
				logger.V(1).Info("returned default configuration invalid",
					"warn", "could not apply component defaults",
					"component", componentName,
				)
				continue
			}

			if componentConf == nil {
				componentConf = map[string]any{}
			}
			if err := mergo.Merge(&mappedCfg, componentConf); err != nil {
				return events, err
			}
			cfg.Object[componentName] = mappedCfg
		}
	}

	return events, nil
}

func GetReceiverPorts(c *v1beta1.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getPortsForComponentKinds(c, logger, v1beta1.KindReceiver)
}

func GetExporterPorts(c *v1beta1.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getPortsForComponentKinds(c, logger, v1beta1.KindExporter)
}

func GetExtensionPorts(c *v1beta1.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getPortsForComponentKinds(c, logger, v1beta1.KindExtension)
}

func GetReceiverAndExporterPorts(c *v1beta1.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getPortsForComponentKinds(c, logger, v1beta1.KindReceiver, v1beta1.KindExporter)
}

func GetAllPorts(c *v1beta1.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getPortsForComponentKinds(c, logger, v1beta1.KindReceiver, v1beta1.KindExporter, v1beta1.KindExtension)
}

func GetEnvironmentVariables(c *v1beta1.Config, logger logr.Logger) ([]corev1.EnvVar, error) {
	return getEnvironmentVariablesForComponentKinds(c, logger, v1beta1.KindReceiver)
}

func GetAllRbacRules(c *v1beta1.Config, logger logr.Logger) ([]rbacv1.PolicyRule, error) {
	return getRbacRulesForComponentKinds(c, logger, v1beta1.KindReceiver, v1beta1.KindExporter, v1beta1.KindProcessor, v1beta1.KindExtension)
}

// ApplyDefaults applies default configuration values to the collector config.
// Optional DefaultsOption arguments can be provided to customize behavior.
func ApplyDefaults(c *v1beta1.Config, logger logr.Logger, opts ...components.DefaultOption) ([]v1beta1.EventInfo, error) {
	return applyDefaultForComponentKinds(c, logger, opts, v1beta1.KindReceiver, v1beta1.KindExporter, v1beta1.KindExtension)
}

// GetLivenessProbe gets the first enabled liveness probe. There should only ever be one extension enabled
// that provides the hinting for the liveness probe.
func GetLivenessProbe(c *v1beta1.Config, logger logr.Logger) (*corev1.Probe, error) {
	if c.Extensions == nil {
		return nil, nil
	}

	enabledComponents := GetEnabledComponents(c)
	for componentName := range enabledComponents[v1beta1.KindExtension] {
		// TODO: Clean up the naming here and make it simpler to use a retriever.
		parser := extensions.ParserFor(componentName)
		if probe, err := parser.GetLivenessProbe(logger, c.Extensions.Object[componentName]); err != nil {
			return nil, err
		} else if probe != nil {
			return probe, nil
		}
	}
	return nil, nil
}

// GetReadinessProbe gets the first enabled readiness probe. There should only ever be one extension enabled
// that provides the hinting for the readiness probe.
func GetReadinessProbe(c *v1beta1.Config, logger logr.Logger) (*corev1.Probe, error) {
	if c.Extensions == nil {
		return nil, nil
	}

	enabledComponents := GetEnabledComponents(c)
	for componentName := range enabledComponents[v1beta1.KindExtension] {
		// TODO: Clean up the naming here and make it simpler to use a retriever.
		parser := extensions.ParserFor(componentName)
		if probe, err := parser.GetReadinessProbe(logger, c.Extensions.Object[componentName]); err != nil {
			return nil, err
		} else if probe != nil {
			return probe, nil
		}
	}
	return nil, nil
}

// GetStartupProbe gets the first enabled startup probe. There should only ever be one extension enabled
// that provides the hinting for the startup probe.
func GetStartupProbe(c *v1beta1.Config, logger logr.Logger) (*corev1.Probe, error) {
	if c.Extensions == nil {
		return nil, nil
	}

	enabledComponents := GetEnabledComponents(c)
	for componentName := range enabledComponents[v1beta1.KindExtension] {
		// TODO: Clean up the naming here and make it simpler to use a retriever.
		parser := extensions.ParserFor(componentName)
		if probe, err := parser.GetStartupProbe(logger, c.Extensions.Object[componentName]); err != nil {
			return nil, err
		} else if probe != nil {
			return probe, nil
		}
	}
	return nil, nil
}
