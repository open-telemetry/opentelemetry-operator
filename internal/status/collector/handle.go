// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
)

const (
	reasonError         = "Error"
	reasonStatusFailure = "StatusFailure"
	reasonInfo          = "Info"
)

// HandleReconcileStatus handles updating the status of the CRDs managed by the operator.
func HandleReconcileStatus(ctx context.Context, log logr.Logger, params manifests.Params, otelcol v1beta1.OpenTelemetryCollector, err error) (ctrl.Result, error) {
	log.V(2).Info("updating collector status")
	if err != nil {
		params.Recorder.Event(&otelcol, corev1.EventTypeWarning, reasonError, err.Error())
		return ctrl.Result{}, err
	}

	changed := otelcol.DeepCopy()
	statusErr := updateCollectorStatus(ctx, params.Client, changed)

	if statusErr != nil {
		params.Recorder.Event(changed, corev1.EventTypeWarning, reasonStatusFailure, statusErr.Error())
		return ctrl.Result{}, statusErr
	}
	statusPatch := client.MergeFrom(&otelcol)
	if err := params.Client.Status().Patch(ctx, changed, statusPatch); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to apply status changes to the OpenTelemetry CR: %w", err)
	}
	params.Recorder.Event(changed, corev1.EventTypeNormal, reasonInfo, "applied status changes")
	return ctrl.Result{}, nil
}
