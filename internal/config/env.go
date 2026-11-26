// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/operator-framework/operator-lib/proxy"
)

// ApplyEnvVars sets the options whose environment variable is set. It returns an
// error for every variable whose value can't be parsed, and leaves that option
// unchanged.
func ApplyEnvVars(cfg *Config) error {
	var errs []error
	if v, ok := os.LookupEnv("RELATED_IMAGE_COLLECTOR"); ok {
		cfg.CollectorImage = v
	}
	if v, ok := os.LookupEnv("RELATED_IMAGE_CLUSTER_OBSERVABILITY_COLLECTOR"); ok {
		cfg.ClusterObservabilityCollectorImage = v
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
	if v, ok := os.LookupEnv("RELATED_IMAGE_AUTO_INSTRUMENTATION_PHP"); ok {
		cfg.AutoInstrumentationPhpImage = v
	}
	if v, ok := os.LookupEnv("RELATED_IMAGE_AUTO_INSTRUMENTATION_PYTHON"); ok {
		cfg.AutoInstrumentationPythonImage = v
	}
	if v, ok := os.LookupEnv("RELATED_IMAGE_AUTO_INSTRUMENTATION_RUBY"); ok {
		cfg.AutoInstrumentationRubyImage = v
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

	errs = append(errs, boolFromEnv("OPENSHIFT_CREATE_DASHBOARD", &cfg.OpenshiftCreateDashboard))
	if v, ok := os.LookupEnv("METRICS_ADDR"); ok {
		cfg.MetricsAddr = v
	}
	errs = append(errs, boolFromEnv("METRICS_SECURE", &cfg.MetricsSecure))
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
	errs = append(errs, boolFromEnv("ENABLE_LEADER_ELECTION", &cfg.EnableLeaderElection))
	errs = append(errs, boolFromEnv("ENABLE_CR_METRICS", &cfg.EnableCRMetrics))
	errs = append(errs, boolFromEnv("CREATE_SM_OPERATOR_METRICS", &cfg.CreateServiceMonitorOperatorMetrics))
	errs = append(errs, intFromEnv("WEBHOOK_PORT", &cfg.WebhookPort))
	if v, ok := os.LookupEnv("FIPS_DISABLED_COMPONENTS"); ok {
		cfg.FipsDisabledComponents = v
	}
	errs = append(errs, boolFromEnv("TLS_CLUSTER_PROFILE", &cfg.TLS.UseClusterProfile))
	errs = append(errs, boolFromEnv("TLS_CONFIGURE_OPERANDS", &cfg.TLS.ConfigureOperands))
	if v, ok := os.LookupEnv("TLS_MIN_VERSION"); ok {
		cfg.TLS.MinVersion = v
	}
	if v, ok := os.LookupEnv("TLS_CIPHER_SUITES"); ok {
		cfg.TLS.CipherSuites = strings.Split(v, ",")
	}
	if v, ok := os.LookupEnv("TLS_CURVE_PREFERENCES"); ok {
		if curves, err := parseCurvePreferences(v); err != nil {
			errs = append(errs, envVarError("TLS_CURVE_PREFERENCES", v, err))
		} else {
			cfg.TLS.CurvePreferences = curves
		}
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
	errs = append(errs, boolFromEnv("ENABLE_JAVA_AUTO_INSTRUMENTATION", &cfg.EnableJavaAutoInstrumentation))
	errs = append(errs, boolFromEnv("ENABLE_NODEJS_AUTO_INSTRUMENTATION", &cfg.EnableNodeJSAutoInstrumentation))
	errs = append(errs, boolFromEnv("ENABLE_DOTNET_AUTO_INSTRUMENTATION", &cfg.EnableDotNetAutoInstrumentation))
	errs = append(errs, boolFromEnv("ENABLE_GO_AUTO_INSTRUMENTATION", &cfg.EnableGoAutoInstrumentation))
	errs = append(errs, boolFromEnv("ENABLE_NGINX_AUTO_INSTRUMENTATION", &cfg.EnableNginxAutoInstrumentation))
	errs = append(errs, boolFromEnv("ENABLE_APACHE_HTTPD_AUTO_INSTRUMENTATION", &cfg.EnableApacheHttpdInstrumentation))
	errs = append(errs, boolFromEnv("ENABLE_PHP_AUTO_INSTRUMENTATION", &cfg.EnablePhpAutoInstrumentation))
	errs = append(errs, boolFromEnv("ENABLE_PYTHON_AUTO_INSTRUMENTATION", &cfg.EnablePythonAutoInstrumentation))
	errs = append(errs, boolFromEnv("ENABLE_RUBY_AUTO_INSTRUMENTATION", &cfg.EnableRubyAutoInstrumentation))
	errs = append(errs, boolFromEnv("IGNORE_MISSING_COLLECTOR_CRDS", &cfg.IgnoreMissingCollectorCRDs))
	errs = append(errs, boolFromEnv("ENABLE_WEBHOOKS", &cfg.EnableWebhooks))
	if v, ok := os.LookupEnv("FEATURE_GATES"); ok {
		cfg.FeatureGates = v
	}
	errs = append(errs, boolFromEnv("ENABLE_MULTI_INSTRUMENTATION", &cfg.EnableMultiInstrumentation))
	if v, ok := os.LookupEnv("WATCH_NAMESPACE"); ok {
		cfg.WatchNamespace = v
	}
	errs = append(errs, int32FromEnv("OPENSHIFT_WEBHOOK_REPLICAS", &cfg.OpenShiftWebhookReplicas))
	cfg.ProxyEnvVars = proxy.ReadProxyVarsFromEnv()
	return errors.Join(errs...)
}

// boolFromEnv sets dst from the environment variable name, if it is set.
func boolFromEnv(name string, dst *bool) error {
	v, ok := os.LookupEnv(name)
	if !ok {
		return nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return envVarError(name, v, err)
	}
	*dst = b
	return nil
}

// intFromEnv sets dst from the environment variable name, if it is set.
func intFromEnv(name string, dst *int) error {
	v, ok := os.LookupEnv(name)
	if !ok {
		return nil
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return envVarError(name, v, err)
	}
	*dst = i
	return nil
}

// int32FromEnv sets dst from the environment variable name, if it is set.
func int32FromEnv(name string, dst *int32) error {
	v, ok := os.LookupEnv(name)
	if !ok {
		return nil
	}
	i, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return envVarError(name, v, err)
	}
	*dst = int32(i)
	return nil
}

// parseCurvePreferences parses a comma-separated list of numeric curve IDs, like
// the --tls-curve-preferences flag.
func parseCurvePreferences(v string) ([]int32, error) {
	var curves []int32
	for s := range strings.SplitSeq(v, ",") {
		i, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)
		if err != nil {
			return nil, err
		}
		curves = append(curves, int32(i))
	}
	return curves, nil
}

func envVarError(name, value string, err error) error {
	return fmt.Errorf("invalid value %q for %s: %w", value, name, err)
}
