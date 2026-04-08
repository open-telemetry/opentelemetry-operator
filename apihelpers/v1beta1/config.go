// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"

	"dario.cat/mergo"
	"github.com/go-logr/logr"
	go_yaml "github.com/goccy/go-yaml"
	otelConfig "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/components"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/exporters"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/extensions"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/processors"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/receivers"
)

// GetConfigEnabledComponents constructs a list of enabled components by component type.
func GetConfigEnabledComponents(c apis.Config) map[apis.ComponentKind]map[string]any {
	toReturn := map[apis.ComponentKind]map[string]any{
		apis.KindReceiver:  {},
		apis.KindProcessor: {},
		apis.KindExporter:  {},
		apis.KindExtension: {},
	}
	for _, extension := range c.Service.Extensions {
		toReturn[apis.KindExtension][extension] = struct{}{}
	}

	for _, pipeline := range c.Service.Pipelines {
		if pipeline == nil {
			continue
		}
		for _, componentId := range pipeline.Receivers {
			toReturn[apis.KindReceiver][componentId] = struct{}{}
		}
		for _, componentId := range pipeline.Exporters {
			toReturn[apis.KindExporter][componentId] = struct{}{}
		}
		for _, componentId := range pipeline.Processors {
			toReturn[apis.KindProcessor][componentId] = struct{}{}
		}
	}
	for _, componentId := range c.Service.Extensions {
		toReturn[apis.KindExtension][componentId] = struct{}{}
	}
	return toReturn
}

