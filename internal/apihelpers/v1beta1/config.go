// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apihelpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"dario.cat/mergo"
	"github.com/go-logr/logr"
	go_yaml "github.com/goccy/go-yaml"
	otelConfig "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/components"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/exporters"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/extensions"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/processors"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/receivers"
)

// ComponentKind is the type of a component (receiver, exporter, processor, extension).
type ComponentKind int

const (
	KindReceiver  ComponentKind = iota
	KindExporter
	KindProcessor
	KindExtension
)

func (c ComponentKind) String() string {
	return [...]string{"receiver", "exporter", "processor", "extension"}[c]
}

// EventInfo represents an event to be recorded.
type EventInfo struct {
	Type    string
	Reason  string
	Message string
}

const (
	defaultServiceHost = "0.0.0.0"
)

// GetEnabledComponents constructs a list of enabled components by component type.
func GetEnabledComponents(c *v1beta1.Config) map[ComponentKind]map[string]any {
	toReturn := map[ComponentKind]map[string]any{
		KindReceiver:  {},
		KindProcessor: {},
		KindExporter:  {},
		KindExtension: {},
	}
	for _, extension := range c.Service.Extensions {
		toReturn[KindExtension][extension] = struct{}{}
	}

	for _, pipeline := range c.Service.Pipelines {
		if pipeline == nil {
			continue
		}
		for _, componentId := range pipeline.Receivers {
			toReturn[KindReceiver][componentId] = struct{}{}
		}
		for _, componentId := range pipeline.Exporters {
			toReturn[KindExporter][componentId] = struct{}{}
		}
		for _, componentId := range pipeline.Processors {
			toReturn[KindProcessor][componentId] = struct{}{}
		}
	}
	for _, componentId := range c.Service.Extensions {
		toReturn[KindExtension][componentId] = struct{}{}
	}
	return toReturn
}

