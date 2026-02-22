// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"

	"sigs.k8s.io/yaml"
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
