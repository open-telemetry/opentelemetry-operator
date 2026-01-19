// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"strconv"
	"strings"
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

	if v, ok := os.LookupEnv("OPENSHIFT_CREATE_DASHBOARD"); ok {
		cfg.OpenshiftCreateDashboard, _ = strconv.ParseBool(v)
	}
	if v, ok := os.LookupEnv("METRICS_ADDR"); ok {
		cfg.MetricsAddr = v
	}
	if v, ok := os.LookupEnv("METRICS_SECURE"); ok {
		cfg.MetricsSecure, _ = strconv.ParseBool(v)
	}
	if v, ok := os.LookupEnv("METRICS_TLS_CERT_FILE"); ok {
		cfg.MetricsTLSCertFile = v
	}
	if v, ok := os.LookupEnv("METRICS_TLS_KEY_FILE"); ok {
		cfg.MetricsTLSKeyFile = v
	}
	if v, ok := os.LookupEnv("HEALTH_PROBE_ADDR"); ok {
		cfg.ProbeAddr = v
	}
	if v, ok := os.LookupEnv("PPROF_ADDR"); ok {
		cfg.PprofAddr = v
	}
	if v, ok := os.LookupEnv("ENABLE_LEADER_ELECTION"); ok {
		cfg.EnableLeaderElection, _ = strconv.ParseBool(v)
	}
	if v, ok := os.LookupEnv("ENABLE_CR_METRICS"); ok {
		cfg.EnableCRMetrics, _ = strconv.ParseBool(v)
	}
	if v, ok := os.LookupEnv("CREATE_SM_OPERATOR_METRICS"); ok {
		cfg.CreateServiceMonitorOperatorMetrics, _ = strconv.ParseBool(v)
	}
	if v, ok := os.LookupEnv("WEBHOOK_PORT"); ok {
		cfg.WebhookPort, _ = strconv.Atoi(v)
	}
	if v, ok := os.LookupEnv("FIPS_DISABLED_COMPONENTS"); ok {
		cfg.FipsDisabledComponents = v
	}
	if v, ok := os.LookupEnv("TLS_MIN_VERSION"); ok {
		cfg.TLS.MinVersion = v
	}
	if v, ok := os.LookupEnv("TLS_CIPHER_SUITES"); ok {
		cfg.TLS.CipherSuites = strings.Split(v, ",")
	}
	if v, ok := os.LookupEnv("LABELS_FILTER"); ok {
		cfg.LabelsFilter = strings.Split(v, ",")
	}
	if v, ok := os.LookupEnv("ANNOTATIONS_FILTER"); ok {
		cfg.AnnotationsFilter = strings.Split(v, ",")
	}
	if v, ok := os.LookupEnv("ZAP_TIME_KEY"); ok {
		cfg.Zap.TimeKey = v
	}
	if v, ok := os.LookupEnv("ZAP_LEVEL_FORMAT"); ok {
		cfg.Zap.LevelFormat = v
	}
	if v, ok := os.LookupEnv("ZAP_LEVEL_KEY"); ok {
		cfg.Zap.LevelKey = v
	}
	if v, ok := os.LookupEnv("ZAP_MESSAGE_KEY"); ok {
		cfg.Zap.MessageKey = v
	}
	if v, ok := os.LookupEnv("ENABLE_JAVA_AUTO_INSTRUMENTATION"); ok {
		cfg.EnableJavaAutoInstrumentation, _ = strconv.ParseBool(v)
	}
	if v, ok := os.LookupEnv("ENABLE_NODEJS_AUTO_INSTRUMENTATION"); ok {
		cfg.EnableNodeJSAutoInstrumentation, _ = strconv.ParseBool(v)
	}
	if v, ok := os.LookupEnv("ENABLE_DOTNET_AUTO_INSTRUMENTATION"); ok {
		cfg.EnableDotNetAutoInstrumentation, _ = strconv.ParseBool(v)
	}
	if v, ok := os.LookupEnv("ENABLE_GO_AUTO_INSTRUMENTATION"); ok {
		cfg.EnableGoAutoInstrumentation, _ = strconv.ParseBool(v)
	}
	if v, ok := os.LookupEnv("ENABLE_NGINX_AUTO_INSTRUMENTATION"); ok {
		cfg.EnableNginxAutoInstrumentation, _ = strconv.ParseBool(v)
	}
	if v, ok := os.LookupEnv("ENABLE_APACHE_HTTPD_AUTO_INSTRUMENTATION"); ok {
		cfg.EnableApacheHttpdInstrumentation, _ = strconv.ParseBool(v)
	}
	if v, ok := os.LookupEnv("ENABLE_PYTHON_AUTO_INSTRUMENTATION"); ok {
		cfg.EnablePythonAutoInstrumentation, _ = strconv.ParseBool(v)
	}
	if v, ok := os.LookupEnv("IGNORE_MISSING_COLLECTOR_CRDS"); ok {
		cfg.IgnoreMissingCollectorCRDs, _ = strconv.ParseBool(v)
	}
	if v, ok := os.LookupEnv("ENABLE_WEBHOOKS"); ok {
		cfg.EnableWebhooks, _ = strconv.ParseBool(v)
	}
	if v, ok := os.LookupEnv("FEATURE_GATES"); ok {
		cfg.FeatureGates = v
	}
	if v, ok := os.LookupEnv("ENABLE_MULTI_INSTRUMENTATION"); ok {
		cfg.EnableMultiInstrumentation, _ = strconv.ParseBool(v)
	}
}