// getConfigRbacRulesForComponentKinds gets the RBAC Rules for the given ComponentKind(s).
func getConfigRbacRulesForComponentKinds(c apis.Config, logger logr.Logger, componentKinds ...apis.ComponentKind) ([]rbacv1.PolicyRule, error) {
	var rules []rbacv1.PolicyRule
	enabledComponents := GetConfigEnabledComponents(c)
	for _, componentKind := range componentKinds {
		var retriever components.ParserRetriever
		var cfg apis.AnyConfig
		switch componentKind {
		case apis.KindReceiver:
			retriever = receivers.ReceiverFor
			cfg = c.Receivers
		case apis.KindExporter:
			retriever = exporters.ParserFor
			cfg = c.Exporters
		case apis.KindProcessor:
			retriever = processors.ProcessorFor
			if c.Processors == nil {
				cfg = apis.AnyConfig{}
			} else {
				cfg = *c.Processors
			}
		case apis.KindExtension:
			retriever = extensions.ParserFor
			if c.Extensions == nil {
				cfg = apis.AnyConfig{}
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

// getConfigPortsForComponentKinds gets the ports for the given ComponentKind(s).
func getConfigPortsForComponentKinds(c apis.Config, logger logr.Logger, componentKinds ...ComponentKind) ([]corev1.ServicePort, error) {
	var ports []corev1.ServicePort
	enabledComponents := GetConfigEnabledComponents(c)
	for _, componentKind := range componentKinds {
		var retriever components.ParserRetriever
		var cfg apis.AnyConfig
		switch componentKind {
		case apis.KindReceiver:
			retriever = receivers.ReceiverFor
			cfg = c.Receivers
		case apis.KindExporter:
			retriever = exporters.ParserFor
			cfg = c.Exporters
		case apis.KindProcessor:
			continue
		case apis.KindExtension:
			retriever = extensions.ParserFor
			if c.Extensions == nil {
				cfg = apis.AnyConfig{}
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

// getConfigEnvironmentVariablesForComponentKinds gets the environment variables for the given ComponentKind(s).
func getConfigEnvironmentVariablesForComponentKinds(c apis.Config, logger logr.Logger, componentKinds ...ComponentKind) ([]corev1.EnvVar, error) {
	envVars := []corev1.EnvVar{}
	enabledComponents := GetConfigEnabledComponents(c)
	for _, componentKind := range componentKinds {
		var retriever components.ParserRetriever
		var apis.cfg AnyConfig

		switch componentKind {
		case apis.KindReceiver:
			retriever = receivers.ReceiverFor
			cfg = c.Receivers
		case apis.KindExporter, apis.KindProcessor, apis.KindExtension:
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

// applyConfigDefaultForComponentKinds applies defaults to the endpoints for the given ComponentKind(s).
// If defaultsCfg.TLSProfile is set, TLS defaults are also applied via the Parser.GetDefaultConfig method.
// Returns a list of events that should be recorded by the caller.
func applyConfigDefaultForComponentKinds(c apis.Config, logger logr.Logger, parserOpts []components.DefaultOption, componentKinds ...ComponentKind) ([]EventInfo, error) {
	events, err := c.Service.ApplyDefaultConfig(logger)
	if err != nil {
		return events, err
	}
	enabledComponents := GetConfigEnabledComponents(c)
	for _, componentKind := range componentKinds {
		var retriever components.ParserRetriever
		var cfg apis.AnyConfig
		switch componentKind {
		case apis.KindReceiver:
			retriever = receivers.ReceiverFor
			cfg = c.Receivers
		case apis.KindExporter, apis.KindProcessor:
			retriever = exporters.ParserFor
			cfg = c.Exporters
		case apis.KindExtension:
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

func GetConfigReceiverPorts(c apis.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getConfigPortsForComponentKinds(c, logger, apis.KindReceiver)
}

func GetConfigExporterPorts(c apis.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getConfigPortsForComponentKinds(c, logger, apis.KindExporter)
}

func GetConfigExtensionPorts(c apis.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getConfigPortsForComponentKinds(c, logger, apis.KindExtension)
}

func GetConfigReceiverAndExporterPorts(c apis.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getConfigPortsForComponentKinds(c, logger, apis.KindReceiver, apis.KindExporter)
}

func GetConfigAllPorts(c apis.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getConfigPortsForComponentKinds(c, logger, apis.KindReceiver, apis.KindExporter, apis.KindExtension)
}

func GetConfigEnvironmentVariables(c apis.Config, logger logr.Logger) ([]corev1.EnvVar, error) {
	return getConfigEnvironmentVariablesForComponentKinds(c, logger, apis.KindReceiver)
}

func GetConfigAllRbacRules(c apis.Config, logger logr.Logger) ([]rbacv1.PolicyRule, error) {
	return getConfigRbacRulesForComponentKinds(c, logger, apis.KindReceiver, apis.KindExporter, apis.KindProcessor, apis.KindExtension)
}

// ApplyDefaultConfig applies default configuration values to the collector config.
// Optional DefaultsOption arguments can be provided to customize behavior.
func ApplyDefaultConfig(c apis.Config, logger logr.Logger, opts ...components.DefaultOption) ([]apis.EventInfo, error) {
	return applyConfigDefaultForComponentKinds(c, logger, opts, apis.KindReceiver, apis.KindExporter, apis.KindExtension)
}

// GetConfigLivenessProbe gets the first enabled liveness probe. There should only ever be one extension enabled
// that provides the hinting for the liveness probe.
func GetConfigLivenessProbe(c apis.Config, logger logr.Logger) (*corev1.Probe, error) {
	if c.Extensions == nil {
		return nil, nil
	}

	enabledComponents := c.GetConfigEnabledComponents()
	for componentName := range enabledComponents[KindExtension] {
		// TODO: Clean up the naming here and make it simpler to use a retriever.
		parser := extensions.ParserFor(componentName)
		if probe, err := parser.GetConfigLivenessProbe(logger, c.Extensions.Object[componentName]); err != nil {
			return nil, err
		} else if probe != nil {
			return probe, nil
		}
	}
	return nil, nil
}

// GetConfigReadinessProbe gets the first enabled readiness probe. There should only ever be one extension enabled
// that provides the hinting for the readiness probe.
func GetConfigReadinessProbe(c apis.Config, logger logr.Logger) (*corev1.Probe, error) {
	if c.Extensions == nil {
		return nil, nil
	}

	enabledComponents := c.GetConfigEnabledComponents()
	for componentName := range enabledComponents[KindExtension] {
		// TODO: Clean up the naming here and make it simpler to use a retriever.
		parser := extensions.ParserFor(componentName)
		if probe, err := parser.GetConfigReadinessProbe(logger, c.Extensions.Object[componentName]); err != nil {
			return nil, err
		} else if probe != nil {
			return probe, nil
		}
	}
	return nil, nil
}

// GetConfigStartupProbe gets the first enabled startup probe. There should only ever be one extension enabled
// that provides the hinting for the startup probe.
func GetConfigStartupProbe(c apis.Config, logger logr.Logger) (*corev1.Probe, error) {
	if c.Extensions == nil {
		return nil, nil
	}

	enabledComponents := GetConfigEnabledComponents(c)
	for componentName := range enabledComponents[apis.KindExtension] {
		// TODO: Clean up the naming here and make it simpler to use a retriever.
		parser := extensions.ParserFor(componentName)
		if probe, err := parser.GetConfigStartupProbe(logger, c.Extensions.Object[componentName]); err != nil {
			return nil, err
		} else if probe != nil {
			return probe, nil
		}
	}
	return nil, nil
}

// Yaml encodes the current object and returns it as a string.
func Yaml(c apis.Config) (string, error) {
	var buf bytes.Buffer
	yamlEncoder := go_yaml.NewEncoder(&buf, go_yaml.IndentSequence(true), go_yaml.AutoInt())
	if err := yamlEncoder.Encode(c); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// NullObjects returns null objects in the config.
func NullObjects(c apis.Config) []string {
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

const (
	defaultServicePort int32 = 8888
	defaultServiceHost       = "0.0.0.0"
)

// MetricsEndpoint attempts gets the host and port number from the host address without doing any validation regarding the
// address itself.
// It works even before env var expansion happens, when a simple `net.SplitHostPort` would fail because of the extra colon
// from the env var, i.e. the address looks like "${env:POD_IP}:4317", "${env:POD_IP}", or "${POD_IP}".
// In cases which the port itself is a variable, i.e. "${env:POD_IP}:${env:PORT}", this returns an error. This happens
// because the port is used to generate Service objects and mappings.
func MetricsEndpoint(s apis.Service, logger logr.Logger) (host string, port int32, err error) {
	telemetry := GetServiceTelemetry(s, logger)
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
			if *prom.Port < 0 || *prom.Port > math.MaxUint16 {
				return "", 0, fmt.Errorf("invalid prometheus metrics port: %d", *prom.Port)
			}
			port = int32(*prom.Port)
		}
		return host, port, nil
	}

	return defaultServiceHost, defaultServicePort, nil
}

// ApplyDefaultService inserts configuration defaults if it has not been set.
// Returns a list of events that should be recorded by the caller.
func ApplyDefaultService(s apis.Service, logger logr.Logger) ([]apis.EventInfo, error) {
	var events []apis.EventInfo
	tel := GetServiceTelemetry(s, logger)

	if tel == nil {
		logger.V(2).Info("no telemetry configuration parsed, creating default")
		tel = &apis.Telemetry{}
		s.Telemetry = &apis.AnyConfig{
			Object: map[string]any{},
		}
	}

	if tel.Metrics.Address != "" || len(tel.Metrics.Readers) != 0 {
		// The user already set the address or the readers, so we don't need to do anything
		logger.V(1).Info("telemetry configuration already provided by user, skipping defaults",
			"metricsAddress", tel.Metrics.Address,
			"readersCount", len(tel.Metrics.Readers))
		return events, nil
	}

	logger.V(2).Info("no telemetry readers configuration found, applying default Prometheus endpoint")

	host, port, err := s.MetricsEndpoint(logger)
	if err != nil {
		logger.Error(err, "failed to determine metrics endpoint for default configuration")
		return events, err
	}

	reader := AddPrometheusMetricsEndpoint(host, port)
	tel.Metrics.Readers = append(tel.Metrics.Readers, reader)

	events = append(events, apis.EventInfo{
		Type:    corev1.EventTypeNormal,
		Reason:  "Spec.Service.Telemetry.DefaultsApplied",
		Message: fmt.Sprintf("Applied default Prometheus telemetry configuration (host: %s, port: %d)", host, port),
	})

	telConfig, err := TelemetryToAnyConfig(tel)
	if err != nil {
		return events, err
	}

	if err := mergo.Merge(&s.Telemetry.Object, telConfig.Object); err != nil {
		return events, err
	}
	return events, nil
}

// TelemetryToAnyConfig converts the Telemetry struct to an AnyConfig struct.
func TelemetryToAnyConfig(t apis.Telemetry) (apis.AnyConfig, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	normalizeConfig(result)

	return &apis.AnyConfig{
		Object: result,
	}, nil
}

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

// GetServiceTelemetry serves as a helper function to access the fields we care about in the underlying telemetry struct.
// This exists to avoid needing to worry extra fields in the telemetry struct.
func GetServiceTelemetry(s apis.Service, logger logr.Logger) *apis.Telemetry {
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

	t := &apis.Telemetry{}
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
