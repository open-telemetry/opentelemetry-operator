// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	DefaultResyncTime                        = 5 * time.Minute
	DefaultConfigFilePath     string         = "/conf/targetallocator.yaml"
	DefaultCRScrapeInterval   model.Duration = model.Duration(time.Second * 30)
	DefaultAllocationStrategy                = "consistent-hashing"
	DefaultFilterStrategy                    = "relabel-config"
)

type Config struct {
	ListenAddr                 string                `yaml:"listen_addr,omitempty"`
	KubeConfigFilePath         string                `yaml:"kube_config_file_path,omitempty"`
	ClusterConfig              *rest.Config          `yaml:"-"`
	RootLogger                 logr.Logger           `yaml:"-"`
	CollectorSelector          *metav1.LabelSelector `yaml:"collector_selector,omitempty"`
	PromConfig                 *promconfig.Config    `yaml:"config"`
	AllocationStrategy         string                `yaml:"allocation_strategy,omitempty"`
	AllocationFallbackStrategy string                `yaml:"allocation_fallback_strategy,omitempty"`
	FilterStrategy             string                `yaml:"filter_strategy,omitempty"`
	PrometheusCR               PrometheusCRConfig    `yaml:"prometheus_cr,omitempty"`
	HTTPS                      HTTPSServerConfig     `yaml:"https,omitempty"`
}

type PrometheusCRConfig struct {
	Enabled                         bool                  `yaml:"enabled,omitempty"`
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

// StringToModelDurationHookFunc returns a DecodeHookFuncType
// that converts string to time.Duration, which can be used
// as model.Duration.
func StringToModelDurationHookFunc() mapstructure.DecodeHookFuncType {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}

		if t != reflect.TypeOf(model.Duration(5)) {
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

		mb, err := yaml.Marshal(data.(map[any]any))
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
			StringToModelDurationHookFunc(),
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
			ScrapeInterval: DefaultCRScrapeInterval,
		},
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
