// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	collectorupgrade "github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
)

const (
	eventTypeNormal  = "Normal"
	eventTypeWarning = "Warning"

	reasonError         = "Error"
	reasonStatusFailure = "StatusFailure"
	reasonInfo          = "Info"
)

// HandleReconcileStatus handles updating the status of the CRDs managed by the operator.
// TODO: make the status more useful https://github.com/open-telemetry/opentelemetry-operator/issues/1972
func HandleReconcileStatus(ctx context.Context, log logr.Logger, params manifests.Params, err error) (ctrl.Result, error) {
	log.V(2).Info("updating collector status")
	if err != nil {
		params.Recorder.Event(&params.OtelCol, eventTypeWarning, reasonError, err.Error())
		return ctrl.Result{}, err
	}
	changed := params.OtelCol.DeepCopy()

	up := &collectorupgrade.VersionUpgrade{
		Log:      params.Log,
		Version:  version.Get(),
		Client:   params.Client,
		Recorder: params.Recorder,
	}
	upgraded, upgradeErr := up.ManagedInstance(ctx, *changed)
	if upgradeErr != nil {
		// don't fail to allow setting the status
		params.Log.Error(upgradeErr, "failed to upgrade the OpenTelemetry CR")
	}
	changed = &upgraded
	statusErr := UpdateCollectorStatus(ctx, params.Client, changed)
	if statusErr != nil {
		params.Recorder.Event(changed, eventTypeWarning, reasonStatusFailure, statusErr.Error())
		return ctrl.Result{}, statusErr
	}
	statusPatch := client.MergeFrom(&params.OtelCol)
	if err := params.Client.Status().Patch(ctx, changed, statusPatch); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to apply status changes to the OpenTelemetry CR: %w", err)
	}
	params.Recorder.Event(changed, eventTypeNormal, reasonInfo, "applied status changes")
	return ctrl.Result{}, nil
}
