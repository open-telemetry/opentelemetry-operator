// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconfig

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"

	"dario.cat/mergo"
	"github.com/go-logr/logr"
	otelConfig "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/components"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/exporters"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/extensions"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/processors"
	"github.com/open-telemetry/opentelemetry-operator/internal/components/receivers"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

// MetricsConfig comes from the collector.
type MetricsConfig struct {
	// Level is the level of telemetry metrics, the possible values are:
	//  - "none" indicates that no telemetry data should be collected;
	//  - "basic" is the recommended and covers the basics of the service telemetry.
	//  - "normal" adds some other indicators on top of basic.
	//  - "detailed" adds dimensions and views to the previous levels.
	Level string `json:"level,omitempty" yaml:"level,omitempty"`

	// Address is the [address]:port that metrics exposition should be bound to.
	Address string `json:"address,omitempty" yaml:"address,omitempty"`

	otelConfig.MeterProvider `mapstructure:",squash"`
}

func (in *MetricsConfig) DeepCopyInto(out *MetricsConfig) {
	*out = *in
	out.MeterProvider = in.MeterProvider
}

// DeepCopy creates a new deepcopy of MetricsConfig.
func (in *MetricsConfig) DeepCopy() *MetricsConfig {
	if in == nil {
		return nil
	}
	out := new(MetricsConfig)
	in.DeepCopyInto(out)
	return out
}

// Telemetry is an intermediary type that allows for easy access to the collector's telemetry settings.
type Telemetry struct {
	Metrics MetricsConfig `json:"metrics,omitzero" yaml:"metrics,omitempty"`

	// Resource specifies user-defined attributes to include with all emitted telemetry.
	// Note that some attributes are added automatically (e.g. service.version) even
	// if they are not specified here. In order to suppress such attributes the
	// attribute must be specified in this map with null YAML value (nil string pointer).
	Resource map[string]*string `json:"resource,omitempty" yaml:"resource,omitempty"`
}

// GetEnabledComponents constructs a list of enabled components by component type.
func GetEnabledComponents(c *v1beta1.Config) map[v1beta1.ComponentKind]map[string]any {
	toReturn := map[v1beta1.ComponentKind]map[string]any{
		v1beta1.KindReceiver:  {},
		v1beta1.KindProcessor: {},
		v1beta1.KindExporter:  {},
		v1beta1.KindExtension: {},
	}
	for _, extension := range c.Service.Extensions {
		toReturn[v1beta1.KindExtension][extension] = struct{}{}
	}

	for _, pipeline := range c.Service.Pipelines {
		if pipeline == nil {
			continue
		}
		for _, componentId := range pipeline.Receivers {
			toReturn[v1beta1.KindReceiver][componentId] = struct{}{}
		}
		for _, componentId := range pipeline.Exporters {
			toReturn[v1beta1.KindExporter][componentId] = struct{}{}
		}
		for _, componentId := range pipeline.Processors {
			toReturn[v1beta1.KindProcessor][componentId] = struct{}{}
		}
	}
	for _, componentId := range c.Service.Extensions {
		toReturn[v1beta1.KindExtension][componentId] = struct{}{}
	}
	return toReturn
}

// getRbacRulesForComponentKinds gets the RBAC Rules for the given ComponentKind(s).
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

// applyDefaultForComponentKinds applies defaults to the endpoints for the given ComponentKind(s).
// If defaultsCfg.TLSProfile is set, TLS defaults are also applied via the Parser.GetDefaultConfig method.
// Returns a list of events that should be recorded by the caller.
func applyDefaultForComponentKinds(c *v1beta1.Config, logger logr.Logger, parserOpts []components.DefaultOption, componentKinds ...v1beta1.ComponentKind) ([]v1beta1.EventInfo, error) {
	events, err := ServiceApplyDefaults(&c.Service, logger)
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

// GetReceiverPorts gets the ports for receivers.
func GetReceiverPorts(c *v1beta1.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getPortsForComponentKinds(c, logger, v1beta1.KindReceiver)
}

// GetExporterPorts gets the ports for exporters.
func GetExporterPorts(c *v1beta1.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getPortsForComponentKinds(c, logger, v1beta1.KindExporter)
}

// GetExtensionPorts gets the ports for extensions.
func GetExtensionPorts(c *v1beta1.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getPortsForComponentKinds(c, logger, v1beta1.KindExtension)
}

// GetReceiverAndExporterPorts gets the ports for receivers and exporters.
func GetReceiverAndExporterPorts(c *v1beta1.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getPortsForComponentKinds(c, logger, v1beta1.KindReceiver, v1beta1.KindExporter)
}

// GetAllPorts gets the ports for all component kinds that expose ports.
func GetAllPorts(c *v1beta1.Config, logger logr.Logger) ([]corev1.ServicePort, error) {
	return getPortsForComponentKinds(c, logger, v1beta1.KindReceiver, v1beta1.KindExporter, v1beta1.KindExtension)
}

// GetEnvironmentVariables gets the environment variables for receivers.
func GetEnvironmentVariables(c *v1beta1.Config, logger logr.Logger) ([]corev1.EnvVar, error) {
	return getEnvironmentVariablesForComponentKinds(c, logger, v1beta1.KindReceiver)
}

// GetAllRbacRules gets the RBAC rules for all component kinds.
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
		parser := extensions.ParserFor(componentName)
		if probe, err := parser.GetStartupProbe(logger, c.Extensions.Object[componentName]); err != nil {
			return nil, err
		} else if probe != nil {
			return probe, nil
		}
	}
	return nil, nil
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

