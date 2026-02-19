// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/components"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	ta "github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator/adapters"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

func ConfigMap(params manifests.Params) (*corev1.ConfigMap, error) {
	// Create a deep copy of the collector to avoid modifying the original
	otelCol := params.OtelCol.DeepCopy()

	// Apply TLS defaults at reconciliation time if configured.
	// This ensures collectors get updated TLS settings when the operator restarts
	// after a cluster TLS profile change, without requiring CR updates.
	if params.Config.Internal.OperandTLSProfile != nil {
		_, err := otelCol.Spec.Config.ApplyDefaults(params.Log, components.WithTLSProfile(params.Config.Internal.OperandTLSProfile))
		if err != nil {
			params.Log.Error(err, "failed to apply TLS defaults to collector config")
			return nil, err
		}
	}

	hash, err := manifestutils.GetConfigMapSHA(otelCol.Spec.Config)
	if err != nil {
		return nil, err
	}
	name := naming.ConfigMap(otelCol.Name, hash)
	collectorName := naming.Collector(otelCol.Name)
	labels := manifestutils.Labels(otelCol.ObjectMeta, collectorName, otelCol.Spec.Image, ComponentOpenTelemetryCollector, []string{})

	annotations, err := manifestutils.Annotations(*otelCol, params.Config.AnnotationsFilter)
	if err != nil {
		return nil, err
	}

	replaceCfgOpts := []ta.TAOption{}

	if otelCol.Spec.TargetAllocator.Enabled && params.Config.CertManagerAvailability == certmanager.Available && featuregate.EnableTargetAllocatorMTLS.IsEnabled() {
		replaceCfgOpts = append(replaceCfgOpts, ta.WithTLSConfig(
			filepath.Join(constants.TACollectorTLSDirPath, constants.TACollectorCAFileName),
			filepath.Join(constants.TACollectorTLSDirPath, constants.TACollectorTLSCertFileName),
			filepath.Join(constants.TACollectorTLSDirPath, constants.TACollectorTLSKeyFileName),
			naming.TAService(otelCol.Name)),
		)
	}

	replacedConf, err := ReplaceConfig(*otelCol, params.TargetAllocator, replaceCfgOpts...)

	if err != nil {
		params.Log.V(2).Info("failed to update prometheus config to use sharded targets: ", "err", err)
		return nil, err
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   otelCol.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Data: map[string]string{
			"collector.yaml": replacedConf,
		},
	}, nil
}
