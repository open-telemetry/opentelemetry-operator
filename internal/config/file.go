// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

// ApplyConfigFile applies the yaml file contents to the configuration.
func ApplyConfigFile(file string, c *Config) error {
	contents, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(contents, c)
	return err
}
