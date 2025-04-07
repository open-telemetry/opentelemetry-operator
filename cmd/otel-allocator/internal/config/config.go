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
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-viper/mapstructure/v2"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	_ "github.com/prometheus/prometheus/discovery/install"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	DefaultResyncTime                                  = 5 * time.Minute
	DefaultConfigFilePath               string         = "/conf/targetallocator.yaml"
	DefaultCRScrapeInterval             model.Duration = model.Duration(time.Second * 30)
	DefaultAllocationStrategy                          = "consistent-hashing"
	DefaultFilterStrategy                              = "relabel-config"
	DefaultCollectorNotReadyGracePeriod                = 0 * time.Second
)

// By default, scrape protocols include PrometheusText1_0_0, which only Prometheus >=3.0 supports.
// Manually exclude this protocol until several versions of the Otel Collector support it.
var DefaultScrapeProtocols = []promconfig.ScrapeProtocol{
	promconfig.OpenMetricsText1_0_0,
	promconfig.OpenMetricsText0_0_1,
	promconfig.PrometheusText0_0_4,
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
	CollectorNotReadyGracePeriod time.Duration         `yaml:"collector_not_ready_grace_period,omitempty"`
}

type PrometheusCRConfig struct {
	Enabled                         bool                  `yaml:"enabled,omitempty"`
	AllowNamespaces                 []string              `yaml:"allow_namespaces,omitempty"`
	DenyNamespaces                  []string              `yaml:"deny_namespaces,omitempty"`
	PodMonitorSelector              *metav1.LabelSelector `yaml:"pod_monitor_selector,omitempty"`
	PodMonitorNamespaceSelector     *metav1.LabelSelector `yaml:"pod_monitor_namespace_selector,omitempty"`
	ServiceMonitorSelector          *metav1.LabelSelector `yaml:"service_monitor_selector,omitempty"`
	ServiceMonitorNamespaceSelector *metav1.LabelSelector `yaml:"service_monitor_namespace_selector,omitempty"`
	ScrapeConfigSelector            *metav1.LabelSelector `yaml:"scrape_config_selector,omitempty"`
	ScrapeConfigNamespaceSelector   *metav1.LabelSelector `yaml:"scrape_config_namespace_selector,omitempty"`
	ProbeSelector                   *metav1.LabelSelector `yaml:"probe_selector,omitempty"`
	ProbeNamespaceSelector          *metav1.LabelSelector `yaml:"probe_namespace_selector,omitempty"`
	ScrapeInterval                  model.Duration        `yaml:"scrape_interval,omitempty"`
}

type HTTPSServerConfig struct {
	Enabled         bool   `yaml:"enabled,omitempty"`
	ListenAddr      string `yaml:"listen_addr,omitempty"`
	CAFilePath      string `yaml:"ca_file_path,omitempty"`
	TLSCertFilePath string `yaml:"tls_cert_file_path,omitempty"`
	TLSKeyFilePath  string `yaml:"tls_key_file_path,omitempty"`
}

// StringToModelOrTimeDurationHookFunc returns a DecodeHookFuncType
// that converts string to time.Duration, which can also be used
// as model.Duration.
func StringToModelOrTimeDurationHookFunc() mapstructure.DecodeHookFuncType {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}

		if t != reflect.TypeOf(model.Duration(5)) && t != reflect.TypeOf(time.Duration(5)) {
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
		data interface{},
	) (interface{}, error) {
		if f.Kind() != reflect.Map {
			return data, nil
		}

		if t != reflect.TypeOf(&promconfig.Config{}) {
			return data, nil
		}

		pConfig := &promconfig.Config{}

		dataMap := data.(map[any]any)
		err := ApplyPromConfigDefaults(dataMap)
		if err != nil {
			return nil, err
		}
		mb, err := yaml.Marshal(dataMap)
		if err != nil {
			return nil, err
		}

		err = yaml.Unmarshal(mb, pConfig)
		if err != nil {
			return nil, err
		}
		return pConfig, nil
	}
}

