// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	conditionTypeReady        = "Ready"
	reasonReconciled          = "Reconciled"
	msgSuccessfullyReconciled = "Successfully reconciled"
)

func HandleReconcileStatus(ctx context.Context, log logr.Logger, c client.Client, inst v1alpha1.Instrumentation) (ctrl.Result, error) {
	log.V(2).Info("updating instrumentation status")

	changed := inst.DeepCopy()
	changed.Status.ObservedGeneration = changed.Generation

	meta.SetStatusCondition(&changed.Status.Conditions, metav1.Condition{
		Type:               conditionTypeReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: changed.Generation,
		Reason:             reasonReconciled,
		Message:            msgSuccessfullyReconciled,
		LastTransitionTime: metav1.Now(),
	})

	statusPatch := client.MergeFrom(&inst)
	if err := c.Status().Patch(ctx, changed, statusPatch); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to apply status changes to the Instrumentation CR: %w", err)
	}
	return ctrl.Result{}, nil
}
