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
	var manifestSliceFactories []manifests.K8sManifestSliceFactory[manifests.Params]
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
		manifests.Factory(Service),
		manifests.Factory(HeadlessService),
		manifests.Factory(MonitoringService),
		manifests.Factory(ExtensionService),
		manifests.Factory(Ingress),
	}...)

	if featuregate.CollectorUsesTargetAllocatorCR.IsEnabled() {
		manifestFactories = append(manifestFactories, manifests.Factory(TargetAllocator))
	}

	if params.OtelCol.Spec.Observability.Metrics.EnableMetrics && featuregate.PrometheusOperatorIsAvailable.IsEnabled() {
		if params.OtelCol.Spec.Mode == v1beta1.ModeSidecar {
			manifestFactories = append(manifestFactories, manifests.Factory(PodMonitor))
		} else {
			manifestFactories = append(manifestFactories, manifests.Factory(ServiceMonitor), manifests.Factory(ServiceMonitorMonitoring))
		}
	}

	if params.Config.CreateRBACPermissions() == rbac.Available {
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

	manifestSliceFactories = append(
		manifestSliceFactories,
		manifests.FactorySlice(Role),
		manifests.FactorySlice(RoleBinding),
		manifests.FactorySlice(Routes),
	)

	for _, factory := range manifestSliceFactories {
		objs, err := factory(params)
		if err != nil {
			return nil, err
		}
		resourceManifests = append(resourceManifests, objs...)
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

	return resourceManifests, nil
}

func needsCheckSaPermissions(params manifests.Params) bool {
	return params.ErrorAsWarning &&
		params.Config.CreateRBACPermissions() == rbac.NotAvailable &&
		params.Reviewer != nil &&
		params.OtelCol.Spec.ServiceAccount != ""
}
