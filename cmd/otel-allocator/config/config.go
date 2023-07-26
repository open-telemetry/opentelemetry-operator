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
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const DefaultResyncTime = 5 * time.Minute
const DefaultConfigFilePath string = "/conf/targetallocator.yaml"
const DefaultCRScrapeInterval model.Duration = model.Duration(time.Second * 30)

type Config struct {
	ListenAddr             string             `yaml:"listen_addr,omitempty"`
	KubeConfigFilePath     string             `yaml:"kube_config_file_path,omitempty"`
	ClusterConfig          *rest.Config       `yaml:"-"`
	RootLogger             logr.Logger        `yaml:"-"`
	LabelSelector          map[string]string  `yaml:"label_selector,omitempty"`
	PromConfig             *promconfig.Config `yaml:"config"`
	AllocationStrategy     *string            `yaml:"allocation_strategy,omitempty"`
	FilterStrategy         *string            `yaml:"filter_strategy,omitempty"`
	PrometheusCR           PrometheusCRConfig `yaml:"prometheus_cr,omitempty"`
	PodMonitorSelector     map[string]string  `yaml:"pod_monitor_selector,omitempty"`
	ServiceMonitorSelector map[string]string  `yaml:"service_monitor_selector,omitempty"`
}

type PrometheusCRConfig struct {
	Enabled        bool           `yaml:"enabled,omitempty"`
	ScrapeInterval model.Duration `yaml:"scrape_interval,omitempty"`
}

func (c Config) GetAllocationStrategy() string {
	if c.AllocationStrategy != nil {
		return *c.AllocationStrategy
	}
	return "least-weighted"
}

func (c Config) GetTargetsFilterStrategy() string {
	if c.FilterStrategy != nil {
		return *c.FilterStrategy
	}
	return ""
}

func Load(file string) (Config, error) {
	cfg := createDefaultConfig()
	if err := unmarshal(&cfg, file); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func unmarshal(cfg *Config, configFile string) error {

	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	if err = yaml.UnmarshalStrict(yamlFile, cfg); err != nil {
		return fmt.Errorf("error unmarshaling YAML: %w", err)
	}
	return nil
}

func createDefaultConfig() Config {
	return Config{
		PrometheusCR: PrometheusCRConfig{
			ScrapeInterval: DefaultCRScrapeInterval,
		},
	}
}

func FromCLI() (*Config, string, error) {
	pflag.Parse()

	// load the config from the config file
	config, err := Load(*configFilePathFlag)
	if err != nil {
		return nil, "", err
	}

	// set the rest of the config attributes based on command-line flag values
	config.RootLogger = zap.New(zap.UseFlagOptions(&zapCmdLineOpts))
	klog.SetLogger(config.RootLogger)
	ctrl.SetLogger(config.RootLogger)

	config.KubeConfigFilePath = *kubeConfigPathFlag
	clusterConfig, err := clientcmd.BuildConfigFromFlags("", *kubeConfigPathFlag)
	if err != nil {
		pathError := &fs.PathError{}
		if ok := errors.As(err, &pathError); !ok {
			return nil, "", err
		}
		clusterConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, "", err
		}
	}
	config.ListenAddr = *listenAddrFlag
	config.PrometheusCR.Enabled = *prometheusCREnabledFlag
	config.ClusterConfig = clusterConfig
	return &config, *configFilePathFlag, nil
}

// ValidateConfig validates the cli and file configs together.
func ValidateConfig(config *Config) error {
	scrapeConfigsPresent := (config.PromConfig != nil && len(config.PromConfig.ScrapeConfigs) > 0)
	if !(config.PrometheusCR.Enabled || scrapeConfigsPresent) {
		return fmt.Errorf("at least one scrape config must be defined, or Prometheus CR watching must be enabled")
	}
	return nil
}
