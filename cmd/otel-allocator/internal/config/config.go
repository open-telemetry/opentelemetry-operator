// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-viper/mapstructure/v2"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	_ "github.com/prometheus/prometheus/discovery/install"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	sigsyaml "sigs.k8s.io/yaml"
)

const (
	DefaultListenAddr                                  = ":8080"
	DefaultHttpsListenAddr                             = ":8443"
	DefaultResyncTime                                  = 5 * time.Minute
	DefaultConfigFilePath               string         = "/conf/targetallocator.yaml"
	DefaultCRScrapeInterval             model.Duration = model.Duration(time.Second * 30)
	DefaultAllocationStrategy                          = "consistent-hashing"
	DefaultFilterStrategy                              = "relabel-config"
	DefaultCollectorNotReadyGracePeriod                = 30 * time.Second
)

var DefaultKubeConfigFilePath = filepath.Join(homedir.HomeDir(), ".kube", "config")

var defaultScrapeProtocolsCR = []monitoringv1.ScrapeProtocol{
	monitoringv1.OpenMetricsText1_0_0,
	monitoringv1.OpenMetricsText0_0_1,
	monitoringv1.PrometheusText1_0_0,
	monitoringv1.PrometheusText0_0_4,
}

// logger which discards all messages written to it. Replace this with slog.DiscardHandler after we require Go 1.24.
var NopLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.Level(math.MaxInt)}))

type Config struct {
	ListenAddr                   string                `yaml:"listen_addr,omitempty"`
	KubeConfigFilePath           string                `yaml:"kube_config_file_path,omitempty"`
	ClusterConfig                *rest.Config          `yaml:"-"`
	RootLogger                   logr.Logger           `yaml:"-"`
	CollectorSelector            *metav1.LabelSelector `yaml:"collector_selector,omitempty"`
	CollectorNamespace           string                `yaml:"collector_namespace,omitempty"`
	PromConfig                   *promconfig.Config    `yaml:"config"`
	AllocationStrategy           string                `yaml:"allocation_strategy,omitempty"`
	AllocationFallbackStrategy   string                `yaml:"allocation_fallback_strategy,omitempty"`
	FilterStrategy               string                `yaml:"filter_strategy,omitempty"`
	PrometheusCR                 PrometheusCRConfig    `yaml:"prometheus_cr,omitempty"`
	HTTPS                        HTTPSServerConfig     `yaml:"https,omitempty"`
	Telemetry                    TelemetryConfig       `yaml:"telemetry,omitempty"`
	CollectorNotReadyGracePeriod time.Duration         `yaml:"collector_not_ready_grace_period,omitempty"`
	AllowInsecureAuthSecrets     bool                  `yaml:"allow_insecure_auth_secrets,omitempty"`
}

type PrometheusCRConfig struct {
	Enabled                         bool                          `yaml:"enabled,omitempty"`
	AllowNamespaces                 []string                      `yaml:"allow_namespaces,omitempty"`
	DenyNamespaces                  []string                      `yaml:"deny_namespaces,omitempty"`
	SecretNamespaces                []string                      `yaml:"secret_namespaces,omitempty"`
	PodMonitorSelector              *metav1.LabelSelector         `yaml:"pod_monitor_selector,omitempty"`
	PodMonitorNamespaceSelector     *metav1.LabelSelector         `yaml:"pod_monitor_namespace_selector,omitempty"`
	ServiceMonitorSelector          *metav1.LabelSelector         `yaml:"service_monitor_selector,omitempty"`
	ServiceMonitorNamespaceSelector *metav1.LabelSelector         `yaml:"service_monitor_namespace_selector,omitempty"`
	ScrapeConfigSelector            *metav1.LabelSelector         `yaml:"scrape_config_selector,omitempty"`
	ScrapeConfigNamespaceSelector   *metav1.LabelSelector         `yaml:"scrape_config_namespace_selector,omitempty"`
	ProbeSelector                   *metav1.LabelSelector         `yaml:"probe_selector,omitempty"`
	ProbeNamespaceSelector          *metav1.LabelSelector         `yaml:"probe_namespace_selector,omitempty"`
	ScrapeInterval                  model.Duration                `yaml:"scrape_interval,omitempty"`
	EvaluationInterval              model.Duration                `yaml:"evaluation_interval,omitempty"`
	ScrapeProtocols                 []monitoringv1.ScrapeProtocol `yaml:"scrape_protocols,omitempty"`
	ScrapeClasses                   []monitoringv1.ScrapeClass    `yaml:"scrape_classes,omitempty"`
	// DenyFSAccessThroughSMs causes the Target Allocator to drop ServiceMonitor and
	// PodMonitor endpoints that reference arbitrary files on the file system. When
	// true, endpoints with bearerTokenFile, tlsConfig.caFile, tlsConfig.certFile, or
	// tlsConfig.keyFile referencing paths outside an operator-owned mount are
	// dropped from the produced scrape configuration while the remaining endpoints
	// are kept. This prevents tenants from stealing the Collector's service account
	// token. This is the equivalent of ArbitraryFSAccessThroughSMs.Deny from the
	// Prometheus Operator.
	// +optional
	DenyFSAccessThroughSMs bool `yaml:"deny_fs_access_through_sms,omitempty"`
}

