// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/operator-framework/operator-lib/proxy"
)

func ApplyEnvVars(cfg *Config) {
	applyEnv(cfg)
	cfg.ProxyEnvVars = proxy.ReadProxyVarsFromEnv()
}
