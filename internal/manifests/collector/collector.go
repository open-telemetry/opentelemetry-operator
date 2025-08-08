// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

const (
	ComponentOpenTelemetryCollector = "opentelemetry-collector"
)

// Build creates the manifest for the collector resource.
func Build(params manifests.Params) ([]client.Object, error) {
	var resourceManifests []client.Object
	var manifestFactories []manifests.K8sManifestFactory[manifests.Params]
	switch params.OtelCol.Spec.Mode {
	case v1beta1.ModeDeployment:
		manifestFactories = append(manifestFactories, manifests.Factory(Deployment))
		manifestFactories = append(manifestFactories, manifests.Factory(PodDisruptionBudget))
	case v1beta1.ModeStatefulSet:
		manifestFactories = append(manifestFactories, manifests.Factory(StatefulSet))
		manifestFactories = append(manifestFactories, manifests.Factory(PodDisruptionBudget))
	case v1beta1.ModeDaemonSet:
		manifestFactories = append(manifestFactories, manifests.Factory(DaemonSet))
	case v1beta1.ModeSidecar:
		params.Log.V(5).Info("not building sidecar...")
	}
	manifestFactories = append(manifestFactories, []manifests.K8sManifestFactory[manifests.Params]{
		manifests.Factory(ConfigMap),
		manifests.Factory(HorizontalPodAutoscaler),
		manifests.Factory(ServiceAccount),
		manifests.Factory(Ingress),
	}...)
	if params.OtelCol.Spec.Service.IsEnabled() {
		manifestFactories = append(manifestFactories, manifests.Factory(Service))
	}
	if params.OtelCol.Spec.HeadlessService.IsEnabled() {
		manifestFactories = append(manifestFactories, manifests.Factory(HeadlessService))
	}
	if params.OtelCol.Spec.MonitoringService.IsEnabled() {
		manifestFactories = append(manifestFactories, manifests.Factory(MonitoringService))
	}
	if params.OtelCol.Spec.ExtensionService.IsEnabled() {
		manifestFactories = append(manifestFactories, manifests.Factory(ExtensionService))
	}

	if featuregate.CollectorUsesTargetAllocatorCR.IsEnabled() {
		manifestFactories = append(manifestFactories, manifests.Factory(TargetAllocator))
	}

	if params.OtelCol.Spec.Observability.Metrics.EnableMetrics {
		if params.OtelCol.Spec.Mode == v1beta1.ModeSidecar {
			manifestFactories = append(manifestFactories, manifests.Factory(PodMonitor))
		} else {
			if params.OtelCol.Spec.Service.IsEnabled() {
				manifestFactories = append(manifestFactories, manifests.Factory(ServiceMonitor))
			}
			if params.OtelCol.Spec.MonitoringService.IsEnabled() {
				manifestFactories = append(manifestFactories, manifests.Factory(ServiceMonitorMonitoring))
			}
		}
	}

	if params.Config.CreateRBACPermissions == rbac.Available {
		manifestFactories = append(manifestFactories,
			manifests.Factory(ClusterRole),
			manifests.Factory(ClusterRoleBinding),
		)
	}

	for _, factory := range manifestFactories {
		res, err := factory(params)
		if err != nil {
			return nil, err
		} else if manifests.ObjectIsNotNil(res) {
			resourceManifests = append(resourceManifests, res)
		}
	}

	if needsCheckSaPermissions(params) {
		warnings, err := CheckRbacRules(params, params.OtelCol.Spec.ServiceAccount)
		if err != nil {
			return nil, fmt.Errorf("error checking RBAC rules for serviceAccount %s: %w", params.OtelCol.Spec.ServiceAccount, err)
		}

		var w []error
		for _, warning := range warnings {
			w = append(w, fmt.Errorf("RBAC rules are missing: %s", warning))
		}
		return nil, errors.Join(w...)
	}

	routes, err := Routes(params)
	if err != nil {
		return nil, err
	}
	// NOTE: we cannot just unpack the slice, the type checker doesn't coerce the type correctly.
	for _, route := range routes {
		resourceManifests = append(resourceManifests, route)
	}
	return resourceManifests, nil
}

func needsCheckSaPermissions(params manifests.Params) bool {
	return params.ErrorAsWarning &&
		params.Config.CreateRBACPermissions == rbac.NotAvailable &&
		params.Reviewer != nil &&
		params.OtelCol.Spec.ServiceAccount != ""
}