type HTTPSServerConfig struct {
	Enabled         bool   `yaml:"enabled,omitempty"`
	ListenAddr      string `yaml:"listen_addr,omitempty"`
	CAFilePath      string `yaml:"ca_file_path,omitempty"`
	TLSCertFilePath string `yaml:"tls_cert_file_path,omitempty"`
	TLSKeyFilePath  string `yaml:"tls_key_file_path,omitempty"`
}

// TelemetryConfig holds the self-telemetry settings for the Target Allocator.
// The Metrics field follows the OTel declarative configuration spec (MeterProvider
// section), which means a telemetry YAML fragment can be validated with
// otelconf.ParseYAML for compatibility checking.
type TelemetryConfig struct {
	Metrics *MetricsConfig `yaml:"metrics,omitempty"`
}

// MetricsConfig mirrors otelconf's MeterProvider schema for the readers list.
type MetricsConfig struct {
	// Readers configures one or more metric readers, following the OTel
	// declarative configuration spec.
	Readers []MetricReader `yaml:"readers,omitempty"`
}

// MetricReader mirrors otelconf's MetricReader type.
type MetricReader struct {
	Periodic *PeriodicMetricReader `yaml:"periodic,omitempty"`
}

// PeriodicMetricReader mirrors otelconf's PeriodicMetricReader type.
// Interval and Timeout are in milliseconds, matching the otelconf spec.
type PeriodicMetricReader struct {
	// Interval is the delay between consecutive exports in milliseconds (default 60000).
	Interval int `yaml:"interval,omitempty"`
	// Timeout is the maximum allowed export duration in milliseconds (default 30000).
	Timeout int `yaml:"timeout,omitempty"`
	// Exporter configures the push exporter for this reader.
	Exporter MetricExporter `yaml:"exporter"`
}

// MetricExporter mirrors otelconf's PushMetricExporter type.
type MetricExporter struct {
	// OTLPGrpc configures an OTLP/gRPC metric exporter.
	OTLPGrpc *OTLPGrpcExporterConfig `yaml:"otlp_grpc,omitempty"`
	// OTLPHttp configures an OTLP/HTTP metric exporter.
	OTLPHttp *OTLPHttpExporterConfig `yaml:"otlp_http,omitempty"`
}

// OTLPGrpcExporterConfig mirrors otelconf's OTLPGrpcMetricExporter type.
type OTLPGrpcExporterConfig struct {
	// Endpoint is the gRPC receiver address. Accepts host:port or a full URL with scheme.
	Endpoint string `yaml:"endpoint,omitempty"`
	// Headers are additional key/value pairs sent with every export request.
	// Values support ${env:VAR} substitution.
	Headers []NameValuePair `yaml:"headers,omitempty"`
	// TemporalityPreference sets aggregation temporality: "cumulative" (default),
	// "delta", or "low_memory".
	TemporalityPreference string `yaml:"temporality_preference,omitempty"`
	// Tls configures TLS for the gRPC connection.
	Tls *GrpcTlsConfig `yaml:"tls,omitempty"`
}

