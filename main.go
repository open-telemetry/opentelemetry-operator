// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/spf13/pflag"
	colfeaturegate "go.opentelemetry.io/collector/featuregate"
	"go.uber.org/zap/zapcore"
	networkingv1 "k8s.io/api/networking/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	k8sapiflag "k8s.io/component-base/cli/flag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/controllers"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/targetallocator"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/fips"
	collectorManifests "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	openshiftDashboards "github.com/open-telemetry/opentelemetry-operator/internal/openshift/dashboards"
	operatormetrics "github.com/open-telemetry/opentelemetry-operator/internal/operator-metrics"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"github.com/open-telemetry/opentelemetry-operator/internal/webhook/podmutation"
	collectorupgrade "github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
	"github.com/open-telemetry/opentelemetry-operator/pkg/instrumentation"
	instrumentationupgrade "github.com/open-telemetry/opentelemetry-operator/pkg/instrumentation/upgrade"
	"github.com/open-telemetry/opentelemetry-operator/pkg/sidecar"
)

var (
	scheme   = k8sruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

type tlsConfig struct {
	minVersion   string
	cipherSuites []string
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(otelv1alpha1.AddToScheme(scheme))
	utilruntime.Must(otelv1beta1.AddToScheme(scheme))
	utilruntime.Must(networkingv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

// stringFlagOrEnv defines a string flag which can be set by an environment variable.
// Precedence: flag > env var > default value.
func stringFlagOrEnv(p *string, name string, envName string, defaultValue string, usage string) {
	envValue := os.Getenv(envName)
	if envValue != "" {
		defaultValue = envValue
	}
	pflag.StringVar(p, name, defaultValue, usage)
}

func main() {
	// registers any flags that underlying libraries might use
	opts := zap.Options{}
	flagset := featuregate.Flags(colfeaturegate.GlobalRegistry())
	opts.BindFlags(flag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flagset)

	v := version.Get()

	// add flags related to this operator
	var (
		metricsAddr                      string
		probeAddr                        string
		pprofAddr                        string
		enableLeaderElection             bool
		createRBACPermissions            bool
		createOpenShiftDashboard         bool
		enableMultiInstrumentation       bool
		enableApacheHttpdInstrumentation bool
		enableDotNetInstrumentation      bool
		enableGoInstrumentation          bool
		enablePythonInstrumentation      bool
		enableNginxInstrumentation       bool
		enableNodeJSInstrumentation      bool
		enableJavaInstrumentation        bool
		enableCRMetrics                  bool
		createSMOperatorMetrics          bool
		collectorImage                   string
		targetAllocatorImage             string
		operatorOpAMPBridgeImage         string
		autoInstrumentationJava          string
		autoInstrumentationNodeJS        string
		autoInstrumentationPython        string
		autoInstrumentationDotNet        string
		autoInstrumentationApacheHttpd   string
		autoInstrumentationNginx         string
		autoInstrumentationGo            string
		labelsFilter                     []string
		tmplabelsFilter                  []string
		annotationsFilter                []string
		webhookPort                      int
		tlsOpt                           tlsConfig
		encodeMessageKey                 string
		encodeLevelKey                   string
		encodeTimeKey                    string
		encodeLevelFormat                string
		fipsDisabledComponents           string
	)

	pflag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	pflag.StringVar(&probeAddr, "health-probe-addr", ":8081", "The address the probe endpoint binds to.")
	pflag.StringVar(&pprofAddr, "pprof-addr", "", "The address to expose the pprof server. Default is empty string which disables the pprof server.")
	pflag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	pflag.BoolVar(&createRBACPermissions, "create-rbac-permissions", false, "Automatically create RBAC permissions needed by the processors (deprecated)")
	pflag.BoolVar(&createOpenShiftDashboard, "openshift-create-dashboard", false, "Create an OpenShift dashboard for monitoring the OpenTelemetryCollector instances")
	pflag.BoolVar(&enableMultiInstrumentation, "enable-multi-instrumentation", true, "Controls whether the operator supports multi instrumentation")
	pflag.BoolVar(&enableApacheHttpdInstrumentation, constants.FlagApacheHttpd, true, "Controls whether the operator supports Apache HTTPD auto-instrumentation")
	pflag.BoolVar(&enableDotNetInstrumentation, constants.FlagDotNet, true, "Controls whether the operator supports dotnet auto-instrumentation")
	pflag.BoolVar(&enableGoInstrumentation, constants.FlagGo, false, "Controls whether the operator supports Go auto-instrumentation")
	pflag.BoolVar(&enablePythonInstrumentation, constants.FlagPython, true, "Controls whether the operator supports python auto-instrumentation")
	pflag.BoolVar(&enableNginxInstrumentation, constants.FlagNginx, false, "Controls whether the operator supports nginx auto-instrumentation")
	pflag.BoolVar(&enableNodeJSInstrumentation, constants.FlagNodeJS, true, "Controls whether the operator supports nodejs auto-instrumentation")
	pflag.BoolVar(&enableJavaInstrumentation, constants.FlagJava, true, "Controls whether the operator supports java auto-instrumentation")
	pflag.BoolVar(&enableCRMetrics, constants.FlagCRMetrics, false, "Controls whether exposing the CR metrics is enabled")
	pflag.BoolVar(&createSMOperatorMetrics, "create-sm-operator-metrics", false, "Create a ServiceMonitor for the operator metrics")

	stringFlagOrEnv(&collectorImage, "collector-image", "RELATED_IMAGE_COLLECTOR", fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector:%s", v.OpenTelemetryCollector), "The default OpenTelemetry collector image. This image is used when no image is specified in the CustomResource.")
	stringFlagOrEnv(&targetAllocatorImage, "target-allocator-image", "RELATED_IMAGE_TARGET_ALLOCATOR", fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/target-allocator:%s", v.TargetAllocator), "The default OpenTelemetry target allocator image. This image is used when no image is specified in the CustomResource.")
	stringFlagOrEnv(&operatorOpAMPBridgeImage, "operator-opamp-bridge-image", "RELATED_IMAGE_OPERATOR_OPAMP_BRIDGE", fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/operator-opamp-bridge:%s", v.OperatorOpAMPBridge), "The default OpenTelemetry Operator OpAMP Bridge image. This image is used when no image is specified in the CustomResource.")
	stringFlagOrEnv(&autoInstrumentationJava, "auto-instrumentation-java-image", "RELATED_IMAGE_AUTO_INSTRUMENTATION_JAVA", fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:%s", v.AutoInstrumentationJava), "The default OpenTelemetry Java instrumentation image. This image is used when no image is specified in the CustomResource.")
	stringFlagOrEnv(&autoInstrumentationNodeJS, "auto-instrumentation-nodejs-image", "RELATED_IMAGE_AUTO_INSTRUMENTATION_NODEJS", fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-nodejs:%s", v.AutoInstrumentationNodeJS), "The default OpenTelemetry NodeJS instrumentation image. This image is used when no image is specified in the CustomResource.")
	stringFlagOrEnv(&autoInstrumentationPython, "auto-instrumentation-python-image", "RELATED_IMAGE_AUTO_INSTRUMENTATION_PYTHON", fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:%s", v.AutoInstrumentationPython), "The default OpenTelemetry Python instrumentation image. This image is used when no image is specified in the CustomResource.")
	stringFlagOrEnv(&autoInstrumentationDotNet, "auto-instrumentation-dotnet-image", "RELATED_IMAGE_AUTO_INSTRUMENTATION_DOTNET", fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-dotnet:%s", v.AutoInstrumentationDotNet), "The default OpenTelemetry DotNet instrumentation image. This image is used when no image is specified in the CustomResource.")
	stringFlagOrEnv(&autoInstrumentationGo, "auto-instrumentation-go-image", "RELATED_IMAGE_AUTO_INSTRUMENTATION_GO", fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-go-instrumentation/autoinstrumentation-go:%s", v.AutoInstrumentationGo), "The default OpenTelemetry Go instrumentation image. This image is used when no image is specified in the CustomResource.")
	stringFlagOrEnv(&autoInstrumentationApacheHttpd, "auto-instrumentation-apache-httpd-image", "RELATED_IMAGE_AUTO_INSTRUMENTATION_APACHE_HTTPD", fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-apache-httpd:%s", v.AutoInstrumentationApacheHttpd), "The default OpenTelemetry Apache HTTPD instrumentation image. This image is used when no image is specified in the CustomResource.")
	stringFlagOrEnv(&autoInstrumentationNginx, "auto-instrumentation-nginx-image", "RELATED_IMAGE_AUTO_INSTRUMENTATION_NGINX", fmt.Sprintf("ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-apache-httpd:%s", v.AutoInstrumentationNginx), "The default OpenTelemetry Nginx instrumentation image. This image is used when no image is specified in the CustomResource.")
	pflag.StringArrayVar(&labelsFilter, "labels-filter", []string{}, "Labels to filter away from propagating onto deploys. It should be a string array containing patterns, which are literal strings optionally containing a * wildcard character. Example: --labels-filter=.*filter.out will filter out labels that looks like: label.filter.out: true")
	pflag.StringArrayVar(&tmplabelsFilter, "label", []string{}, "(Deprecated, please use the labels-filter flag) Labels to filter away from propagating onto deploys. It should be a string array containing patterns, which are literal strings optionally containing a * wildcard character. Example: --label=.*filter.out will filter out labels that looks like: label.filter.out: true")
	pflag.StringArrayVar(&annotationsFilter, "annotations-filter", []string{}, "Annotations to filter away from propagating onto deploys. It should be a string array containing patterns, which are literal strings optionally containing a * wildcard character. Example: --annotations-filter=.*filter.out will filter out annotations that looks like: annotation.filter.out: true")
	pflag.StringVar(&tlsOpt.minVersion, "tls-min-version", "VersionTLS12", "Minimum TLS version supported. Value must match version names from https://golang.org/pkg/crypto/tls/#pkg-constants.")
	pflag.StringSliceVar(&tlsOpt.cipherSuites, "tls-cipher-suites", nil, "Comma-separated list of cipher suites for the server. Values are from tls package constants (https://golang.org/pkg/crypto/tls/#pkg-constants). If omitted, the default Go cipher suites will be used")
	pflag.StringVar(&encodeMessageKey, "zap-message-key", "message", "The message key to be used in the customized Log Encoder")
	pflag.StringVar(&encodeLevelKey, "zap-level-key", "level", "The level key to be used in the customized Log Encoder")
	pflag.StringVar(&encodeTimeKey, "zap-time-key", "timestamp", "The time key to be used in the customized Log Encoder")
	pflag.StringVar(&encodeLevelFormat, "zap-level-format", "uppercase", "The level format to be used in the customized Log Encoder")
	pflag.StringVar(&fipsDisabledComponents, "fips-disabled-components", "uppercase", "Disabled collector components when operator runs on FIPS enabled platform. Example flag value =receiver.foo,receiver.bar,exporter.baz")
	pflag.IntVar(&webhookPort, "webhook-port", 9443, "The port the webhook endpoint binds to.")
	pflag.Parse()

	// Using labelfilters both from label and labels-filter flags, until label flag is removed
	labelsFilter = append(labelsFilter, tmplabelsFilter...)

	opts.EncoderConfigOptions = append(opts.EncoderConfigOptions, func(ec *zapcore.EncoderConfig) {
		ec.MessageKey = encodeMessageKey
		ec.LevelKey = encodeLevelKey
		ec.TimeKey = encodeTimeKey
		ec.EncodeLevel = config.WithEncodeLevelFormat(encodeLevelFormat)
	})

	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)

	logger.Info("Starting the OpenTelemetry Operator",
		"opentelemetry-operator", v.Operator,
		"opentelemetry-collector", collectorImage,
		"opentelemetry-targetallocator", targetAllocatorImage,
		"operator-opamp-bridge", operatorOpAMPBridgeImage,
		"auto-instrumentation-java", autoInstrumentationJava,
		"auto-instrumentation-nodejs", autoInstrumentationNodeJS,
		"auto-instrumentation-python", autoInstrumentationPython,
		"auto-instrumentation-dotnet", autoInstrumentationDotNet,
		"auto-instrumentation-go", autoInstrumentationGo,
		"auto-instrumentation-apache-httpd", autoInstrumentationApacheHttpd,
		"auto-instrumentation-nginx", autoInstrumentationNginx,
		"feature-gates", flagset.Lookup(featuregate.FeatureGatesFlag).Value.String(),
		"build-date", v.BuildDate,
		"go-version", v.Go,
		"go-arch", runtime.GOARCH,
		"go-os", runtime.GOOS,
		"labels-filter", labelsFilter,
		"annotations-filter", annotationsFilter,
		"enable-multi-instrumentation", enableMultiInstrumentation,
		"enable-apache-httpd-instrumentation", enableApacheHttpdInstrumentation,
		"enable-dotnet-instrumentation", enableDotNetInstrumentation,
		"enable-go-instrumentation", enableGoInstrumentation,
		"enable-python-instrumentation", enablePythonInstrumentation,
		"enable-nginx-instrumentation", enableNginxInstrumentation,
		"enable-nodejs-instrumentation", enableNodeJSInstrumentation,
		"enable-java-instrumentation", enableJavaInstrumentation,
		"create-openshift-dashboard", createOpenShiftDashboard,
		"create-sm-operator-metrics", createSMOperatorMetrics,
		"zap-message-key", encodeMessageKey,
		"zap-level-key", encodeLevelKey,
		"zap-time-key", encodeTimeKey,
		"zap-level-format", encodeLevelFormat,
	)

	restConfig := ctrl.GetConfigOrDie()

	var namespaces map[string]cache.Config
	watchNamespace, found := os.LookupEnv("WATCH_NAMESPACE")
	if found {
		setupLog.Info("watching namespace(s)", "namespaces", watchNamespace)
		namespaces = map[string]cache.Config{}
		for _, ns := range strings.Split(watchNamespace, ",") {
			namespaces[ns] = cache.Config{}
		}
	} else {
		setupLog.Info("the env var WATCH_NAMESPACE isn't set, watching all namespaces")
	}

	// see https://github.com/openshift/library-go/blob/4362aa519714a4b62b00ab8318197ba2bba51cb7/pkg/config/leaderelection/leaderelection.go#L104
	leaseDuration := time.Second * 137
	renewDeadline := time.Second * 107
	retryPeriod := time.Second * 26

	optionsTlSOptsFuncs := []func(*tls.Config){
		func(config *tls.Config) { tlsConfigSetting(config, tlsOpt) },
	}

	mgrOptions := ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress:        probeAddr,
		LeaderElection:                enableLeaderElection,
		LeaderElectionID:              "9f7554c3.opentelemetry.io",
		LeaderElectionReleaseOnCancel: true,
		LeaseDuration:                 &leaseDuration,
		RenewDeadline:                 &renewDeadline,
		RetryPeriod:                   &retryPeriod,
		PprofBindAddress:              pprofAddr,
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    webhookPort,
			TLSOpts: optionsTlSOptsFuncs,
		}),
		Cache: cache.Options{
			DefaultNamespaces: namespaces,
		},
	}

	mgr, err := ctrl.NewManager(restConfig, mgrOptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "failed to create kubernetes clientset")
	}

	ctx := ctrl.SetupSignalHandler()

	if createOpenShiftDashboard {
		dashErr := mgr.Add(openshiftDashboards.NewDashboardManagement(clientset))
		if dashErr != nil {
			setupLog.Error(dashErr, "failed to create the OpenShift dashboards")
		}
	}

	reviewer := rbac.NewReviewer(clientset)

	// builds the operator's configuration
	ad, err := autodetect.New(restConfig, reviewer)
	if err != nil {
		setupLog.Error(err, "failed to setup auto-detect routine")
		os.Exit(1)
	}

	cfg := config.New(
		config.WithLogger(ctrl.Log.WithName("config")),
		config.WithVersion(v),
		config.WithCollectorImage(collectorImage),
		config.WithEnableMultiInstrumentation(enableMultiInstrumentation),
		config.WithEnableApacheHttpdInstrumentation(enableApacheHttpdInstrumentation),
		config.WithEnableDotNetInstrumentation(enableDotNetInstrumentation),
		config.WithEnableGoInstrumentation(enableGoInstrumentation),
		config.WithEnableNginxInstrumentation(enableNginxInstrumentation),
		config.WithEnablePythonInstrumentation(enablePythonInstrumentation),
		config.WithEnableNodeJSInstrumentation(enableNodeJSInstrumentation),
		config.WithEnableJavaInstrumentation(enableJavaInstrumentation),
		config.WithTargetAllocatorImage(targetAllocatorImage),
		config.WithOperatorOpAMPBridgeImage(operatorOpAMPBridgeImage),
		config.WithAutoInstrumentationJavaImage(autoInstrumentationJava),
		config.WithAutoInstrumentationNodeJSImage(autoInstrumentationNodeJS),
		config.WithAutoInstrumentationPythonImage(autoInstrumentationPython),
		config.WithAutoInstrumentationDotNetImage(autoInstrumentationDotNet),
		config.WithAutoInstrumentationGoImage(autoInstrumentationGo),
		config.WithAutoInstrumentationApacheHttpdImage(autoInstrumentationApacheHttpd),
		config.WithAutoInstrumentationNginxImage(autoInstrumentationNginx),
		config.WithAutoDetect(ad),
		config.WithLabelFilters(labelsFilter),
		config.WithAnnotationFilters(annotationsFilter),
	)
	err = cfg.AutoDetect()
	if err != nil {
		setupLog.Error(err, "failed to autodetect config variables")
	}
	// Only add these to the scheme if they are available
	if cfg.PrometheusCRAvailability() == prometheus.Available {
		setupLog.Info("Prometheus CRDs are installed, adding to scheme.")
		utilruntime.Must(monitoringv1.AddToScheme(scheme))
	} else {
		setupLog.Info("Prometheus CRDs are not installed, skipping adding to scheme.")
	}
	if cfg.OpenShiftRoutesAvailability() == openshift.RoutesAvailable {
		setupLog.Info("Openshift CRDs are installed, adding to scheme.")
		utilruntime.Must(routev1.Install(scheme))
	} else {
		setupLog.Info("Openshift CRDs are not installed, skipping adding to scheme.")
	}
	if cfg.CertManagerAvailability() == certmanager.Available {
		setupLog.Info("Cert-Manager is available to the operator, adding to scheme.")
		utilruntime.Must(cmv1.AddToScheme(scheme))

		if featuregate.EnableTargetAllocatorMTLS.IsEnabled() {
			setupLog.Info("Securing the connection between the target allocator and the collector")
		}
	} else {
		setupLog.Info("Cert-Manager is not available to the operator, skipping adding to scheme.")
	}
	if cfg.AnnotationsFilter() != nil {
		for _, basePattern := range cfg.AnnotationsFilter() {
			_, compileErr := regexp.Compile(basePattern)
			if compileErr != nil {
				setupLog.Error(compileErr, "could not compile the regexp pattern for Annotations filter")
			}
		}
	}

	if cfg.LabelsFilter() != nil {
		for _, basePattern := range cfg.LabelsFilter() {
			_, compileErr := regexp.Compile(basePattern)
			if compileErr != nil {
				setupLog.Error(compileErr, "could not compile the regexp pattern for Labels filter")
			}
		}
	}

	err = addDependencies(ctx, mgr, cfg, v)
	if err != nil {
		setupLog.Error(err, "failed to add/run bootstrap dependencies to the controller manager")
		os.Exit(1)
	}

	collectorReconciler := controllers.NewReconciler(controllers.Params{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("OpenTelemetryCollector"),
		Scheme:   mgr.GetScheme(),
		Config:   cfg,
		Recorder: mgr.GetEventRecorderFor("opentelemetry-operator"),
		Reviewer: reviewer,
	})

	if err = collectorReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OpenTelemetryCollector")
		os.Exit(1)
	}

	if cfg.TargetAllocatorAvailability() == targetallocator.Available {
		if err = controllers.NewTargetAllocatorReconciler(
			mgr.GetClient(),
			mgr.GetScheme(),
			mgr.GetEventRecorderFor("targetallocator"),
			cfg,
			ctrl.Log.WithName("controllers").WithName("TargetAllocator"),
		).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "TargetAllocator")
			os.Exit(1)
		}
	}

	if err = controllers.NewOpAMPBridgeReconciler(controllers.OpAMPBridgeReconcilerParams{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("OpAMPBridge"),
		Scheme:   mgr.GetScheme(),
		Config:   cfg,
		Recorder: mgr.GetEventRecorderFor("opamp-bridge"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OpAMPBridge")
		os.Exit(1)
	}

	if cfg.PrometheusCRAvailability() == prometheus.Available && createSMOperatorMetrics {
		operatorMetrics, opError := operatormetrics.NewOperatorMetrics(mgr.GetConfig(), scheme, ctrl.Log.WithName("operator-metrics-sm"))
		if opError != nil {
			setupLog.Error(opError, "Failed to create the operator metrics SM")
		}
		err = mgr.Add(operatorMetrics)
		if err != nil {
			setupLog.Error(err, "Failed to add the operator metrics SM")
		}
	}

	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		var crdMetrics *otelv1beta1.Metrics

		if enableCRMetrics {
			meterProvider, metricsErr := otelv1beta1.BootstrapMetrics()
			if metricsErr != nil {
				setupLog.Error(metricsErr, "Error bootstrapping CRD metrics")
			}

			crdMetrics, err = otelv1beta1.NewMetrics(meterProvider, ctx, mgr.GetAPIReader())
			if err != nil {
				setupLog.Error(err, "Error init CRD metrics")
			}
		}

		bv := func(ctx context.Context, collector otelv1beta1.OpenTelemetryCollector) admission.Warnings {
			var warnings admission.Warnings
			params, newErr := collectorReconciler.GetParams(ctx, collector)
			if err != nil {
				warnings = append(warnings, newErr.Error())
				return warnings
			}

			params.ErrorAsWarning = true
			_, newErr = collectorManifests.Build(params)
			if newErr != nil {
				warnings = append(warnings, newErr.Error())
				return warnings
			}
			return warnings
		}

		var fipsCheck fips.FIPSCheck
		if ad.FIPSEnabled(ctx) {
			receivers, exporters, processors, extensions := parseFipsFlag(fipsDisabledComponents)
			logger.Info("Fips disabled components", "receivers", receivers, "exporters", exporters, "processors", processors, "extensions", extensions)
			fipsCheck = fips.NewFipsCheck(receivers, exporters, processors, extensions)
		}
		if err = otelv1beta1.SetupCollectorWebhook(mgr, cfg, reviewer, crdMetrics, bv, fipsCheck); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OpenTelemetryCollector")
			os.Exit(1)
		}
		if cfg.TargetAllocatorAvailability() == targetallocator.Available {
			if err = otelv1alpha1.SetupTargetAllocatorWebhook(mgr, cfg, reviewer); err != nil {
				setupLog.Error(err, "unable to create webhook", "webhook", "TargetAllocator")
				os.Exit(1)
			}
		}
		if err = otelv1alpha1.SetupInstrumentationWebhook(mgr, cfg); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Instrumentation")
			os.Exit(1)
		}
		decoder := admission.NewDecoder(mgr.GetScheme())
		mgr.GetWebhookServer().Register("/mutate-v1-pod", &webhook.Admission{
			Handler: podmutation.NewWebhookHandler(cfg, ctrl.Log.WithName("pod-webhook"), decoder, mgr.GetClient(),
				[]podmutation.PodMutator{
					sidecar.NewMutator(logger, cfg, mgr.GetClient()),
					instrumentation.NewMutator(logger, mgr.GetClient(), mgr.GetEventRecorderFor("opentelemetry-operator"), cfg),
				}),
		})

		if err = otelv1alpha1.SetupOpAMPBridgeWebhook(mgr, cfg); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "OpAMPBridge")
			os.Exit(1)
		}
	} else {
		ctrl.Log.Info("Webhooks are disabled, operator is running an unsupported mode", "ENABLE_WEBHOOKS", "false")
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	// NOTE: We enable LeaderElectionReleaseOnCancel, and to be safe we need to exit right after the manager does
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func addDependencies(_ context.Context, mgr ctrl.Manager, cfg config.Config, v version.Version) error {
	// adds the upgrade mechanism to be executed once the manager is ready
	err := mgr.Add(manager.RunnableFunc(func(c context.Context) error {
		up := &collectorupgrade.VersionUpgrade{
			Log:      ctrl.Log.WithName("collector-upgrade"),
			Version:  v,
			Client:   mgr.GetClient(),
			Recorder: mgr.GetEventRecorderFor("opentelemetry-operator"),
		}
		return up.ManagedInstances(c)
	}))
	if err != nil {
		return fmt.Errorf("failed to upgrade OpenTelemetryCollector instances: %w", err)
	}

	// adds the upgrade mechanism to be executed once the manager is ready
	err = mgr.Add(manager.RunnableFunc(func(c context.Context) error {
		u := instrumentationupgrade.NewInstrumentationUpgrade(
			mgr.GetClient(),
			ctrl.Log.WithName("instrumentation-upgrade"),
			mgr.GetEventRecorderFor("opentelemetry-operator"),
			cfg,
		)
		return u.ManagedInstances(c)
	}))
	if err != nil {
		return fmt.Errorf("failed to upgrade Instrumentation instances: %w", err)
	}
	return nil
}

// This function get the option from command argument (tlsConfig), check the validity through k8sapiflag
// and set the config for webhook server.
// refer to https://pkg.go.dev/k8s.io/component-base/cli/flag
func tlsConfigSetting(cfg *tls.Config, tlsOpt tlsConfig) {
	// TLSVersion helper function returns the TLS Version ID for the version name passed.
	tlsVersion, err := k8sapiflag.TLSVersion(tlsOpt.minVersion)
	if err != nil {
		setupLog.Error(err, "TLS version invalid")
	}
	cfg.MinVersion = tlsVersion

	// TLSCipherSuites helper function returns a list of cipher suite IDs from the cipher suite names passed.
	cipherSuiteIDs, err := k8sapiflag.TLSCipherSuites(tlsOpt.cipherSuites)
	if err != nil {
		setupLog.Error(err, "Failed to convert TLS cipher suite name to ID")
	}
	cfg.CipherSuites = cipherSuiteIDs
}

func parseFipsFlag(fipsFlag string) ([]string, []string, []string, []string) {
	split := strings.Split(fipsFlag, ",")
	var receivers []string
	var exporters []string
	var processors []string
	var extensions []string
	for _, val := range split {
		val = strings.TrimSpace(val)
		typeAndName := strings.Split(val, ".")
		if len(typeAndName) == 2 {
			componentType := typeAndName[0]
			name := typeAndName[1]

			switch componentType {
			case "receiver":
				receivers = append(receivers, name)
			case "exporter":
				exporters = append(exporters, name)
			case "processor":
				processors = append(processors, name)
			case "extension":
				extensions = append(extensions, name)
			}
		}
	}
	return receivers, exporters, processors, extensions
}
