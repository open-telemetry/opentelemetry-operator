// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/targetallocator"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
)

const (
	eventTypeNormal  = "Normal"
	eventTypeWarning = "Warning"

	reasonError         = "Error"
	reasonStatusFailure = "StatusFailure"
	reasonInfo          = "Info"
)

// HandleReconcileStatus handles updating the status of the CRDs managed by the operator.
func HandleReconcileStatus(ctx context.Context, log logr.Logger, params targetallocator.Params, err error) (ctrl.Result, error) {
	log.V(2).Info("updating opampbridge status")
	if err != nil {
		params.Recorder.Event(&params.TargetAllocator, eventTypeWarning, reasonError, err.Error())
		return ctrl.Result{}, err
	}
	changed := params.TargetAllocator.DeepCopy()

	if changed.Status.Version == "" {
		changed.Status.Version = version.TargetAllocator()
	}
	statusPatch := client.MergeFrom(&params.TargetAllocator)
	if err := params.Client.Status().Patch(ctx, changed, statusPatch); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to apply status changes to the OpenTelemetry CR: %w", err)
	}
	params.Recorder.Event(changed, eventTypeNormal, reasonInfo, "applied status changes")
	return ctrl.Result{}, nil
}