// OTLPHttpExporterConfig mirrors otelconf's OTLPHttpMetricExporter type.
type OTLPHttpExporterConfig struct {
	// Endpoint is the OTLP/HTTP receiver base URL (e.g. "https://example.com:4318").
	// /v1/metrics is appended automatically unless already present.
	// Values support ${env:VAR} substitution.
	Endpoint string `yaml:"endpoint,omitempty"`
	// Headers are additional key/value pairs sent with every export request.
	// Values support ${env:VAR} substitution.
	Headers []NameValuePair `yaml:"headers,omitempty"`
	// TemporalityPreference sets aggregation temporality: "cumulative" (default),
	// "delta", or "low_memory".
	TemporalityPreference string `yaml:"temporality_preference,omitempty"`
}

// NameValuePair mirrors otelconf's NameStringValuePair type for HTTP/gRPC headers.
type NameValuePair struct {
	Name  string  `yaml:"name"`
	Value *string `yaml:"value"`
}

// GrpcTlsConfig mirrors otelconf's GrpcTls type.
type GrpcTlsConfig struct {
	// Insecure disables TLS — only suitable for local development.
	Insecure bool `yaml:"insecure,omitempty"`
}

// StringToModelOrTimeDurationHookFunc returns a DecodeHookFuncType
// that converts string to time.Duration, which can also be used
// as model.Duration.
func StringToModelOrTimeDurationHookFunc() mapstructure.DecodeHookFuncType {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any,
	) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}

		if t != reflect.TypeFor[model.Duration]() && t != reflect.TypeFor[time.Duration]() {
			return data, nil
		}

		return time.ParseDuration(data.(string))
	}
}

// MapToPromConfig returns a DecodeHookFuncType that provides a mechanism
// for decoding promconfig.Config involving its own unmarshal logic.
func MapToPromConfig() mapstructure.DecodeHookFuncType {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any,
	) (any, error) {
		if f.Kind() != reflect.Map {
			return data, nil
		}

		if t != reflect.TypeFor[*promconfig.Config]() {
			return data, nil
		}

		dataMap := data.(map[any]any)
		mb, err := yaml.Marshal(dataMap)
		if err != nil {
			return nil, err
		}

		pConfig := &promconfig.Config{}
		err = yaml.Unmarshal(mb, pConfig)
		if err != nil {
			return nil, err
		}
		return pConfig, nil
	}
}

const monitoringV1PkgPath = "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

// MapToMonitoringV1 handles prom-operator types that use json:",inline" for embedded structs.
// mapstructure with TagName:"yaml" can't squash these, so we round-trip through sigs.k8s.io/yaml
// which converts YAML → JSON → json.Unmarshal, correctly handling json:",inline".
func MapToMonitoringV1() mapstructure.DecodeHookFuncType {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any,
	) (any, error) {
		if f.Kind() != reflect.Map {
			return data, nil
		}
		target := t
		if t.Kind() == reflect.Pointer {
			target = t.Elem()
		}
		if target.PkgPath() != monitoringV1PkgPath {
			return data, nil
		}
		yamlBytes, err := yaml.Marshal(data)
		if err != nil {
			return data, err
		}
		result := reflect.New(target)
		if err := sigsyaml.Unmarshal(yamlBytes, result.Interface()); err != nil {
			return data, err
		}
		if t.Kind() == reflect.Pointer {
			return result.Interface(), nil
		}
		return result.Elem().Interface(), nil
	}
}

// MapToLabelSelector returns a DecodeHookFuncType that
// provides a mechanism for decoding both matchLabels and matchExpressions from camelcase to lowercase
// because we use yaml unmarshaling that supports lowercase field names if no `yaml` tag is defined
// and metav1.LabelSelector uses `json` tags.
// If both the camelcase and lowercase version is present, then the camelcase version takes precedence.
func MapToLabelSelector() mapstructure.DecodeHookFuncType {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any,
	) (any, error) {
		if f.Kind() != reflect.Map {
			return data, nil
		}

		if t != reflect.TypeFor[*metav1.LabelSelector]() {
			return data, nil
		}

		result := &metav1.LabelSelector{}
		fMap := data.(map[any]any)
		if matchLabels, ok := fMap["matchLabels"]; ok {
			fMap["matchlabels"] = matchLabels
			delete(fMap, "matchLabels")
		}
		if matchExpressions, ok := fMap["matchExpressions"]; ok {
			fMap["matchexpressions"] = matchExpressions
			delete(fMap, "matchExpressions")
		}

		b, err := yaml.Marshal(fMap)
		if err != nil {
			return nil, err
		}

		err = yaml.Unmarshal(b, result)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
}

