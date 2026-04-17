// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apihelpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"dario.cat/mergo"
	"github.com/go-logr/logr"
	go_yaml "github.com/goccy/go-yaml"
	otelConfig "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

const (
	defaultServicePort int32 = 8888
	defaultServiceHost       = "0.0.0.0"
)

// MetricsConfig holds the telemetry metrics settings parsed from the collector config.
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
	Metrics MetricsConfig `json:"metrics,omitempty" yaml:"metrics,omitempty"`

	// Resource specifies user-defined attributes to include with all emitted telemetry.
	// Note that some attributes are added automatically (e.g. service.version) even
	// if they are not specified here. In order to suppress such attributes the
	// attribute must be specified in this map with null YAML value (nil string pointer).
	Resource map[string]*string `json:"resource,omitempty" yaml:"resource,omitempty"`
}

// ToAnyConfig converts the Telemetry struct to an AnyConfig struct.
func (t *Telemetry) ToAnyConfig() (*v1beta1.AnyConfig, error) {
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

// normalizeConfig fixes the config to be valid for the collector.
// It removes nil values, converts float64 port values to int32.
func normalizeConfig(m map[string]any) {
	for k, v := range m {
		switch val := v.(type) {
		case nil:
			delete(m, k)
		case map[string]any:
			normalizeConfig(val)
		case []any:
			for i, item := range val {
				if item == nil {
					val[i] = map[string]any{}
				} else if sub, ok := item.(map[string]any); ok {
					normalizeConfig(sub)
				}
			}
		case float64:
			if k == "port" {
				m[k] = int32(val)
			}
		default:
		}
	}
}

// ConfigYAML encodes the Config and returns it as a YAML string.
func ConfigYAML(c *v1beta1.Config) (string, error) {
	var buf bytes.Buffer
	yamlEncoder := go_yaml.NewEncoder(&buf, go_yaml.IndentSequence(true), go_yaml.AutoInt())
	if err := yamlEncoder.Encode(c); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// GetServiceTelemetry parses the service telemetry configuration.
func GetServiceTelemetry(s *v1beta1.Service, logger logr.Logger) *Telemetry {
	if s.Telemetry == nil {
		logger.V(2).Info("no spec.service.telemetry configuration found")
		return nil
	}

	jsonData, err := json.Marshal(s.Telemetry)
	if err != nil {
		logger.Error(err, "failed to marshal telemetry configuration to JSON", "telemetry", s.Telemetry.Object)
		return nil
	}

	logger.V(2).Info("marshaled telemetry configuration", "json", string(jsonData))

	t := &Telemetry{}
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

// ServiceMetricsEndpoint attempts to get the host and port number from the service telemetry config.
// It works even before env var expansion happens, when a simple net.SplitHostPort would fail because
// of the extra colon from the env var, i.e. the address looks like "${env:POD_IP}:4317".
// In cases where the port itself is a variable, i.e. "${env:POD_IP}:${env:PORT}", this returns an error.
func ServiceMetricsEndpoint(s *v1beta1.Service, logger logr.Logger) (host string, port int32, err error) {
	telemetry := GetServiceTelemetry(s, logger)
	if telemetry == nil {
		return defaultServiceHost, defaultServicePort, nil
	}

	if telemetry.Metrics.Address != "" && len(telemetry.Metrics.Readers) == 0 {
		host, port, err := parseAddr(telemetry.Metrics.Address)
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

// applyServiceDefaults inserts telemetry configuration defaults for the service if not already set.
// Returns a list of events that should be recorded by the caller.
func applyServiceDefaults(s *v1beta1.Service, logger logr.Logger) ([]v1beta1.EventInfo, error) {
	var events []v1beta1.EventInfo
	tel := GetServiceTelemetry(s, logger)

	if tel == nil {
		logger.V(2).Info("no telemetry configuration parsed, creating default")
		tel = &Telemetry{}
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

	host, port, err := ServiceMetricsEndpoint(s, logger)
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

	telConfig, err := tel.ToAnyConfig()
	if err != nil {
		return events, err
	}

	if err := mergo.Merge(&s.Telemetry.Object, telConfig.Object); err != nil {
		return events, err
	}
	return events, nil
}

// AddPrometheusMetricsEndpoint builds an otelconf MetricReader for a Prometheus pull endpoint.
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

// parseAddr parses an address string and returns the host and port.
// It works even before env var expansion happens.
func parseAddr(address string) (host string, port int32, err error) {
	const portEnvVarRegex = `:\${[env:]?.*}$`
	if regexp.MustCompile(portEnvVarRegex).MatchString(address) {
		return "", 0, fmt.Errorf("couldn't determine metrics port from configuration: %s", address)
	}

	const explicitPortRegex = `:(\d+$)`
	explicitPortMatches := regexp.MustCompile(explicitPortRegex).FindStringSubmatch(address)
	if len(explicitPortMatches) <= 1 {
		return address, defaultServicePort, nil
	}

	p, err := strconv.ParseInt(explicitPortMatches[1], 10, 32)
	if err != nil {
		return "", 0, fmt.Errorf("couldn't determine metrics port from configuration: %s", address)
	}
	port = intToInt32Safe(int(p))
	host, _, _ = strings.Cut(address, explicitPortMatches[0])
	return host, port, nil
}

func intToInt32Safe(v int) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}

// GetEnabledComponents constructs a map of enabled components by component kind.
func GetEnabledComponents(c *v1beta1.Config) map[v1beta1.ComponentKind]map[string]any {
	toReturn := map[v1beta1.ComponentKind]map[string]any{
		v1beta1.KindReceiver:  {},
		v1beta1.KindProcessor: {},
		v1beta1.KindExporter:  {},
		v1beta1.KindExtension: {},
	}
	for _, pipeline := range c.Service.Pipelines {
		if pipeline == nil {
			continue
		}
		for _, id := range pipeline.Receivers {
			toReturn[v1beta1.KindReceiver][id] = struct{}{}
		}
		for _, id := range pipeline.Exporters {
			toReturn[v1beta1.KindExporter][id] = struct{}{}
		}
		for _, id := range pipeline.Processors {
			toReturn[v1beta1.KindProcessor][id] = struct{}{}
		}
	}
	for _, id := range c.Service.Extensions {
		toReturn[v1beta1.KindExtension][id] = struct{}{}
	}
	return toReturn
}

// NullObjects returns keys with null values in the config, useful for validating collector configs.
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
	slices.Sort(nullKeys)
	return nullKeys
}

// addPrefix adds a prefix to each element of the array.
func addPrefix(prefix string, arr []string) []string {
	if len(arr) == 0 {
		return []string{}
	}
	var prefixed []string
	for _, v := range arr {
		prefixed = append(prefixed, fmt.Sprintf("%s%s", prefix, v))
	}
	return prefixed
}

// getNullValuedKeys returns keys from the input map whose values are nil, using dot notation for nested keys.
func getNullValuedKeys(cfg map[string]any) []string {
	var nullKeys []string
	for k, v := range cfg {
		if v == nil {
			nullKeys = append(nullKeys, fmt.Sprintf("%s:", k))
		}
		if reflect.ValueOf(v).Kind() == reflect.Map {
			if val, ok := v.(map[string]any); ok {
				if nulls := getNullValuedKeys(val); len(nulls) > 0 {
					nullKeys = append(nullKeys, addPrefix(k+".", nulls)...)
				}
			}
		}
	}
	return nullKeys
}
