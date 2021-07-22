package config

import (
	"errors"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// ErrInvalidYAML represents an error in the format of the original YAML configuration file.
var ErrInvalidYAML = errors.New("couldn't parse the loadbalancer configuration")

const defaultConfigFile string = "/conf/targetallocator.yaml"

type Config struct {
	LabelSelector map[string]string `yaml:"label_selector,omitempty"`
	Config        ScrapeConfig      `yaml:"config"`
}

type ScrapeConfig struct {
	ScrapeConfigs []map[string]interface{} `yaml:"scrape_configs"`
}

func Load(file string) (Config, error) {
	if file == "" {
		file = defaultConfigFile
	}

	var cfg Config
	if err := unmarshall(&cfg, file); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func unmarshall(cfg *Config, configFile string) error {
	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	if err = yaml.UnmarshalStrict(yamlFile, cfg); err != nil {
		return fmt.Errorf("error unmarshaling YAML: %w", err)
	}
	return nil
}
