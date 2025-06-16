// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v2"
)

// ApplyConfigFile applies the yaml file contents to the configuration.
func ApplyConfigFile(file string, c *Config, log logr.Logger) error {
	contents, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(contents, c)
	return err
}