// getRbacRulesForComponentKinds gets the RBAC Rules for the given ComponentKind(s).
func getRbacRulesForComponentKinds(c *v1beta1.Config, logger logr.Logger, componentKinds ...ComponentKind) ([]rbacv1.PolicyRule, error) {
	var rules []rbacv1.PolicyRule
	enabledComponents := GetEnabledComponents(c)
	for _, componentKind := range componentKinds {
		var retriever components.ParserRetriever
		var cfg v1beta1.AnyConfig
		switch componentKind {
		case KindReceiver:
			retriever = receivers.ReceiverFor
			cfg = c.Receivers
		case KindExporter:
			retriever = exporters.ParserFor
			cfg = c.Exporters
		case KindProcessor:
			retriever = processors.ProcessorFor
			if c.Processors == nil {
				cfg = v1beta1.AnyConfig{}
			} else {
				cfg = *c.Processors
			}
		case KindExtension:
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

// getPortsForComponentKinds gets the ports for the given ComponentKind(s).
func getPortsForComponentKinds(c *v1beta1.Config, logger logr.Logger, componentKinds ...ComponentKind) ([]corev1.ServicePort, error) {
	var ports []corev1.ServicePort
	enabledComponents := GetEnabledComponents(c)
	for _, componentKind := range componentKinds {
		var retriever components.ParserRetriever
		var cfg v1beta1.AnyConfig
		switch componentKind {
		case KindReceiver:
			retriever = receivers.ReceiverFor
			cfg = c.Receivers
		case KindExporter:
			retriever = exporters.ParserFor
			cfg = c.Exporters
		case KindProcessor:
			continue
		case KindExtension:
			retriever = extensions.ParserFor
			if c.Extensions == nil {
				cfg = v1beta1.AnyConfig{}
			} else {
				cfg = *c.Extensions
			}
		}
		for componentName := range enabledComponents[componentKind] {
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

// getEnvironmentVariablesForComponentKinds gets the environment variables for the given ComponentKind(s).
func getEnvironmentVariablesForComponentKinds(c *v1beta1.Config, logger logr.Logger, componentKinds ...ComponentKind) ([]corev1.EnvVar, error) {
	envVars := []corev1.EnvVar{}
	enabledComponents := GetEnabledComponents(c)
	for _, componentKind := range componentKinds {
		var retriever components.ParserRetriever
		var cfg v1beta1.AnyConfig

		switch componentKind {
		case KindReceiver:
			retriever = receivers.ReceiverFor
			cfg = c.Receivers
		case KindExporter, KindProcessor, KindExtension:
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

// applyDefaultForComponentKinds applies defaults to the endpoints for the given ComponentKind(s).
func applyDefaultForComponentKinds(c *v1beta1.Config, logger logr.Logger, parserOpts []components.DefaultOption, componentKinds ...ComponentKind) ([]EventInfo, error) {
	events, err := ServiceApplyDefaults(&c.Service, logger)
	if err != nil {
		return events, err
	}
	enabledComponents := GetEnabledComponents(c)
	for _, componentKind := range componentKinds {
		var retriever components.ParserRetriever
		var cfg v1beta1.AnyConfig
		switch componentKind {
		case KindReceiver:
			retriever = receivers.ReceiverFor
			cfg = c.Receivers
		case KindExporter, KindProcessor:
			retriever = exporters.ParserFor
			cfg = c.Exporters
		case KindExtension:
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

// GetReceiverPorts returns the ports for the receivers.
func GetReceiverPorts(c *v1beta1.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getPortsForComponentKinds(c, logger, KindReceiver)
}

// GetExporterPorts returns the ports for the exporters.
func GetExporterPorts(c *v1beta1.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getPortsForComponentKinds(c, logger, KindExporter)
}

// GetExtensionPorts returns the ports for the extensions.
func GetExtensionPorts(c *v1beta1.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getPortsForComponentKinds(c, logger, KindExtension)
}

// GetReceiverAndExporterPorts returns the ports for the receivers and exporters.
func GetReceiverAndExporterPorts(c *v1beta1.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getPortsForComponentKinds(c, logger, KindReceiver, KindExporter)
}

// GetAllPorts returns all ports for the receivers, exporters, and extensions.
func GetAllPorts(c *v1beta1.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getPortsForComponentKinds(c, logger, KindReceiver, KindExporter, KindExtension)
}

// GetEnvironmentVariables returns the environment variables for the receivers.
func GetEnvironmentVariables(c *v1beta1.Config, logger logr.Logger) ([]corev1.EnvVar, error) {
	return getEnvironmentVariablesForComponentKinds(c, logger, KindReceiver)
}

// GetAllRbacRules returns all RBAC rules.
func GetAllRbacRules(c *v1beta1.Config, logger logr.Logger) ([]rbacv1.PolicyRule, error) {
	return getRbacRulesForComponentKinds(c, logger, KindReceiver, KindExporter, KindProcessor, KindExtension)
}

// ApplyDefaults applies default configuration values to the collector config.
func ApplyDefaults(c *v1beta1.Config, logger logr.Logger, opts ...components.DefaultOption) ([]EventInfo, error) {
	return applyDefaultForComponentKinds(c, logger, opts, KindReceiver, KindExporter, KindExtension)
}

// GetLivenessProbe gets the first enabled liveness probe.
func GetLivenessProbe(c *v1beta1.Config, logger logr.Logger) (*corev1.Probe, error) {
	if c.Extensions == nil {
		return nil, nil
	}

	enabledComponents := GetEnabledComponents(c)
	for componentName := range enabledComponents[KindExtension] {
		parser := extensions.ParserFor(componentName)
		if probe, err := parser.GetLivenessProbe(logger, c.Extensions.Object[componentName]); err != nil {
			return nil, err
		} else if probe != nil {
			return probe, nil
		}
	}
	return nil, nil
}

// GetReadinessProbe gets the first enabled readiness probe.
func GetReadinessProbe(c *v1beta1.Config, logger logr.Logger) (*corev1.Probe, error) {
	if c.Extensions == nil {
		return nil, nil
	}

	enabledComponents := GetEnabledComponents(c)
	for componentName := range enabledComponents[KindExtension] {
		parser := extensions.ParserFor(componentName)
		if probe, err := parser.GetReadinessProbe(logger, c.Extensions.Object[componentName]); err != nil {
			return nil, err
		} else if probe != nil {
			return probe, nil
		}
	}
	return nil, nil
}

// GetStartupProbe gets the first enabled startup probe.
func GetStartupProbe(c *v1beta1.Config, logger logr.Logger) (*corev1.Probe, error) {
	if c.Extensions == nil {
		return nil, nil
	}

	enabledComponents := GetEnabledComponents(c)
	for componentName := range enabledComponents[KindExtension] {
		parser := extensions.ParserFor(componentName)
		if probe, err := parser.GetStartupProbe(logger, c.Extensions.Object[componentName]); err != nil {
			return nil, err
		} else if probe != nil {
			return probe, nil
		}
	}
	return nil, nil
}

// Yaml encodes the config and returns it as a YAML string.
func Yaml(c *v1beta1.Config) (string, error) {
	var buf bytes.Buffer
	yamlEncoder := go_yaml.NewEncoder(&buf, go_yaml.IndentSequence(true), go_yaml.AutoInt())
	if err := yamlEncoder.Encode(c); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// NullObjects returns null objects in the config.
func NullObjects(c *v1beta1.Config) []string {
	var nullKeys []string
	if nulls := getNullValuedKeys(c.Receivers.Object); len(nulls) > 0 {
		nullKeys = append(nullKeys, addPrefix("receivers.", nulls)...)
	}
	if nulls := getNullValuedKeys(c.Exporters.Object); len(nulls) > 0 {
		nullKeys = append(nullKeys, addPrefix("exporters.", nulls)...)
	}
	if c.Processors != nil {
		if nulls := getNullValuedKeys(c.Processors.Object); len(nulls) > 0 {
			nullKeys = append(nullKeys, addPrefix("processors.", nulls)...)
		}
	}
	if c.Extensions != nil {
		if nulls := getNullValuedKeys(c.Extensions.Object); len(nulls) > 0 {
			nullKeys = append(nullKeys, addPrefix("extensions.", nulls)...)
		}
	}
	if c.Connectors != nil {
		if nulls := getNullValuedKeys(c.Connectors.Object); len(nulls) > 0 {
			nullKeys = append(nullKeys, addPrefix("connectors.", nulls)...)
		}
	}
	// Make the return deterministic. The config uses maps therefore processing order is non-deterministic.
	slices.Sort(nullKeys)
	return nullKeys
}

// MetricsEndpoint attempts to get the host and port number from the service telemetry config.
func MetricsEndpoint(s *v1beta1.Service, logger logr.Logger) (host string, port int32, err error) {
	telemetry := GetTelemetry(s, logger)
	if telemetry == nil {
		return defaultServiceHost, defaultServicePort, nil
	}

	if telemetry.Metrics.Address != "" && len(telemetry.Metrics.Readers) == 0 {
		host, port, err := parseAddressEndpoint(telemetry.Metrics.Address)
		if err != nil {
			return "", 0, err
		}

		return host, port, nil
	}

	for _, r := range telemetry.Metrics.Readers {
		if r.Pull == nil {
			continue
		}
		prom := r.Pull.Exporter.Prometheus
		if prom == nil {
			continue
		}
		host := defaultServiceHost
		if prom.Host != nil && *prom.Host != "" {
			host = *prom.Host
		}
		port := defaultServicePort
		if prom.Port != nil && *prom.Port != 0 {
			if *prom.Port < 0 || *prom.Port > 65535 {
				return "", 0, fmt.Errorf("invalid prometheus metrics port: %d", *prom.Port)
			}
			port = int32(*prom.Port)
		}
		return host, port, nil
	}

	return defaultServiceHost, defaultServicePort, nil
}

// ServiceApplyDefaults inserts configuration defaults if not set.
func ServiceApplyDefaults(s *v1beta1.Service, logger logr.Logger) ([]EventInfo, error) {
	var events []EventInfo
	tel := GetTelemetry(s, logger)

	if tel == nil {
		logger.V(2).Info("no telemetry configuration parsed, creating default")
		tel = &v1beta1.Telemetry{}
		s.Telemetry = &v1beta1.AnyConfig{
			Object: map[string]any{},
		}
	}

	if tel.Metrics.Address != "" || len(tel.Metrics.Readers) != 0 {
		logger.V(1).Info("telemetry configuration already provided by user, skipping defaults",
			"metricsAddress", tel.Metrics.Address,
			"readersCount", len(tel.Metrics.Readers))
		return events, nil
	}

	logger.V(2).Info("no telemetry readers configuration found, applying default Prometheus endpoint")

	host, port, err := MetricsEndpoint(s, logger)
	if err != nil {
		logger.Error(err, "failed to determine metrics endpoint for default configuration")
		return events, err
	}

	reader := AddPrometheusMetricsEndpoint(host, port)
	tel.Metrics.Readers = append(tel.Metrics.Readers, reader)

	events = append(events, EventInfo{
		Type:    corev1.EventTypeNormal,
		Reason:  "Spec.Service.Telemetry.DefaultsApplied",
		Message: fmt.Sprintf("Applied default Prometheus telemetry configuration (host: %s, port: %d)", host, port),
	})

	telConfig, err := ToAnyConfig(tel)
	if err != nil {
		return events, err
	}

	if err := mergo.Merge(&s.Telemetry.Object, telConfig.Object); err != nil {
		return events, err
	}
	return events, nil
}

// GetTelemetry extracts the telemetry settings from the service config.
func GetTelemetry(s *v1beta1.Service, logger logr.Logger) *v1beta1.Telemetry {
	if s.Telemetry == nil {
		logger.V(2).Info("no spec.service.telemetry configuration found")
		return nil
	}

	// Convert map to JSON bytes
	jsonData, err := json.Marshal(s.Telemetry)
	if err != nil {
		logger.Error(err, "failed to marshal telemetry configuration to JSON", "telemetry", s.Telemetry.Object)
		return nil
	}

	logger.V(2).Info("marshaled telemetry configuration", "json", string(jsonData))

	t := &v1beta1.Telemetry{}
	// Unmarshal JSON into the provided struct
	if err := json.Unmarshal(jsonData, t); err != nil {
		logger.Error(err, "failed to unmarshal telemetry configuration, this may indicate invalid configuration", "json", string(jsonData), "originalConfig", s.Telemetry.Object)
		return nil
	}

	logger.V(2).Info("successfully parsed telemetry configuration",
		"metricsLevel", t.Metrics.Level,
		"metricsAddress", t.Metrics.Address,
		"readersCount", len(t.Metrics.Readers))

	return t
}

// ToAnyConfig converts the Telemetry struct to an AnyConfig struct.
func ToAnyConfig(t *v1beta1.Telemetry) (*v1beta1.AnyConfig, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	normalizeConfig(result)

	return &v1beta1.AnyConfig{
		Object: result,
	}, nil
}

// AddPrometheusMetricsEndpoint creates a MetricReader for a Prometheus pull endpoint.
func AddPrometheusMetricsEndpoint(host string, port int32) otelConfig.MetricReader {
	portInt := int(port)
	return otelConfig.MetricReader{
		Pull: &otelConfig.PullMetricReader{
			Exporter: otelConfig.PullMetricExporter{
				Prometheus: &otelConfig.Prometheus{
					Host: &host,
					Port: &portInt,
				},
			},
		},
	}
}

