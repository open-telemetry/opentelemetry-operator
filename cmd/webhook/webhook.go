// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"flag"
	"os"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/spf13/cobra"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/prometheus"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	operatorsetup "github.com/open-telemetry/opentelemetry-operator/internal/operator"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

var setupLog = ctrl.Log.WithName("setup")

// NewWebhookCmd creates the webhook-server cobra subcommand.
func NewWebhookCmd(scheme *k8sruntime.Scheme) *cobra.Command {
	cfg := config.New()
	cliFlags := config.CreateCLIParser(cfg)

	opts := zap.Options{}
	var zapFlagSet flag.FlagSet
	opts.BindFlags(&zapFlagSet)
	cliFlags.AddGoFlagSet(&zapFlagSet)

	var configFile string
	cliFlags.StringVar(&configFile, "config-file", "", "Path to config file")

	cmd := &cobra.Command{
		Use:   "webhook-server",
		Short: "Run only the webhooks",
		Long:  "Run only the webhooks without the controllers.",
		Run: func(_ *cobra.Command, _ []string) {
			runWebhookServer(cfg, configFile, opts, scheme)
		},
	}

	cmd.Flags().AddFlagSet(cliFlags)
	return cmd
}

// runWebhookServer runs only the webhooks without the controllers.
// This enables High Availability (HA) deployment where the webhook can be scaled independently.
func runWebhookServer(cfg config.Config, configFile string, opts zap.Options, scheme *k8sruntime.Scheme) {
	result := operatorsetup.SetupManager(&cfg, configFile, opts, scheme, false, "Starting the OpenTelemetry webhook server")

	if result.Config.FeatureGates != "" {
		configLog := ctrl.Log.WithName("config")
		configLog.Info("Applying feature gates from configuration", "gates", result.Config.FeatureGates)
		if err := featuregate.ApplyFeatureGateOverrides(result.Config.FeatureGates); err != nil {
			setupLog.Error(err, "failed to apply feature gate overrides")
			os.Exit(1)
		}
	}

	mgr := result.Manager

	if result.Config.PrometheusCRAvailability == prometheus.Available {
		setupLog.Info("Prometheus CRDs are installed, adding to scheme.")
		utilruntime.Must(monitoringv1.AddToScheme(scheme))
	}
	if result.Config.OpenShiftRoutesAvailability == openshift.RoutesAvailable {
		setupLog.Info("Openshift CRDs are installed, adding to scheme.")
		utilruntime.Must(routev1.Install(scheme))
	}
	if result.Config.CertManagerAvailability == certmanager.Available {
		setupLog.Info("Cert-Manager is available to the operator, adding to scheme.")
		utilruntime.Must(cmv1.AddToScheme(scheme))
	}

	signalCtx := ctrl.SetupSignalHandler()
	ctx, cancel := context.WithCancel(signalCtx)
	defer cancel()

	bv := operatorsetup.NewStandaloneBuildValidator(mgr, result.Config, result.Reviewer)
	if err := operatorsetup.SetupWebhooks(ctx, mgr, result.Config, result.Reviewer, result.Autodetector, bv); err != nil {
		setupLog.Error(err, "unable to setup webhooks")
		os.Exit(1)
	}

	operatorsetup.AddHealthChecks(mgr, true)

	if result.Config.TLS.UseClusterProfile {
		operatorsetup.SetupTLSProfileWatcher(mgr, result.InitialTLSProfile, cancel)
	}

	operatorsetup.StartManager(ctx, mgr)
}