func LoadFromFile(file string, target *Config) error {
	return unmarshal(target, file)
}

func LoadFromCLI(target *Config, flagSet *pflag.FlagSet) error {
	var err error
	// set the rest of the config attributes based on command-line flag values
	target.RootLogger = zap.New(zap.UseFlagOptions(&zapCmdLineOpts))
	klog.SetLogger(target.RootLogger)
	ctrl.SetLogger(target.RootLogger)

	if kubeConfigFilePath, changed, flagErr := getKubeConfigFilePath(flagSet); flagErr != nil {
		return flagErr
	} else if changed {
		target.KubeConfigFilePath = kubeConfigFilePath
	}
	clusterConfig, err := clientcmd.BuildConfigFromFlags("", target.KubeConfigFilePath)
	if err != nil {
		pathError := &fs.PathError{}
		if ok := errors.As(err, &pathError); !ok {
			return err
		}
		clusterConfig, err = rest.InClusterConfig()
		if err != nil {
			return err
		}
		target.KubeConfigFilePath = ""
	}
	target.ClusterConfig = clusterConfig

	if listenAddr, changed, flagErr := getListenAddr(flagSet); flagErr != nil {
		return flagErr
	} else if changed {
		target.ListenAddr = listenAddr
	}

	if prometheusCREnabled, changed, flagErr := getPrometheusCREnabled(flagSet); flagErr != nil {
		return flagErr
	} else if changed {
		target.PrometheusCR.Enabled = prometheusCREnabled
	}

	if httpsEnabled, changed, err := getHttpsEnabled(flagSet); err != nil {
		return err
	} else if changed {
		target.HTTPS.Enabled = httpsEnabled
	}

	if listenAddrHttps, changed, err := getHttpsListenAddr(flagSet); err != nil {
		return err
	} else if changed {
		target.HTTPS.ListenAddr = listenAddrHttps
	}

	if caFilePath, changed, err := getHttpsCAFilePath(flagSet); err != nil {
		return err
	} else if changed {
		target.HTTPS.CAFilePath = caFilePath
	}

	if tlsCertFilePath, changed, err := getHttpsTLSCertFilePath(flagSet); err != nil {
		return err
	} else if changed {
		target.HTTPS.TLSCertFilePath = tlsCertFilePath
	}

	if tlsKeyFilePath, changed, err := getHttpsTLSKeyFilePath(flagSet); err != nil {
		return err
	} else if changed {
		target.HTTPS.TLSKeyFilePath = tlsKeyFilePath
	}

	if allowInsecureAuthSecrets, changed, err := getAllowInsecureAuthSecrets(flagSet); err != nil {
		return err
	} else if changed {
		target.AllowInsecureAuthSecrets = allowInsecureAuthSecrets
	}

	return nil
}

// LoadFromEnv loads configuration from environment variables.
func LoadFromEnv(target *Config) error {
	if ns, ok := os.LookupEnv("OTELCOL_NAMESPACE"); ok {
		target.CollectorNamespace = ns
	}
	if val, ok := os.LookupEnv("ALLOW_INSECURE_AUTH_SECRETS"); ok && val == "true" {
		target.AllowInsecureAuthSecrets = true
	}
	return nil
}

