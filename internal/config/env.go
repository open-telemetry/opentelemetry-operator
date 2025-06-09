// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
)

func ApplyEnvVars(cfg *Config) {
	if v, ok := os.LookupEnv("RELATED_IMAGE_COLLECTOR"); ok {
		cfg.CollectorImage = v
	}
	if v, ok := os.LookupEnv("RELATED_IMAGE_TARGET_ALLOCATOR"); ok {
		cfg.TargetAllocatorImage = v
	}
	if v, ok := os.LookupEnv("RELATED_IMAGE_OPERATOR_OPAMP_BRIDGE"); ok {
		cfg.OperatorOpAMPBridgeImage = v
	}
	if v, ok := os.LookupEnv("RELATED_IMAGE_AUTO_INSTRUMENTATION_JAVA"); ok {
		cfg.AutoInstrumentationJavaImage = v
	}
	if v, ok := os.LookupEnv("RELATED_IMAGE_AUTO_INSTRUMENTATION_NODEJS"); ok {
		cfg.AutoInstrumentationNodeJSImage = v
	}
	if v, ok := os.LookupEnv("RELATED_IMAGE_AUTO_INSTRUMENTATION_PYTHON"); ok {
		cfg.AutoInstrumentationPythonImage = v
	}
	if v, ok := os.LookupEnv("RELATED_IMAGE_AUTO_INSTRUMENTATION_DOTNET"); ok {
		cfg.AutoInstrumentationDotNetImage = v
	}
	if v, ok := os.LookupEnv("RELATED_IMAGE_AUTO_INSTRUMENTATION_GO"); ok {
		cfg.AutoInstrumentationGoImage = v
	}
	if v, ok := os.LookupEnv("RELATED_IMAGE_AUTO_INSTRUMENTATION_APACHE_HTTPD"); ok {
		cfg.AutoInstrumentationApacheHttpdImage = v
	}
	if v, ok := os.LookupEnv("RELATED_IMAGE_AUTO_INSTRUMENTATION_NGINX"); ok {
		cfg.AutoInstrumentationNginxImage = v
	}
}
