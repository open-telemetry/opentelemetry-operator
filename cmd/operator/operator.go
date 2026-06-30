// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/spf13/cobra"
	colfeaturegate "go.opentelemetry.io/collector/featuregate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
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
	"github.com/open-telemetry/opentelemetry-operator/internal/metrics"
	openshiftDashboards "github.com/open-telemetry/opentelemetry-operator/internal/openshift/dashboards"
	operatorsetup "github.com/open-telemetry/opentelemetry-operator/internal/operator"
	operatormetrics "github.com/open-telemetry/opentelemetry-operator/internal/operator-metrics"
	"github.com/open-telemetry/opentelemetry-operator/internal/operatornetworkpolicy"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	wh "github.com/open-telemetry/opentelemetry-operator/internal/webhook"
	"github.com/open-telemetry/opentelemetry-operator/internal/webhook/podmutation"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
	"github.com/open-telemetry/opentelemetry-operator/pkg/sidecar"
)

var setupLog = ctrl.Log.WithName("setup")

// NewOperatorCmd creates the root cobra command for the operator.
func NewOperatorCmd(scheme *k8sruntime.Scheme) *cobra.Command {
	cfg := config.New()
	cliFlags := config.CreateCLIParser(cfg)

	opts := zap.Options{}
	var zapFlagSet flag.FlagSet
	opts.BindFlags(&zapFlagSet)
	cliFlags.AddGoFlagSet(&zapFlagSet)

	featureGates := featuregate.Flags(colfeaturegate.GlobalRegistry())
	cliFlags.AddGoFlagSet(featureGates)

	var configFile string
	cliFlags.StringVar(&configFile, "config-file", "", "Path to config file")

	cmd := &cobra.Command{
		Use:   "operator",
		Short: "OpenTelemetry Operator",
		Long:  "OpenTelemetry Operator manages OpenTelemetry Collectors, auto-instrumentation, and the Target Allocator",
		Run: func(_ *cobra.Command, _ []string) {
			runOperator(cfg, configFile, opts, featureGates, scheme)
		},
	}

	cmd.Flags().AddFlagSet(cliFlags)
	return cmd
}