// applyPromConfigDefaults applies our own defaults to the Prometheus configuration. The unmarshalling process for
// Prometheus config is quite involved, and as a result, we need to apply our own defaults before it happens.
func ApplyPromConfigDefaults(promcCfgMap map[any]any) error {
	// use our own struct definition here because we don't want Prometheus unmarshalling logic to apply here
	promCfg := struct {
		GlobalConfig struct {
			ScrapeProtocols []promconfig.ScrapeProtocol `mapstructure:"scrape_protocols"`
			Rest            map[any]any                 `mapstructure:",remain"`
		} `mapstructure:"global"`
		Rest map[any]any `mapstructure:",remain"`
	}{}
	err := mapstructure.Decode(promcCfgMap, &promCfg)
	if err != nil {
		return err
	}
	// apply defaults here
	promCfg.GlobalConfig.ScrapeProtocols = DefaultScrapeProtocols

	// decode back into the map
	err = mapstructure.Decode(promCfg, &promcCfgMap)
	if err != nil {
		return err
	}
	return nil
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
		data interface{},
	) (interface{}, error) {
		if f.Kind() != reflect.Map {
			return data, nil
		}

		if t != reflect.TypeOf(&metav1.LabelSelector{}) {
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

	target.KubeConfigFilePath, err = getKubeConfigFilePath(flagSet)
	if err != nil {
		return err
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

	target.ListenAddr, err = getListenAddr(flagSet)
	if err != nil {
		return err
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

	return nil
}

// LoadFromEnv loads configuration from environment variables.
func LoadFromEnv(target *Config) error {
	target.CollectorNamespace = os.Getenv("OTELCOL_NAMESPACE")
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

	m := make(map[string]interface{})
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
			MapToLabelSelector(),
		),
	}

	decoder, err := mapstructure.NewDecoder(&dc)
	if err != nil {
		return err
	}
	if err := decoder.Decode(m); err != nil {
		return err
	}

	return nil
}

func CreateDefaultConfig() Config {
	return Config{
		AllocationStrategy:         DefaultAllocationStrategy,
		AllocationFallbackStrategy: "",
		FilterStrategy:             DefaultFilterStrategy,
		PrometheusCR: PrometheusCRConfig{
			ScrapeInterval:                  DefaultCRScrapeInterval,
			ServiceMonitorNamespaceSelector: &metav1.LabelSelector{},
			PodMonitorNamespaceSelector:     &metav1.LabelSelector{},
			ScrapeConfigNamespaceSelector:   &metav1.LabelSelector{},
			ProbeNamespaceSelector:          &metav1.LabelSelector{},
		},
		CollectorNotReadyGracePeriod: DefaultCollectorNotReadyGracePeriod,
	}
}

func Load() (*Config, error) {
	var err error

	flagSet := getFlagSet(pflag.ExitOnError)
	err = flagSet.Parse(os.Args)
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
	if !(config.PrometheusCR.Enabled || scrapeConfigsPresent) {
		return fmt.Errorf("at least one scrape config must be defined, or Prometheus CR watching must be enabled")
	}
	if config.CollectorNamespace == "" {
		return fmt.Errorf("collector namespace must be set")
	}
	if len(config.PrometheusCR.AllowNamespaces) != 0 && len(config.PrometheusCR.DenyNamespaces) != 0 {
		return fmt.Errorf("only one of allowNamespaces or denyNamespaces can be set")
	}
	return nil
}

func (c HTTPSServerConfig) NewTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(c.TLSCertFilePath, c.TLSKeyFilePath)
	if err != nil {
		return nil, err
	}

	caCert, err := os.ReadFile(c.CAFilePath)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		MinVersion:   tls.VersionTLS12,
	}
	return tlsConfig, nil
}

// GetAllowDenyLists returns the allow and deny lists as maps. If the allow list is empty, it defaults to all namespaces.
// If the deny list is empty, it defaults to an empty map.
func (c PrometheusCRConfig) GetAllowDenyLists() (map[string]struct{}, map[string]struct{}) {
	allowList := map[string]struct{}{}
	if len(c.AllowNamespaces) != 0 {
		for _, ns := range c.AllowNamespaces {
			allowList[ns] = struct{}{}
		}
	} else {
		allowList = map[string]struct{}{v1.NamespaceAll: {}}
	}

	denyList := map[string]struct{}{}
	if len(c.DenyNamespaces) != 0 {
		for _, ns := range c.DenyNamespaces {
			denyList[ns] = struct{}{}
		}
	}

	return allowList, denyList
}
