// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"

	"gopkg.in/yaml.v3"
	k8syaml "sigs.k8s.io/yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

// ApplyConfigFile applies the yaml file contents to the configuration.
func ApplyConfigFile(file string, c *Config) error {
	contents, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(contents, c); err != nil {
		return err
	}
	// Operator Config fields use kebab-case yaml tags and load with
	// gopkg.in/yaml.v3. The instrumentations block embeds Instrumentation /
	// corev1 API types that only have json tags (CRD camelCase). yaml.v3
	// ignores those tags, so re-parse instrumentations with sigs.k8s.io/yaml.
	return overlayInstrumentationFromJSONTags(contents, c)
}

func overlayInstrumentationFromJSONTags(contents []byte, c *Config) error {
	var partial struct {
		Instrumentation *v1alpha1.Instrumentation `json:"instrumentations"`
	}
	if err := k8syaml.Unmarshal(contents, &partial); err != nil {
		return err
	}
	if partial.Instrumentation == nil {
		return nil
	}
	c.Instrumentation = *partial.Instrumentation
	return nil
}
