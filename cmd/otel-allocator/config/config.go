// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/go-logr/logr"
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
	ListenAddr         string                `yaml:"listen_addr,omitempty"`
	KubeConfigFilePath string                `yaml:"kube_config_file_path,omitempty"`
	ClusterConfig      *rest.Config          `yaml:"-"`
	RootLogger         logr.Logger           `yaml:"-"`
	CollectorSelector  *metav1.LabelSelector `yaml:"collector_selector,omitempty"`
	PromConfig         *promconfig.Config    `yaml:"config"`
	AllocationStrategy string                `yaml:"allocation_strategy,omitempty"`
	FilterStrategy     string                `yaml:"filter_strategy,omitempty"`
	PrometheusCR       PrometheusCRConfig    `yaml:"prometheus_cr,omitempty"`
	HTTPS              HTTPSServerConfig     `yaml:"https,omitempty"`
}

type PrometheusCRConfig struct {
	Enabled                         bool                  `yaml:"enabled,omitempty"`
	PodMonitorSelector              *metav1.LabelSelector `yaml:"pod_monitor_selector,omitempty"`
	ServiceMonitorSelector          *metav1.LabelSelector `yaml:"service_monitor_selector,omitempty"`
	ServiceMonitorNamespaceSelector *metav1.LabelSelector `yaml:"service_monitor_namespace_selector,omitempty"`
	PodMonitorNamespaceSelector     *metav1.LabelSelector `yaml:"pod_monitor_namespace_selector,omitempty"`
	ScrapeInterval                  model.Duration        `yaml:"scrape_interval,omitempty"`
}

type HTTPSServerConfig struct {
	Enabled         bool   `yaml:"enabled,omitempty"`
	ListenAddr      string `yaml:"listen_addr,omitempty"`
	CAFilePath      string `yaml:"ca_file_path,omitempty"`
	TLSCertFilePath string `yaml:"tls_cert_file_path,omitempty"`
	TLSKeyFilePath  string `yaml:"tls_key_file_path,omitempty"`
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

func unmarshal(cfg *Config, configFile string) error {
	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	if err = yaml.Unmarshal(yamlFile, cfg); err != nil {
		return fmt.Errorf("error unmarshaling YAML: %w", err)
	}
	return nil
}

func CreateDefaultConfig() Config {
	return Config{
		AllocationStrategy: DefaultAllocationStrategy,
		FilterStrategy:     DefaultFilterStrategy,
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