// MetricsEndpoint attempts gets the host and port number from the host address without doing any validation regarding the
// address itself.
// It works even before env var expansion happens, when a simple `net.SplitHostPort` would fail because of the extra colon
// from the env var, i.e. the address looks like "${env:POD_IP}:4317", "${env:POD_IP}", or "${POD_IP}".
// In cases which the port itself is a variable, i.e. "${env:POD_IP}:${env:PORT}", this returns an error. This happens
// because the port is used to generate Service objects and mappings.
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
			if *prom.Port < 0 || *prom.Port > math.MaxUint16 {
				return "", 0, fmt.Errorf("invalid prometheus metrics port: %d", *prom.Port)
			}
			port = int32(*prom.Port)
		}
		return host, port, nil
	}

	return defaultServiceHost, defaultServicePort, nil
}

// ServiceApplyDefaults inserts configuration defaults if it has not been set.
// Returns a list of events that should be recorded by the caller.
func ServiceApplyDefaults(s *v1beta1.Service, logger logr.Logger) ([]v1beta1.EventInfo, error) {
	var events []v1beta1.EventInfo
	tel := GetTelemetry(s, logger)

	if tel == nil {
		logger.V(2).Info("no telemetry configuration parsed, creating default")
		tel = &Telemetry{}
		s.Telemetry = &v1beta1.AnyConfig{
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

	host, port, err := MetricsEndpoint(s, logger)
	if err != nil {
		logger.Error(err, "failed to determine metrics endpoint for default configuration")
		return events, err
	}

	reader := AddPrometheusMetricsEndpoint(host, port)
	tel.Metrics.Readers = append(tel.Metrics.Readers, reader)

	events = append(events, v1beta1.EventInfo{
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

// AddPrometheusMetricsEndpoint creates a MetricReader with a Prometheus pull exporter.
// without_type_suffix/without_units/without_scope_info are explicitly set to false to
// preserve the historical metric name shape produced by operator-managed collectors
// before open-telemetry/opentelemetry-collector#15027. Opt into collector defaults via
// the operator.collector.usedefaulttelemetryshape feature gate. See issue #5075.
func AddPrometheusMetricsEndpoint(host string, port int32) otelConfig.MetricReader {
	portInt := int(port)
	prom := &otelConfig.Prometheus{
		Host: &host,
		Port: &portInt,
	}
	if !featuregate.UseCollectorDefaultTelemetryShape.IsEnabled() {
		falseVal := false
		prom.WithoutTypeSuffix = &falseVal
		prom.WithoutUnits = &falseVal
		prom.WithoutScopeInfo = &falseVal
	}
	return otelConfig.MetricReader{
		Pull: &otelConfig.PullMetricReader{
			Exporter: otelConfig.PullMetricExporter{
				Prometheus: prom,
			},
		},
	}
}

// GetTelemetry serves as a helper function to access the fields we care about in the underlying telemetry struct.
// This exists to avoid needing to worry extra fields in the telemetry struct.
func GetTelemetry(s *v1beta1.Service, logger logr.Logger) *Telemetry {
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

	t := &Telemetry{}
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

// TelemetryToAnyConfig converts the Telemetry struct to an AnyConfig struct.
func TelemetryToAnyConfig(t *Telemetry) (*v1beta1.AnyConfig, error) {
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
