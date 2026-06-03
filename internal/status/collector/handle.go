// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
)

const (
	reasonError              = "Error"
	reasonStatusFailure      = "StatusFailure"
	reasonInfo               = "Info"
	reasonReconcileError     = "ReconcileError"
	reasonStatusUpdateFailed = "StatusUpdateFailed"
	reasonReconciled         = "Reconciled"

	conditionTypeReady = "Ready"

	msgReconciliationFailed   = "Reconciliation failed"
	msgSuccessfullyReconciled = "Successfully reconciled"
)

// HandleReconcileStatus handles updating the status of the CRDs managed by the operator.
func HandleReconcileStatus(ctx context.Context, log logr.Logger, params manifests.Params, otelcol v1beta1.OpenTelemetryCollector, reconcileErr error) (ctrl.Result, error) {
	log.V(2).Info("updating collector status")

	changed := otelcol.DeepCopy()
	changed.Status.ObservedGeneration = changed.Generation

	if reconcileErr != nil {
		var conditionMsg string
		if reconcileErr.Error() != "" {
			conditionMsg = reconcileErr.Error()
		} else {
			conditionMsg = msgReconciliationFailed
		}
		meta.SetStatusCondition(&changed.Status.Conditions, metav1.Condition{
			Type:               conditionTypeReady,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: changed.Generation,
			Reason:             reasonReconcileError,
			Message:            conditionMsg,
			LastTransitionTime: metav1.Now(),
		})
		statusPatch := client.MergeFrom(&otelcol)
		if err := params.Client.Status().Patch(ctx, changed, statusPatch); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to apply status changes to the OpenTelemetry CR: %w", err)
		}
		params.Recorder.Eventf(&otelcol, nil, corev1.EventTypeWarning, reasonError, reasonError, reconcileErr.Error())
		return ctrl.Result{}, reconcileErr
	}

	statusErr := updateCollectorStatus(ctx, params.Client, changed)

	if statusErr != nil {
		// if status update fails, still update the condition to reflect the error
		meta.SetStatusCondition(&changed.Status.Conditions, metav1.Condition{
			Type:               conditionTypeReady,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: changed.Generation,
			Reason:             reasonStatusUpdateFailed,
			Message:            fmt.Sprintf("Failed to update status fields: %v", statusErr),
			LastTransitionTime: metav1.Now(),
		})
		params.Recorder.Eventf(changed, nil, corev1.EventTypeWarning, reasonStatusFailure, reasonStatusFailure, statusErr.Error())
	} else {
		meta.SetStatusCondition(&changed.Status.Conditions, metav1.Condition{
			Type:               conditionTypeReady,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: changed.Generation,
			Reason:             reasonReconciled,
			Message:            msgSuccessfullyReconciled,
			LastTransitionTime: metav1.Now(),
		})
	}

	statusPatch := client.MergeFrom(&otelcol)
	if err := params.Client.Status().Patch(ctx, changed, statusPatch); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to apply status changes to the OpenTelemetry CR: %w", err)
	}
	params.Recorder.Eventf(changed, nil, corev1.EventTypeNormal, reasonInfo, reasonInfo, "applied status changes")
	return ctrl.Result{}, statusErr
}
