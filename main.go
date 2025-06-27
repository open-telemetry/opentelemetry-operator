// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/tls"
	"errors"
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
	colfeaturegate "go.opentelemetry.io/collector/featuregate"
	"go.uber.org/zap/zapcore"
	networkingv1 "k8s.io/api/networking/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
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
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/opampbridge"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/targetallocator"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/controllers"
	"github.com/open-telemetry/opentelemetry-operator/internal/fips"
	"github.com/open-telemetry/opentelemetry-operator/internal/instrumentation"
	instrumentationupgrade "github.com/open-telemetry/opentelemetry-operator/internal/instrumentation/upgrade"
	collectorManifests "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	openshiftDashboards "github.com/open-telemetry/opentelemetry-operator/internal/openshift/dashboards"
	operatormetrics "github.com/open-telemetry/opentelemetry-operator/internal/operator-metrics"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"github.com/open-telemetry/opentelemetry-operator/internal/webhook/podmutation"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
	"github.com/open-telemetry/opentelemetry-operator/pkg/sidecar"
)

var (
	scheme   = k8sruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(otelv1alpha1.AddToScheme(scheme))
	utilruntime.Must(otelv1beta1.AddToScheme(scheme))
	utilruntime.Must(networkingv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	cfg := config.New()

	cliFlags := config.CreateCLIParser(cfg)

	// registers any flags that underlying libraries might use
	opts := zap.Options{}
	var zapFlagSet flag.FlagSet
	opts.BindFlags(&zapFlagSet)
	cliFlags.AddGoFlagSet(&zapFlagSet)

	featureGates := featuregate.Flags(colfeaturegate.GlobalRegistry())
	cliFlags.AddGoFlagSet(featureGates)

	var configFile string
	cliFlags.StringVar(&configFile, "config-file", "", "Path to config file")
	if err := cliFlags.Parse(os.Args[1:]); err != nil {
		panic(err)
	}

	opts.EncoderConfigOptions = append(opts.EncoderConfigOptions, func(ec *zapcore.EncoderConfig) {
		ec.MessageKey = cfg.Zap.MessageKey
		ec.LevelKey = cfg.Zap.LevelKey
		ec.TimeKey = cfg.Zap.TimeKey
		if cfg.Zap.LevelFormat == "lowercase" {
			ec.EncodeLevel = zapcore.LowercaseLevelEncoder
		} else {
			ec.EncodeLevel = zapcore.CapitalLevelEncoder
		}
	})

	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)

	configLog := ctrl.Log.WithName("config")

	err := cfg.Apply(configFile)
	if err != nil {
		configLog.Error(err, "configuration error")
		os.Exit(1)
	}

	v := version.Get()

	logger.Info("Starting the OpenTelemetry Operator",
		"opentelemetry-operator", v.Operator,
		"build-date", v.BuildDate,
		"go-version", v.Go,
		"go-arch", runtime.GOARCH,
		"go-os", runtime.GOOS,
		"feature-gates", featureGates.Lookup(featuregate.FeatureGatesFlag).Value.String(),
		"config", cfg.ToStringMap(),
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
		func(config *tls.Config) {
			if err = cfg.TLS.ApplyTLSConfig(config); err != nil {
				setupLog.Error(err, "error setting up TLS")
			}
		},
	}

	mgrOptions := ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: cfg.MetricsAddr,
		},
		HealthProbeBindAddress:        cfg.ProbeAddr,
		LeaderElection:                cfg.EnableLeaderElection,
		LeaderElectionID:              "9f7554c3.opentelemetry.io",
		LeaderElectionReleaseOnCancel: true,
		LeaseDuration:                 &leaseDuration,
		RenewDeadline:                 &renewDeadline,
		RetryPeriod:                   &retryPeriod,
		PprofBindAddress:              cfg.PprofAddr,
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    cfg.WebhookPort,
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

	if cfg.OpenshiftCreateDashboard {
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

	if err = autodetect.ApplyAutoDetect(ad, &cfg, configLog); err != nil {
		setupLog.Error(err, "failed to autodetect config variables")
	}
	// Only add these to the scheme if they are available
	if cfg.PrometheusCRAvailability == prometheus.Available {
		setupLog.Info("Prometheus CRDs are installed, adding to scheme.")
		utilruntime.Must(monitoringv1.AddToScheme(scheme))
	} else {
		setupLog.Info("Prometheus CRDs are not installed, skipping adding to scheme.")
	}
	if cfg.OpenShiftRoutesAvailability == openshift.RoutesAvailable {
		setupLog.Info("Openshift CRDs are installed, adding to scheme.")
		utilruntime.Must(routev1.Install(scheme))
	} else {
		setupLog.Info("Openshift CRDs are not installed, skipping adding to scheme.")
	}
	if cfg.CertManagerAvailability == certmanager.Available {
		setupLog.Info("Cert-Manager is available to the operator, adding to scheme.")
		utilruntime.Must(cmv1.AddToScheme(scheme))

		if featuregate.EnableTargetAllocatorMTLS.IsEnabled() {
			setupLog.Info("Securing the connection between the target allocator and the collector")
		}
	} else {
		setupLog.Info("Cert-Manager is not available to the operator, skipping adding to scheme.")
	}
	if cfg.CollectorAvailability == collector.Available {
		setupLog.Info("OpenTelemetryCollectorCRDSs are available to the operator")
	} else {
		setupLog.Info("OpenTelemetryCollectorCRDSs are not available to the operator")
		if !cfg.IgnoreMissingCollectorCRDs {
			setupLog.Error(errors.New("missing OpenTelemetryCollector CRDs"), "The OpenTelemetryCollector CRDs are not present in the cluster. Set ignore_missing_collector_crds to true or install the CRDs in the cluster.")
			os.Exit(1)
		}
	}
	if cfg.AnnotationsFilter != nil {
		for _, basePattern := range cfg.AnnotationsFilter {
			_, compileErr := regexp.Compile(basePattern)
			if compileErr != nil {
				setupLog.Error(compileErr, "could not compile the regexp pattern for Annotations filter")
			}
		}
	}

	if cfg.LabelsFilter != nil {
		for _, basePattern := range cfg.LabelsFilter {
			_, compileErr := regexp.Compile(basePattern)
			if compileErr != nil {
				setupLog.Error(compileErr, "could not compile the regexp pattern for Labels filter")
			}
		}
	}

	err = addDependencies(ctx, mgr, cfg)
	if err != nil {
		setupLog.Error(err, "failed to add/run bootstrap dependencies to the controller manager")
		os.Exit(1)
	}

	var collectorReconciler *controllers.OpenTelemetryCollectorReconciler
	if cfg.CollectorAvailability == collector.Available {
		collectorReconciler = controllers.NewReconciler(controllers.Params{
			Client:   mgr.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("OpenTelemetryCollector"),
			Scheme:   mgr.GetScheme(),
			Config:   cfg,
			Recorder: mgr.GetEventRecorderFor("opentelemetry-operator"),
			Reviewer: reviewer,
			Version:  v,
		})

		if err = collectorReconciler.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "OpenTelemetryCollector")
			os.Exit(1)
		}
	}

	if cfg.TargetAllocatorAvailability == targetallocator.Available {
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

	if cfg.OpAmpBridgeAvailability == opampbridge.Available {
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
	}

	if cfg.PrometheusCRAvailability == prometheus.Available && cfg.CreateServiceMonitorOperatorMetrics {
		operatorMetrics, opError := operatormetrics.NewOperatorMetrics(mgr.GetConfig(), scheme, ctrl.Log.WithName("operator-metrics-sm"))
		if opError != nil {
			setupLog.Error(opError, "Failed to create the operator metrics SM")
		}
		err = mgr.Add(operatorMetrics)
		if err != nil {
			setupLog.Error(err, "Failed to add the operator metrics SM")
		}
	}

	if cfg.EnableWebhooks {
		var crdMetrics *otelv1beta1.Metrics

		if cfg.EnableCRMetrics {
			meterProvider, metricsErr := otelv1beta1.BootstrapMetrics()
			if metricsErr != nil {
				setupLog.Error(metricsErr, "Error bootstrapping CRD metrics")
			}

			crdMetrics, err = otelv1beta1.NewMetrics(meterProvider, ctx, mgr.GetAPIReader())
			if err != nil {
				setupLog.Error(err, "Error init CRD metrics")
			}
		}

		if cfg.CollectorAvailability == collector.Available {
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
				receivers, exporters, processors, extensions := parseFipsFlag(cfg.FipsDisabledComponents)
				logger.Info("Fips disabled components", "receivers", receivers, "exporters", exporters, "processors", processors, "extensions", extensions)
				fipsCheck = fips.NewFipsCheck(receivers, exporters, processors, extensions)
			}
			if err = otelv1beta1.SetupCollectorWebhook(mgr, cfg, reviewer, crdMetrics, bv, fipsCheck); err != nil {
				setupLog.Error(err, "unable to create webhook", "webhook", "OpenTelemetryCollector")
				os.Exit(1)
			}
		}
		if cfg.TargetAllocatorAvailability == targetallocator.Available {
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

		if cfg.OpAmpBridgeAvailability == opampbridge.Available {
			if err = otelv1alpha1.SetupOpAMPBridgeWebhook(mgr, cfg); err != nil {
				setupLog.Error(err, "unable to create webhook", "webhook", "OpAMPBridge")
				os.Exit(1)
			}
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

func addDependencies(_ context.Context, mgr ctrl.Manager, cfg config.Config) error {
	// adds the upgrade mechanism to be executed once the manager is ready
	err := mgr.Add(manager.RunnableFunc(func(c context.Context) error {
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