// unmarshal decodes the contents of the configFile into the cfg argument, using a
// mapstructure decoder with the following notable behaviors.
// Decodes time.Duration from strings (see StringToModelDurationHookFunc).
// Allows custom unmarshaling for promconfig.Config struct that implements yaml.Unmarshaler (see MapToPromConfig).
// Allows custom unmarshaling for metav1.LabelSelector struct using both camelcase and lowercase field names (see MapToLabelSelector).
func unmarshal(cfg *Config, configFile string) error {
	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	m := make(map[string]any)
	err = yaml.Unmarshal(yamlFile, &m)
	if err != nil {
		return fmt.Errorf("error unmarshaling YAML: %w", err)
	}

	dc := mapstructure.DecoderConfig{
		TagName: "yaml",
		Result:  cfg,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			StringToModelOrTimeDurationHookFunc(),
			MapToPromConfig(),
			MapToMonitoringV1(),
			MapToLabelSelector(),
		),
	}

	decoder, err := mapstructure.NewDecoder(&dc)
	if err != nil {
		return err
	}
	return decoder.Decode(m)
}

func CreateDefaultConfig() Config {
	return Config{
		ListenAddr:         DefaultListenAddr,
		KubeConfigFilePath: DefaultKubeConfigFilePath,
		HTTPS: HTTPSServerConfig{
			ListenAddr: DefaultHttpsListenAddr,
		},
		AllocationStrategy:         DefaultAllocationStrategy,
		AllocationFallbackStrategy: "",
		FilterStrategy:             DefaultFilterStrategy,
		PrometheusCR: PrometheusCRConfig{
			ScrapeInterval:                  DefaultCRScrapeInterval,
			ServiceMonitorNamespaceSelector: &metav1.LabelSelector{},
			PodMonitorNamespaceSelector:     &metav1.LabelSelector{},
			ScrapeConfigNamespaceSelector:   &metav1.LabelSelector{},
			ProbeNamespaceSelector:          &metav1.LabelSelector{},
			ScrapeProtocols:                 defaultScrapeProtocolsCR,
		},
		CollectorNotReadyGracePeriod: DefaultCollectorNotReadyGracePeriod,
	}
}

