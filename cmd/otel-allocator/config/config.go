package config

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/go-logr/logr"
	promconfig "github.com/prometheus/prometheus/config"
	_ "github.com/prometheus/prometheus/discovery/install"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/util/homedir"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// ErrInvalidYAML represents an error in the format of the original YAML configuration file.
var (
	ErrInvalidYAML = errors.New("couldn't parse the loadbalancer configuration")
	cLIConf        = CLIConfig{
		ListenAddr:     pflag.String("listen-addr", ":8080", "The address where this service serves."),
		ConfigFilePath: pflag.String("config-file", DefaultConfigFilePath, "The path to the config file."),
		KubeconfigPath: pflag.String("KubeconfigPath", filepath.Join(homedir.HomeDir(), ".kube", "config"), "absolute path to the KubeconfigPath file"),
	}
)

const DefaultConfigFilePath string = "/conf/targetallocator.yaml"

type Config struct {
	LabelSelector map[string]string  `yaml:"label_selector,omitempty"`
	Config        *promconfig.Config `yaml:"config"`
}

type CLIConfig struct {
	ListenAddr     *string
	ConfigFilePath *string
	KubeconfigPath *string
	RootLogger     logr.Logger
}

func Load(file string) (Config, error) {
	var cfg Config
	if err := unmarshal(&cfg, file); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func unmarshal(cfg *Config, configFile string) error {

	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	if err = yaml.UnmarshalStrict(yamlFile, cfg); err != nil {
		return fmt.Errorf("error unmarshaling YAML: %w", err)
	}
	return nil
}

func ParseCLI() CLIConfig {
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	pflag.Parse()

	cLIConf.RootLogger = zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(cLIConf.RootLogger)
	return cLIConf
}
