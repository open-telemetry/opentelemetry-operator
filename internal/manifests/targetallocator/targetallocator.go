// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

const (
	ComponentOpenTelemetryTargetAllocator = "opentelemetry-targetallocator"
)

// Build creates the manifest for the TargetAllocator resource.
func Build(params Params) ([]client.Object, error) {
	var resourceManifests []client.Object
	resourceFactories := []manifests.K8sManifestFactory[Params]{
		manifests.Factory(ConfigMap),
		manifests.Factory(Deployment),
		manifests.FactoryWithoutError(ServiceAccount),
		manifests.FactoryWithoutError(Service),
		manifests.Factory(PodDisruptionBudget),
	}

	if params.TargetAllocator.Spec.Observability.Metrics.EnableMetrics && featuregate.PrometheusOperatorIsAvailable.IsEnabled() {
		resourceFactories = append(resourceFactories, manifests.FactoryWithoutError(ServiceMonitor))
	}

	if params.Config.CertManagerAvailability() == certmanager.Available && featuregate.EnableTargetAllocatorMTLS.IsEnabled() {
		resourceFactories = append(resourceFactories,
			manifests.FactoryWithoutError(SelfSignedIssuer),
			manifests.FactoryWithoutError(CACertificate),
			manifests.FactoryWithoutError(CAIssuer),
			manifests.FactoryWithoutError(ServingCertificate),
			manifests.FactoryWithoutError(ClientCertificate),
		)
	}

	for _, factory := range resourceFactories {
		res, err := factory(params)
		if err != nil {
			return nil, err
		} else if manifests.ObjectIsNotNil(res) {
			resourceManifests = append(resourceManifests, res)
		}
	}
	return resourceManifests, nil
}

type Params struct {
	Client          client.Client
	Recorder        record.EventRecorder
	Scheme          *runtime.Scheme
	Log             logr.Logger
	Collector       *v1beta1.OpenTelemetryCollector
	TargetAllocator v1alpha1.TargetAllocator
	Config          config.Config
}
