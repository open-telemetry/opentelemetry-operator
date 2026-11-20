// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"

	"github.com/operator-framework/operator-lib/proxy"
)

func ApplyEnvVars(cfg *Config) error {
	if err := applyEnv(cfg, os.LookupEnv); err != nil {
		return err
	}
	cfg.ProxyEnvVars = proxy.ReadProxyVarsFromEnv()
	return nil
}