func Load(args []string) (*Config, error) {
	var err error

	flagSet := getFlagSet(pflag.ExitOnError)
	err = flagSet.Parse(args)
	if err != nil {
		return nil, err
	}

	config := CreateDefaultConfig()

	// load the config from the config file
	configFilePath, err := getConfigFilePath(flagSet)
	if err != nil {
		return nil, err
	}
	err = LoadFromFile(configFilePath, &config)
	if err != nil {
		return nil, err
	}

	err = LoadFromEnv(&config)
	if err != nil {
		return nil, err
	}

	err = LoadFromCLI(&config, flagSet)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// ValidateConfig validates the cli and file configs together.
func ValidateConfig(config *Config) error {
	scrapeConfigsPresent := (config.PromConfig != nil && len(config.PromConfig.ScrapeConfigs) > 0)
	if !config.PrometheusCR.Enabled && !scrapeConfigsPresent {
		return errors.New("at least one scrape config must be defined, or Prometheus CR watching must be enabled")
	}
	if config.CollectorNamespace == "" {
		return errors.New("collector namespace must be set")
	}
	if len(config.PrometheusCR.AllowNamespaces) != 0 && len(config.PrometheusCR.DenyNamespaces) != 0 {
		return errors.New("only one of allowNamespaces or denyNamespaces can be set")
	}
	return validateTelemetry(config.Telemetry)
}

// validateTelemetry validates the self-telemetry configuration. The operator sets these
// values from a validated CRD, but the Target Allocator can also be run standalone with a
// config file, so we validate here as well for a clear error instead of a runtime failure.
func validateTelemetry(t TelemetryConfig) error {
	if t.Metrics == nil {
		return nil
	}
	for i, reader := range t.Metrics.Readers {
		if reader.Periodic == nil {
			continue
		}
		exp := reader.Periodic.Exporter
		if exp.OTLPGrpc == nil && exp.OTLPHttp == nil {
			return fmt.Errorf("telemetry.metrics.readers[%d].periodic: must configure otlp_grpc or otlp_http exporter", i)
		}
		if exp.OTLPGrpc != nil {
			if exp.OTLPGrpc.Endpoint == "" {
				return fmt.Errorf("telemetry.metrics.readers[%d].periodic.otlp_grpc: endpoint must be set", i)
			}
			switch exp.OTLPGrpc.TemporalityPreference {
			case "", "cumulative", "delta", "low_memory":
			default:
				return fmt.Errorf("telemetry.metrics.readers[%d].periodic.otlp_grpc: temporality_preference must be 'cumulative', 'delta', or 'low_memory', got %q", i, exp.OTLPGrpc.TemporalityPreference)
			}
		}
		if exp.OTLPHttp != nil {
			if exp.OTLPHttp.Endpoint == "" {
				return fmt.Errorf("telemetry.metrics.readers[%d].periodic.otlp_http: endpoint must be set", i)
			}
			switch exp.OTLPHttp.TemporalityPreference {
			case "", "cumulative", "delta", "low_memory":
			default:
				return fmt.Errorf("telemetry.metrics.readers[%d].periodic.otlp_http: temporality_preference must be 'cumulative', 'delta', or 'low_memory', got %q", i, exp.OTLPHttp.TemporalityPreference)
			}
		}
	}
	return nil
}

func (c HTTPSServerConfig) NewTLSConfig(logger logr.Logger) (*tls.Config, *certwatcher.CertWatcher, error) {
	// Create certwatcher for server certificate/key reloading
	certWatcher, err := certwatcher.New(c.TLSCertFilePath, c.TLSKeyFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cert watcher: %w", err)
	}

	// Create CA reloader for client CA certificate reloading
	caReloader, err := NewCAReloader(c.CAFilePath, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create CA reloader: %w", err)
	}

	// Register callback to reload CA when server cert changes
	// Since Kubernetes updates secrets atomically, the CA will be updated at the same time
	certWatcher.RegisterCallback(func(tls.Certificate) {
		if reloadErr := caReloader.Reload(); reloadErr != nil {
			logger.Error(reloadErr, "Failed to reload CA via callback")
		}
	})

	tlsConfig := &tls.Config{
		GetCertificate: certWatcher.GetCertificate,
		// Request client certificate but don't verify automatically
		// We'll do custom verification in VerifyConnection with the dynamic CA pool
		ClientAuth: tls.RequestClientCert,
		MinVersion: tls.VersionTLS12,
		// Use VerifyConnection for dynamic CA pool access
		// This allows the CA pool to be reloaded at runtime
		VerifyConnection: func(cs tls.ConnectionState) error {
			// Require client certificate
			if len(cs.PeerCertificates) == 0 {
				return errors.New("no client certificate provided")
			}

			// Verify using current CA pool (which can be reloaded)
			opts := x509.VerifyOptions{
				Roots:         caReloader.GetClientCAs(),
				Intermediates: x509.NewCertPool(),
				KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			}

			// Add intermediate certificates to the pool
			for _, cert := range cs.PeerCertificates[1:] {
				opts.Intermediates.AddCert(cert)
			}

			// Verify only the leaf certificate
			if _, err := cs.PeerCertificates[0].Verify(opts); err != nil {
				return fmt.Errorf("client certificate verification failed: %w", err)
			}
			return nil
		},
	}

	return tlsConfig, certWatcher, nil
}

// GetSecretsAllowList returns the namespaces to watch for secrets as a map.
// If SecretNamespaces is explicitly configured, those namespaces are used.
// Otherwise, it defaults to the collectorNamespace (the target allocator's own namespace).
func (c PrometheusCRConfig) GetSecretsAllowList(collectorNamespace string) map[string]struct{} {
	secretsAllowList := make(map[string]struct{})
	if len(c.SecretNamespaces) > 0 {
		for _, ns := range c.SecretNamespaces {
			secretsAllowList[ns] = struct{}{}
		}
	} else if collectorNamespace != "" {
		secretsAllowList[collectorNamespace] = struct{}{}
	}
	return secretsAllowList
}

func (c PrometheusCRConfig) GetAllowDenyLists() (allowList, denyList map[string]struct{}) {
	allowList = map[string]struct{}{}
	if len(c.AllowNamespaces) != 0 {
		for _, ns := range c.AllowNamespaces {
			allowList[ns] = struct{}{}
		}
	} else {
		allowList = map[string]struct{}{v1.NamespaceAll: {}}
	}

	denyList = map[string]struct{}{}
	if len(c.DenyNamespaces) != 0 {
		for _, ns := range c.DenyNamespaces {
			denyList[ns] = struct{}{}
		}
	}

	return allowList, denyList
}
