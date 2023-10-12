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

// Package upgrade handles the upgrade routine from one OpenTelemetry Collector to the next.
package upgrade

import (
	"context"
	"fmt"
	"reflect"

	semver "github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
)

type VersionUpgrade struct {
	Client   client.Client
	Recorder record.EventRecorder
	Version  version.Version
	Log      logr.Logger
}

const RecordBufferSize int = 10

// ManagedInstances finds all the otelcol instances for the current operator and upgrades them, if necessary.
func (u VersionUpgrade) ManagedInstances(ctx context.Context) error {
	u.Log.Info("looking for managed instances to upgrade")

	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &v1alpha1.OpenTelemetryCollectorList{}
	if err := u.Client.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	for i := range list.Items {
		original := list.Items[i]
		itemLogger := u.Log.WithValues("name", original.Name, "namespace", original.Namespace)

		if original.Spec.ManagementState == v1alpha1.ManagementStateUnmanaged {
			itemLogger.Info("skipping upgrade because instance is not managed")
			continue
		}

		if original.Spec.UpgradeStrategy == v1alpha1.UpgradeStrategyNone {
			itemLogger.Info("skipping instance upgrade due to UpgradeStrategy")
			continue
		}
		upgraded, err := u.ManagedInstance(ctx, original)
		if err != nil {
			const msg = "automated update not possible. Configuration must be corrected manually and CR instance must be re-created."
			itemLogger.Info(msg)
			u.Recorder.Event(&original, "Error", "Upgrade", msg)
			continue
		}

		if !reflect.DeepEqual(upgraded, list.Items[i]) {
			// the resource update overrides the status, so, keep it so that we can reset it later
			st := upgraded.Status
			patch := client.MergeFrom(&original)
			if err := u.Client.Patch(ctx, &upgraded, patch); err != nil {
				itemLogger.Error(err, "failed to apply changes to instance")
				continue
			}

			// the status object requires its own update
			upgraded.Status = st
			if err := u.Client.Status().Patch(ctx, &upgraded, patch); err != nil {
				itemLogger.Error(err, "failed to apply changes to instance's status object")
				continue
			}

			itemLogger.Info("instance upgraded", "version", upgraded.Status.Version)
		}
	}

	if len(list.Items) == 0 {
		u.Log.Info("no instances to upgrade")
	}

	return nil
}

// ManagedInstance performs the necessary changes to bring the given otelcol instance to the current version.
func (u VersionUpgrade) ManagedInstance(ctx context.Context, otelcol v1alpha1.OpenTelemetryCollector) (v1alpha1.OpenTelemetryCollector, error) {
	// this is likely a new instance, assume it's already up to date
	if otelcol.Status.Version == "" {
		return otelcol, nil
	}

	instanceV, err := semver.NewVersion(otelcol.Status.Version)
	if err != nil {
		u.Log.Error(err, "failed to parse version for OpenTelemetry Collector instance", "name", otelcol.Name, "namespace", otelcol.Namespace, "version", otelcol.Status.Version)
		return otelcol, err
	}

	if instanceV.GreaterThan(&Latest.Version) {
		// Update with the latest known version, which is what we have from versions.txt
		u.Log.Info("no upgrade routines are needed for the OpenTelemetry instance", "name", otelcol.Name, "namespace", otelcol.Namespace, "version", otelcol.Status.Version, "latest", Latest.Version.String())

		otelColV, err := semver.NewVersion(u.Version.OpenTelemetryCollector)
		if err != nil {
			return otelcol, err
		}
		if instanceV.LessThan(otelColV) {
			u.Log.Info("upgraded OpenTelemetry Collector version", "name", otelcol.Name, "namespace", otelcol.Namespace, "version", otelcol.Status.Version)
			otelcol.Status.Version = u.Version.OpenTelemetryCollector
		} else {
			u.Log.Info("skipping upgrade for OpenTelemetry Collector instance", "name", otelcol.Name, "namespace", otelcol.Namespace)
		}

		return otelcol, nil
	}

	for _, available := range versions {
		if available.GreaterThan(instanceV) {
			upgraded, err := available.upgrade(u, &otelcol) //available.upgrade(params., &otelcol)

			if err != nil {
				u.Log.Error(err, "failed to upgrade managed otelcol instances", "name", otelcol.Name, "namespace", otelcol.Namespace)
				return otelcol, err
			}

			u.Log.V(1).Info("step upgrade", "name", otelcol.Name, "namespace", otelcol.Namespace, "version", available.String())
			upgraded.Status.Version = available.String()
			otelcol = *upgraded
		}
	}
	// Update with the latest known version, which is what we have from versions.txt
	otelcol.Status.Version = u.Version.OpenTelemetryCollector

	u.Log.V(1).Info("final version", "name", otelcol.Name, "namespace", otelcol.Namespace, "version", otelcol.Status.Version)
	return otelcol, nil
}
