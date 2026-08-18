// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/opampbridge"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/targetallocator"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/fips"
	"github.com/open-telemetry/opentelemetry-operator/internal/instrumentation"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	collectorManifests "github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/metrics"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
	wh "github.com/open-telemetry/opentelemetry-operator/internal/webhook"
	"github.com/open-telemetry/opentelemetry-operator/internal/webhook/podmutation"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
	"github.com/open-telemetry/opentelemetry-operator/pkg/sidecar"
)

// SetupWebhooks registers all webhooks on the given manager. The bv parameter can be nil
// to skip manifest build validation (e.g. in standalone webhook-server mode, pass
// NewStandaloneBuildValidator; in operator mode, pass the reconciler-based callback).
func SetupWebhooks(ctx context.Context, mgr ctrl.Manager, cfg config.Config, reviewer *rbac.Reviewer, autodetector autodetect.AutoDetect, bv wh.BuildValidator) error {
	logger := ctrl.Log

	var crdMetrics *metrics.Metrics
	if cfg.EnableCRMetrics {
		meterProvider, metricsErr := metrics.Bootstrap()
		if metricsErr != nil {
			setupLog.Error(metricsErr, "Error bootstrapping CRD metrics")
		} else {
			var err error
			crdMetrics, err = metrics.New(ctx, meterProvider, mgr.GetAPIReader())
			if err != nil {
				setupLog.Error(err, "Error init CRD metrics")
			}
		}
	}

	if cfg.CollectorAvailability == collector.Available {
		var fipsCheck fips.FIPSCheck
		if autodetector.FIPSEnabled(ctx) {
			receivers, exporters, processors, extensions := fips.ParseFipsFlag(cfg.FipsDisabledComponents)
			logger.Info("Fips disabled components", "receivers", receivers, "exporters", exporters, "processors", processors, "extensions", extensions)
			fipsCheck = fips.NewFipsCheck(receivers, exporters, processors, extensions)
		}
		if err := wh.SetupCollectorWebhook(mgr, cfg, reviewer, crdMetrics, bv, fipsCheck); err != nil {
			return err
		}
	}

	if cfg.TargetAllocatorAvailability == targetallocator.Available {
		if err := wh.SetupTargetAllocatorWebhook(mgr, cfg, reviewer); err != nil {
			return err
		}
	}

	if err := wh.SetupInstrumentationWebhook(mgr, cfg); err != nil {
		return err
	}

	decoder := admission.NewDecoder(mgr.GetScheme())
	mgr.GetWebhookServer().Register("/mutate-v1-pod", &webhook.Admission{
		Handler: podmutation.NewWebhookHandler(cfg, ctrl.Log.WithName("pod-webhook"), decoder, mgr.GetClient(),
			[]podmutation.PodMutator{
				sidecar.NewMutator(logger, cfg, mgr.GetClient()),
				instrumentation.NewMutator(logger, mgr.GetClient(), mgr.GetEventRecorder("opentelemetry-operator"), cfg),
			}),
	})

	if cfg.OpAmpBridgeAvailability == opampbridge.Available {
		if err := wh.SetupOpAMPBridgeWebhook(mgr, cfg); err != nil {
			return err
		}
	}

	return nil
}

// NewStandaloneBuildValidator creates a BuildValidator that works without the collector
// reconciler. It constructs manifests.Params directly from the manager and config.
func NewStandaloneBuildValidator(mgr ctrl.Manager, cfg config.Config, reviewer rbac.SAReviewer) wh.BuildValidator {
	return func(ctx context.Context, col v1beta1.OpenTelemetryCollector) admission.Warnings {
		var warnings admission.Warnings
		p := manifests.Params{
			Config:         cfg,
			Client:         mgr.GetClient(),
			OtelCol:        col,
			Log:            mgr.GetLogger(),
			Scheme:         mgr.GetScheme(),
			Recorder:       mgr.GetEventRecorder("opentelemetry-operator"),
			Reviewer:       reviewer,
			ErrorAsWarning: true,
		}

		ta, err := getTargetAllocator(ctx, mgr.GetClient(), p)
		if err != nil {
			warnings = append(warnings, err.Error())
			return warnings
		}
		p.TargetAllocator = ta

		_, err = collectorManifests.Build(p)
		if err != nil {
			warnings = append(warnings, err.Error())
			return warnings
		}
		return warnings
	}
}

func getTargetAllocator(ctx context.Context, cl client.Client, params manifests.Params) (*v1alpha1.TargetAllocator, error) {
	if taName, ok := params.OtelCol.GetLabels()[constants.LabelTargetAllocator]; ok {
		ta := &v1alpha1.TargetAllocator{}
		taKey := client.ObjectKey{Name: taName, Namespace: params.OtelCol.GetNamespace()}
		err := cl.Get(ctx, taKey, ta)
		if err != nil {
			return nil, err
		}
		return ta, nil
	}
	return collectorManifests.TargetAllocator(params)
}
