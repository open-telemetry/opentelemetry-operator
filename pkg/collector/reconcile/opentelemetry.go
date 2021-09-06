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

// Package reconcile contains reconciliation logic for OpenTelemetry Collector components.
package reconcile

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/signalfx/splunk-otel-operator/internal/version"
)

// Self updates this instance's self data. This should be the last item in the reconciliation, as it causes changes
// making params.Instance obsolete. Default values should be set in the Defaulter webhook, this should only be used
// for the Status, which can't be set by the defaulter.
func Self(ctx context.Context, params Params) error {
	if params.Instance.Status.Version != "" {
		// a version is already set, let the upgrade mechanism take care of it!
		return nil
	}

	changed := params.Instance
	changed.Status.Version = version.SplunkOtelCollector()

	// this is only a change for new instances: existing instances are reconciled when the operator is first started
	statusPatch := client.MergeFrom(&params.Instance)
	if err := params.Client.Status().Patch(ctx, &changed, statusPatch); err != nil {
		return fmt.Errorf("failed to apply status changes to the OpenTelemetry CR: %w", err)
	}

	return nil
}