func runOperator(cfg config.Config, configFile string, opts zap.Options, featureGates *flag.FlagSet, scheme *k8sruntime.Scheme) {
	// Pass true to use the EnableLeaderElection setting from config (after config is applied)
	result := operatorsetup.SetupManager(&cfg, configFile, opts, scheme, true, "Starting the OpenTelemetry Operator")

	logger := ctrl.Log
	logger.Info("Feature gates", "feature-gates", featureGates.Lookup(featuregate.FeatureGatesFlag).Value.String())

	if err := discoverKubeAPIServer(context.Background(), result.Clientset, &result.Config); err != nil {
		setupLog.Info("Failed to discover Kubernetes API server from EndpointSlice", "error", err)
	}

	signalCtx := ctrl.SetupSignalHandler()
	ctx, cancel := context.WithCancel(signalCtx)
	defer cancel()

	if result.Config.TLS.UseClusterProfile {
		operatorsetup.SetupTLSProfileWatcher(result.Manager, result.InitialTLSProfile, cancel)
	}

	configLog := ctrl.Log.WithName("config")
	if result.Config.FeatureGates != "" {
		configLog.Info("Applying feature gates from configuration", "gates", result.Config.FeatureGates)
		if err := featuregate.ApplyFeatureGateOverrides(result.Config.FeatureGates); err != nil {
			setupLog.Error(err, "failed to apply feature gate overrides")
			os.Exit(1)
		}
	}

	mgr := result.Manager
	clientset := result.Clientset

	if result.Config.OpenshiftCreateDashboard {
		dashErr := mgr.Add(openshiftDashboards.NewDashboardManagement(clientset))
		if dashErr != nil {
			setupLog.Error(dashErr, "failed to create the OpenShift dashboards")
		}
	}
	if featuregate.EnableOperatorNetworkPolicy.IsEnabled() {
		errNetworkPolicy := enableOperatorNetworkPolicy(result.Config, clientset, mgr)
		if errNetworkPolicy != nil {
			setupLog.Error(errNetworkPolicy, "failed to create the Operator network policies")
			os.Exit(1)
		}
	}

	if result.Config.PrometheusCRAvailability == prometheus.Available {
		setupLog.Info("Prometheus CRDs are installed, adding to scheme.")
		utilruntime.Must(monitoringv1.AddToScheme(scheme))
	} else {
		setupLog.Info("Prometheus CRDs are not installed, skipping adding to scheme.")
	}
	if result.Config.OpenShiftRoutesAvailability == openshift.RoutesAvailable {
		setupLog.Info("Openshift CRDs are installed, adding to scheme.")
		utilruntime.Must(routev1.Install(scheme))
	} else {
		setupLog.Info("Openshift CRDs are not installed, skipping adding to scheme.")
	}
	if result.Config.CertManagerAvailability == certmanager.Available {
		setupLog.Info("Cert-Manager is available to the operator, adding to scheme.")
		utilruntime.Must(cmv1.AddToScheme(scheme))
	} else {
		setupLog.Info("Cert-Manager is not available to the operator, skipping adding to scheme.")
	}
	if result.Config.CollectorAvailability == collector.Available {
		setupLog.Info("OpenTelemetryCollectorCRDSs are available to the operator")
	} else {
		setupLog.Info("OpenTelemetryCollectorCRDSs are not available to the operator")
		if !result.Config.IgnoreMissingCollectorCRDs {
			setupLog.Error(errors.New("missing OpenTelemetryCollector CRDs"), "The OpenTelemetryCollector CRDs are not present in the cluster. Set ignore_missing_collector_crds to true or install the CRDs in the cluster.")
			os.Exit(1)
		}
	}

	v := version.Get()

	if result.Config.EnableInstrumentationCRDs {
		err := addInstrumentationUpgrader(ctx, mgr, result.Config)
		if err != nil {
			setupLog.Error(err, "failed to add/run bootstrap dependencies to the controller manager")
			os.Exit(1)
		}
	}

	var collectorReconciler *controllers.OpenTelemetryCollectorReconciler
	if result.Config.CollectorAvailability == collector.Available {
		collectorReconciler = controllers.NewReconciler(controllers.Params{
			Client:   mgr.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("OpenTelemetryCollector"),
			Scheme:   mgr.GetScheme(),
			Config:   result.Config,
			Recorder: mgr.GetEventRecorder("opentelemetry-operator"),
			Reviewer: result.Reviewer,
			Version:  v,
		})

		if err := collectorReconciler.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "OpenTelemetryCollector")
			os.Exit(1)
		}
	}

	if result.Config.TargetAllocatorAvailability == targetallocator.Available {
		if err := controllers.NewTargetAllocatorReconciler(
			mgr.GetClient(),
			mgr.GetScheme(),
			mgr.GetEventRecorder("targetallocator"),
			result.Config,
			ctrl.Log.WithName("controllers").WithName("TargetAllocator"),
		).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "TargetAllocator")
			os.Exit(1)
		}
	}

	if result.Config.OpAmpBridgeAvailability == opampbridge.Available {
		if err := controllers.NewOpAMPBridgeReconciler(controllers.OpAMPBridgeReconcilerParams{
			Client:   mgr.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("OpAMPBridge"),
			Scheme:   mgr.GetScheme(),
			Config:   result.Config,
			Recorder: mgr.GetEventRecorder("opamp-bridge"),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "OpAMPBridge")
			os.Exit(1)
		}
	}

	if featuregate.EnableClusterObservability.IsEnabled() {
		setupLog.Info("ClusterObservability feature is enabled")
		if err := controllers.NewClusterObservabilityReconciler(controllers.ClusterObservabilityReconcilerParams{
			Client:   mgr.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("ClusterObservability"),
			Scheme:   mgr.GetScheme(),
			Config:   result.Config,
			Recorder: mgr.GetEventRecorder("cluster-observability"),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ClusterObservability")
			os.Exit(1)
		}
	} else {
		setupLog.Info("ClusterObservability feature is disabled")
	}

	// Setup pod-webhook replica controller to maintain desired replica count.
	// On OpenShift with OLM, this ensures replicas survive upgrades (OLM resets to CSV default).
	if result.Config.OpenShiftRoutesAvailability == openshift.RoutesAvailable {
		namespace := os.Getenv("NAMESPACE")
		if namespace == "" {
			namespace = "opentelemetry-operator-system"
		}
		setupLog.Info("Setting up pod-webhook replica controller",
			"namespace", namespace,
			"desiredReplicas", result.Config.OpenShiftWebhookReplicas)
		if err := (&controllers.PodWebhookReconciler{
			Client:          mgr.GetClient(),
			Namespace:       namespace,
			DesiredReplicas: result.Config.OpenShiftWebhookReplicas,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "PodWebhook")
			os.Exit(1)
		}
	}

	if result.Config.PrometheusCRAvailability == prometheus.Available && result.Config.CreateServiceMonitorOperatorMetrics {
		operatorMetrics, opError := operatormetrics.NewOperatorMetrics(mgr.GetConfig(), scheme, ctrl.Log.WithName("operator-metrics-sm"))
		if opError != nil {
			setupLog.Error(opError, "Failed to create the operator metrics SM")
		}
		err := mgr.Add(operatorMetrics)
		if err != nil {
			setupLog.Error(err, "Failed to add the operator metrics SM")
		}
	}

	if result.Config.EnableWebhooks {
		var crdMetrics *metrics.Metrics

		if result.Config.EnableCRMetrics {
			meterProvider, metricsErr := metrics.Bootstrap()
			if metricsErr != nil {
				setupLog.Error(metricsErr, "Error bootstrapping CRD metrics")
			}

			var err error
			crdMetrics, err = metrics.New(ctx, meterProvider, mgr.GetAPIReader())
			if err != nil {
				setupLog.Error(err, "Error init CRD metrics")
			}
		}

		if result.Config.CollectorAvailability == collector.Available {
			bv := func(ctx context.Context, col otelv1beta1.OpenTelemetryCollector) admission.Warnings {
				var warnings admission.Warnings
				params, newErr := collectorReconciler.GetParams(ctx, col)
				if newErr != nil {
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
			if result.Autodetector.FIPSEnabled(ctx) {
				receivers, exporters, processors, extensions := parseFipsFlag(result.Config.FipsDisabledComponents)
				logger.Info("Fips disabled components", "receivers", receivers, "exporters", exporters, "processors", processors, "extensions", extensions)
				fipsCheck = fips.NewFipsCheck(receivers, exporters, processors, extensions)
			}
			if err := wh.SetupCollectorWebhook(mgr, result.Config, result.Reviewer, crdMetrics, bv, fipsCheck); err != nil {
				setupLog.Error(err, "unable to create webhook", "webhook", "OpenTelemetryCollector")
				os.Exit(1)
			}
		}
		if result.Config.TargetAllocatorAvailability == targetallocator.Available {
			if err := wh.SetupTargetAllocatorWebhook(mgr, result.Config, result.Reviewer); err != nil {
				setupLog.Error(err, "unable to create webhook", "webhook", "TargetAllocator")
				os.Exit(1)
			}
		}
		if err := wh.SetupInstrumentationWebhook(mgr, result.Config); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Instrumentation")
			os.Exit(1)
		}
		decoder := admission.NewDecoder(mgr.GetScheme())
		mgr.GetWebhookServer().Register("/mutate-v1-pod", &webhook.Admission{
			Handler: podmutation.NewWebhookHandler(result.Config, ctrl.Log.WithName("pod-webhook"), decoder, mgr.GetClient(),
				[]podmutation.PodMutator{
					sidecar.NewMutator(logger, result.Config, mgr.GetClient()),
					instrumentation.NewMutator(logger, mgr.GetClient(), mgr.GetEventRecorder("opentelemetry-operator"), result.Config),
				}),
		})

		if result.Config.OpAmpBridgeAvailability == opampbridge.Available {
			if err := wh.SetupOpAMPBridgeWebhook(mgr, result.Config); err != nil {
				setupLog.Error(err, "unable to create webhook", "webhook", "OpAMPBridge")
				os.Exit(1)
			}
		}
	} else {
		ctrl.Log.Info("Webhooks are disabled, operator is running an unsupported mode", "ENABLE_WEBHOOKS", "false")
	}
	// +kubebuilder:scaffold:builder

	operatorsetup.AddHealthChecks(mgr, result.Config.EnableWebhooks)

	operatorsetup.StartManager(ctx, mgr)
}

func discoverKubeAPIServer(ctx context.Context, clientset kubernetes.Interface, cfg *config.Config) error {
	endpointSlices, err := clientset.DiscoveryV1().EndpointSlices("default").List(ctx, metav1.ListOptions{
		LabelSelector: "kubernetes.io/service-name=kubernetes",
	})
	if err != nil {
		return fmt.Errorf("failed to list kubernetes EndpointSlices: %w", err)
	}

	if len(endpointSlices.Items) == 0 {
		return errors.New("no EndpointSlice found for kubernetes service in default namespace")
	}

	for _, endpointSlice := range endpointSlices.Items {
		for _, p := range endpointSlice.Ports {
			if p.Port != nil && p.Name != nil && *p.Name == "https" {
				cfg.Internal.KubeAPIServerPort = *p.Port
				break
			}
		}
		for _, endpoint := range endpointSlice.Endpoints {
			cfg.Internal.KubeAPIServerIPs = append(cfg.Internal.KubeAPIServerIPs, endpoint.Addresses...)
		}
	}

	if cfg.Internal.KubeAPIServerPort == 0 {
		return errors.New("no https port found in kubernetes EndpointSlice")
	}

	if len(cfg.Internal.KubeAPIServerIPs) == 0 {
		return errors.New("no endpoint IPs found in kubernetes EndpointSlice")
	}

	setupLog.Info("Discovered Kubernetes API server", "port", cfg.Internal.KubeAPIServerPort, "ips", cfg.Internal.KubeAPIServerIPs)
	return nil
}

func enableOperatorNetworkPolicy(cfg config.Config, clientset kubernetes.Interface, mgr ctrl.Manager) error {
	operatorNamespace := os.Getenv("NAMESPACE")
	if operatorNamespace == "" {
		return errors.New("NAMESPACE environment variable is not set, it is required for the Operator Network Policy to work")
	}

	if cfg.Internal.KubeAPIServerPort == 0 || len(cfg.Internal.KubeAPIServerIPs) == 0 {
		return errors.New("Kubernetes API server info not discovered from EndpointSlice") //nolint:staticcheck // ST1005
	}

	var policyOpts []operatornetworkpolicy.Option
	policyOpts = append(policyOpts, operatornetworkpolicy.WithOperatorNamespace(operatorNamespace))
	policyOpts = append(policyOpts, operatornetworkpolicy.WithAPIServerPort(cfg.Internal.KubeAPIServerPort))
	policyOpts = append(policyOpts, operatornetworkpolicy.WithAPIServerIPs(cfg.Internal.KubeAPIServerIPs))

	if cfg.OpenShiftRoutesAvailability == openshift.RoutesAvailable {
		policyOpts = append(policyOpts, operatornetworkpolicy.WithAPISererPodLabelSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				"apiserver": "true",
			},
		}))
		policyOpts = append(policyOpts, operatornetworkpolicy.WithAPISererNamespaceLabelSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				"kubernetes.io/metadata.name": "openshift-kube-apiserver",
			},
		}))
	}

	if cfg.EnableWebhooks {
		//nolint:gosec // disable G115
		policyOpts = append(policyOpts, operatornetworkpolicy.WithWebhookPort(int32(cfg.WebhookPort)))
	}
	if cfg.MetricsAddr != "" {
		_, portStr, errParse := net.SplitHostPort(cfg.MetricsAddr)
		if errParse != nil {
			return fmt.Errorf("failed to parse port from metrics address: %w", errParse)
		}
		metricsPort, errParse := strconv.ParseInt(portStr, 10, 32)
		if errParse != nil {
			return fmt.Errorf("failed to parse port for the metrics address :%w", errParse)
		}
		policyOpts = append(policyOpts, operatornetworkpolicy.WithMetricsPort(int32(metricsPort)))
	}
	operatorNetworkPoliciesErr := mgr.Add(operatornetworkpolicy.NewOperatorNetworkPolicy(clientset, mgr.GetScheme(), policyOpts...))
	if operatorNetworkPoliciesErr != nil {
		return fmt.Errorf("failed to create the Operator network policies: %w", operatorNetworkPoliciesErr)
	}
	return nil
}

func addInstrumentationUpgrader(_ context.Context, mgr ctrl.Manager, cfg config.Config) error {
	err := mgr.Add(manager.RunnableFunc(func(c context.Context) error {
		u := instrumentationupgrade.NewInstrumentationUpgrade(
			mgr.GetClient(),
			ctrl.Log.WithName("instrumentation-upgrade"),
			mgr.GetEventRecorder("opentelemetry-operator"),
			cfg,
		)
		return u.ManagedInstances(c)
	}))
	if err != nil {
		return fmt.Errorf("failed to upgrade Instrumentation instances: %w", err)
	}
	return nil
}

func parseFipsFlag(fipsFlag string) (receivers, exporters, processors, extensions []string) {
	split := strings.SplitSeq(fipsFlag, ",")
	for val := range split {
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
