// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"

	"github.com/spf13/pflag"

	autoRBAC "github.com/open-telemetry/opentelemetry-operator/internal/autodetect/rbac"
)

var args = os.Args[1:]

func CreateCLIParser(cfg Config) *pflag.FlagSet {
	f := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	f.ParseErrorsWhitelist.UnknownFlags = true
	f.String("metrics-addr", cfg.MetricsAddr, "The address the metric endpoint binds to.")
	f.Bool("metrics-secure", cfg.MetricsSecure, "Enable secure serving for metrics endpoint with authentication and authorization. When enabled ano no TLS certificates are provided, the operator generates self signed certificates.")
	f.String("metrics-tls-cert-file", cfg.MetricsTLSCertFile, "TLS certificate file for the metrics server")
	f.String("metrics-tls-key-file", cfg.MetricsTLSKeyFile, "TLS private key file for the metrics server")
	f.String("health-probe-addr", cfg.ProbeAddr, "The address the probe endpoint binds to.")
	f.String("pprof-addr", cfg.PprofAddr, "The address to expose the pprof server. Default is empty string which disables the pprof server.")
	f.Bool("enable-leader-election", cfg.EnableLeaderElection,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	f.Bool("create-rbac-permissions", cfg.CreateRBACPermissions == autoRBAC.Available, "Automatically create RBAC permissions needed by the processors (deprecated)")
	f.Bool("openshift-create-dashboard", cfg.OpenshiftCreateDashboard, "Create an OpenShift dashboard for monitoring the OpenTelemetryCollector instances")
	f.Bool("enable-multi-instrumentation", cfg.EnableMultiInstrumentation, "Controls whether the operator supports multi instrumentation")
	f.Bool("enable-apache-httpd-instrumentation", cfg.EnableApacheHttpdInstrumentation, "Controls whether the operator supports Apache HTTPD auto-instrumentation")
	f.Bool("enable-dotnet-instrumentation", cfg.EnableDotNetAutoInstrumentation, "Controls whether the operator supports dotnet auto-instrumentation")
	f.Bool("enable-go-instrumentation", cfg.EnableGoAutoInstrumentation, "Controls whether the operator supports Go auto-instrumentation")
	f.Bool("enable-python-instrumentation", cfg.EnablePythonAutoInstrumentation, "Controls whether the operator supports python auto-instrumentation")
	f.Bool("enable-nginx-instrumentation", cfg.EnableNginxAutoInstrumentation, "Controls whether the operator supports nginx auto-instrumentation")
	f.Bool("enable-nodejs-instrumentation", cfg.EnableNodeJSAutoInstrumentation, "Controls whether the operator supports nodejs auto-instrumentation")
	f.Bool("enable-java-instrumentation", cfg.EnableJavaAutoInstrumentation, "Controls whether the operator supports java auto-instrumentation")
	f.Bool("enable-cr-metrics", cfg.EnableCRMetrics, "Controls whether exposing the CR metrics is enabled")
	f.Bool("create-sm-operator-metrics", cfg.CreateServiceMonitorOperatorMetrics, "Create a ServiceMonitor for the operator metrics")
	f.Bool("ignore-missing-collector-crds", cfg.IgnoreMissingCollectorCRDs, "Ignore missing OpenTelemetryCollector CRDs presence in the cluster")
	f.String("collector-image", cfg.CollectorImage, "The default OpenTelemetry collector image. This image is used when no image is specified in the CustomResource.")
	f.String("target-allocator-image", cfg.TargetAllocatorImage, "The default OpenTelemetry target allocator image. This image is used when no image is specified in the CustomResource.")
	f.String("operator-opamp-bridge-image", cfg.OperatorOpAMPBridgeImage, "The default OpenTelemetry Operator OpAMP Bridge image. This image is used when no image is specified in the CustomResource.")
	f.String("auto-instrumentation-java-image", cfg.AutoInstrumentationJavaImage, "The default OpenTelemetry Java instrumentation image. This image is used when no image is specified in the CustomResource.")
	f.String("auto-instrumentation-nodejs-image", cfg.AutoInstrumentationNodeJSImage, "The default OpenTelemetry NodeJS instrumentation image. This image is used when no image is specified in the CustomResource.")
	f.String("auto-instrumentation-python-image", cfg.AutoInstrumentationPythonImage, "The default OpenTelemetry Python instrumentation image. This image is used when no image is specified in the CustomResource.")
	f.String("auto-instrumentation-dotnet-image", cfg.AutoInstrumentationDotNetImage, "The default OpenTelemetry DotNet instrumentation image. This image is used when no image is specified in the CustomResource.")
	f.String("auto-instrumentation-go-image", cfg.AutoInstrumentationGoImage, "The default OpenTelemetry Go instrumentation image. This image is used when no image is specified in the CustomResource.")
	f.String("auto-instrumentation-apache-httpd-image", cfg.AutoInstrumentationApacheHttpdImage, "The default OpenTelemetry Apache HTTPD instrumentation image. This image is used when no image is specified in the CustomResource.")
	f.String("auto-instrumentation-nginx-image", cfg.AutoInstrumentationNginxImage, "The default OpenTelemetry Nginx instrumentation image. This image is used when no image is specified in the CustomResource.")
	f.StringArray("labels-filter", cfg.LabelsFilter, "Labels to filter away from propagating onto deploys. It should be a string array containing patterns, which are literal strings optionally containing a * wildcard character. Example: --labels-filter=.*filter.out will filter out labels that looks like: label.filter.out: true")
	f.StringArray("annotations-filter", cfg.AnnotationsFilter, "Annotations to filter away from propagating onto deploys. It should be a string array containing patterns, which are literal strings optionally containing a * wildcard character. Example: --annotations-filter=.*filter.out will filter out annotations that looks like: annotation.filter.out: true")
	f.String("fips-disabled-components", cfg.FipsDisabledComponents, "Disabled collector components when operator runs on FIPS enabled platform. Example flag value =receiver.foo,receiver.bar,exporter.baz")
	f.Int("webhook-port", cfg.WebhookPort, "The port the webhook endpoint binds to.")
	f.String("tls-min-version", "VersionTLS12", "Minimum TLS version supported. Value must match version names from https://golang.org/pkg/crypto/tls/#pkg-constants.")
	f.StringSlice("tls-cipher-suites", nil, "Comma-separated list of cipher suites for the server. Values are from tls package constants (https://golang.org/pkg/crypto/tls/#pkg-constants). If omitted, the default Go cipher suites will be used")
	f.String("zap-message-key", "message", "The message key to be used in the customized Log Encoder")
	f.String("zap-level-key", "level", "The level key to be used in the customized Log Encoder")
	f.String("zap-time-key", "timestamp", "The time key to be used in the customized Log Encoder")
	f.String("zap-level-format", "uppercase", "The level format to be used in the customized Log Encoder")
	f.Bool("enable-webhooks", cfg.EnableWebhooks, "Enable webhooks for the controllers")

	return f
}

func ApplyCLI(cfg *Config) error {

	f := CreateCLIParser(*cfg)
	err := f.Parse(args)
	if err != nil {
		return err
	}

	f.Visit(func(fl *pflag.Flag) {
		if fl.Changed {
			switch fl.Name {
			case "enable-multi-instrumentation":
				cfg.EnableMultiInstrumentation, _ = f.GetBool("enable-multi-instrumentation")
			case "enable-apache-httpd-instrumentation":
				cfg.EnableApacheHttpdInstrumentation, _ = f.GetBool("enable-apache-httpd-instrumentation")
			case "enable-dotnet-instrumentation":
				cfg.EnableDotNetAutoInstrumentation, _ = f.GetBool("enable-dotnet-instrumentation")
			case "enable-go-instrumentation":
				cfg.EnableGoAutoInstrumentation, _ = f.GetBool("enable-go-instrumentation")
			case "enable-python-instrumentation":
				cfg.EnablePythonAutoInstrumentation, _ = f.GetBool("enable-python-instrumentation")
			case "enable-nginx-instrumentation":
				cfg.EnableNginxAutoInstrumentation, _ = f.GetBool("enable-nginx-instrumentation")
			case "enable-nodejs-instrumentation":
				cfg.EnableNodeJSAutoInstrumentation, _ = f.GetBool("enable-nodejs-instrumentation")
			case "enable-java-instrumentation":
				cfg.EnableJavaAutoInstrumentation, _ = f.GetBool("enable-java-instrumentation")
			case "ignore-missing-collector-crds":
				cfg.IgnoreMissingCollectorCRDs, _ = f.GetBool("ignore-missing-collector-crds")
			case "collector-image":
				cfg.CollectorImage, _ = f.GetString("collector-image")
			case "target-allocator-image":
				cfg.TargetAllocatorImage, _ = f.GetString("target-allocator-image")
			case "operator-opamp-bridge-image":
				cfg.OperatorOpAMPBridgeImage, _ = f.GetString("operator-opamp-bridge-image")
			case "auto-instrumentation-java-image":
				cfg.AutoInstrumentationJavaImage, _ = f.GetString("auto-instrumentation-java-image")
			case "auto-instrumentation-nodejs-image":
				cfg.AutoInstrumentationNodeJSImage, _ = f.GetString("auto-instrumentation-nodejs-image")
			case "auto-instrumentation-python-image":
				cfg.AutoInstrumentationPythonImage, _ = f.GetString("auto-instrumentation-python-image")
			case "auto-instrumentation-dotnet-image":
				cfg.AutoInstrumentationDotNetImage, _ = f.GetString("auto-instrumentation-dotnet-image")
			case "auto-instrumentation-go-image":
				cfg.AutoInstrumentationGoImage, _ = f.GetString("auto-instrumentation-go-image")
			case "auto-instrumentation-apache-httpd-image":
				cfg.AutoInstrumentationApacheHttpdImage, _ = f.GetString("auto-instrumentation-apache-httpd-image")
			case "auto-instrumentation-nginx-image":
				cfg.AutoInstrumentationNginxImage, _ = f.GetString("auto-instrumentation-nginx-image")
			case "labels-filter":
				cfg.LabelsFilter, _ = f.GetStringSlice("labels-filter")
			case "annotations-filter":
				cfg.AnnotationsFilter, _ = f.GetStringSlice("annotations-filter")
			case "openshift-create-dashboard":
				cfg.OpenshiftCreateDashboard, _ = f.GetBool("openshift-create-dashboard")
			case "metrics-addr":
				cfg.MetricsAddr, _ = f.GetString("metrics-addr")
			case "metrics-secure":
				cfg.MetricsSecure, _ = f.GetBool("metrics-secure")
			case "metrics-tls-cert-file":
				cfg.MetricsTLSCertFile, _ = f.GetString("metrics-tls-cert-file")
			case "metrics-tls-key-file":
				cfg.MetricsTLSKeyFile, _ = f.GetString("metrics-tls-key-file")
			case "health-probe-addr":
				cfg.ProbeAddr, _ = f.GetString("health-probe-addr")
			case "pprof-addr":
				cfg.PprofAddr, _ = f.GetString("pprof-addr")
			case "enable-leader-election":
				cfg.EnableLeaderElection, _ = f.GetBool("enable-leader-election")
			case "enable-cr-metrics":
				cfg.EnableCRMetrics, _ = f.GetBool("enable-cr-metrics")
			case "create-sm-operator-metrics":
				cfg.CreateServiceMonitorOperatorMetrics, _ = f.GetBool("create-sm-operator-metrics")
			case "webhook-port":
				cfg.WebhookPort, _ = f.GetInt("webhook-port")
			case "fips-disabled-components":
				cfg.FipsDisabledComponents, _ = f.GetString("fips-disabled-components")
			case "min-tls-version":
				cfg.TLS.MinVersion, _ = f.GetString("min-tls-version")
			case "tls-cipher-suites":
				cfg.TLS.CipherSuites, _ = f.GetStringSlice("tls-cipher-suites")
			case "zap-message-key":
				cfg.Zap.MessageKey, _ = f.GetString("zap-message-key")
			case "zap-level-key":
				cfg.Zap.LevelKey, _ = f.GetString("zap-level-key")
			case "zap-time-key":
				cfg.Zap.TimeKey, _ = f.GetString("zap-time-key")
			case "zap-level-format":
				cfg.Zap.LevelFormat, _ = f.GetString("zap-level-format")
			case "enable-webhooks":
				cfg.EnableWebhooks, _ = f.GetBool("enable-webhooks")
			}
		}
	})

	return nil
}
