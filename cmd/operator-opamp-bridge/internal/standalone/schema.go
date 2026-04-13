// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"errors"
	"fmt"
	"strings"
)

const standaloneConfigVersion = "opentelemetry.io/opamp-bridge-standalone/v1alpha1"

type standaloneConfig struct {
	Version   string            `json:"version" yaml:"version"`
	Name      string            `json:"name" yaml:"name"`
	Namespace string            `json:"namespace" yaml:"namespace"`
	Config    map[string]string `json:"config" yaml:"config"`
}

func (c standaloneConfig) validate(name, namespace string) error {
	if c.Version != standaloneConfigVersion {
		return fmt.Errorf("unsupported standalone config version %q", c.Version)
	}
	if c.Name == "" {
		return errors.New("standalone config name is required")
	}
	if c.Namespace == "" {
		return errors.New("standalone config namespace is required")
	}
	if c.Name != name {
		return fmt.Errorf("standalone config name %q does not match target name %q", c.Name, name)
	}
	if c.Namespace != namespace {
		return fmt.Errorf("standalone config namespace %q does not match target namespace %q", c.Namespace, namespace)
	}
	if len(c.Config) == 0 {
		return errors.New("standalone config data is required")
	}
	for key := range c.Config {
		if strings.TrimSpace(key) == "" {
			return errors.New("standalone config contains an empty data key")
		}
	}
	return nil
}
