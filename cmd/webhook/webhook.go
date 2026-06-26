// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"flag"
	"os"

	"github.com/spf13/cobra"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/instrumentation"
	operatorsetup "github.com/open-telemetry/opentelemetry-operator/internal/operator"
	wh "github.com/open-telemetry/opentelemetry-operator/internal/webhook"
	"github.com/open-telemetry/opentelemetry-operator/internal/webhook/podmutation"
	"github.com/open-telemetry/opentelemetry-operator/pkg/sidecar"
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

	logger := ctrl.Log
	mgr := result.Manager

	if err := wh.SetupInstrumentationWebhook(mgr, result.Config); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Instrumentation")
		os.Exit(1)
	}

	podMutators := []podmutation.PodMutator{
		sidecar.NewMutator(logger, result.Config, mgr.GetClient()),
		instrumentation.NewMutator(logger, mgr.GetClient(), mgr.GetEventRecorder("opentelemetry-operator"), result.Config),
	}

	decoder := admission.NewDecoder(mgr.GetScheme())
	mgr.GetWebhookServer().Register("/mutate-v1-pod", &webhook.Admission{
		Handler: podmutation.NewWebhookHandler(result.Config, ctrl.Log.WithName("pod-webhook"), decoder, mgr.GetClient(), podMutators),
	})

	operatorsetup.AddHealthChecks(mgr, true)

	signalCtx := ctrl.SetupSignalHandler()
	ctx, cancel := context.WithCancel(signalCtx)
	defer cancel()

	if result.Config.TLS.UseClusterProfile {
		operatorsetup.SetupTLSProfileWatcher(mgr, result.InitialTLSProfile, cancel)
	}

	operatorsetup.StartManager(mgr, ctx)
}
