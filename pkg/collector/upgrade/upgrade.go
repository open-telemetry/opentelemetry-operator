// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package upgrade handles the upgrade routine from one OpenTelemetry Collector to the next.
package upgrade

import (
	"context"
	"reflect"

	semver "github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
)

type VersionUpgrade struct {
	Client   client.Client
	Recorder record.EventRecorder
	Version  version.Version
	Log      logr.Logger
}

const RecordBufferSize int = 100

func (u VersionUpgrade) semVer() *semver.Version {
	if len(u.Version.OpenTelemetryCollector) == 0 {
		return &Latest.Version
	}
	if v, err := semver.NewVersion(u.Version.OpenTelemetryCollector); err != nil {
		return &Latest.Version
	} else {
		return v
	}
}

// NeedsUpgrade checks if this CR needs to be upgraded.
func (u VersionUpgrade) NeedsUpgrade(instance v1beta1.OpenTelemetryCollector) bool {
	// CRs with an empty version are ignored, as they're already up-to-date and
	// the version will be set when the status field is refreshed.
	return instance.Status.Version != "" &&
		instance.Status.Version != u.Version.OpenTelemetryCollector &&
		instance.Spec.ManagementState != v1beta1.ManagementStateUnmanaged &&
		instance.Spec.UpgradeStrategy != v1beta1.UpgradeStrategyNone
}

// Upgrade performs an upgrade of an OpenTelemetryCollector CR in the cluster.
func (u VersionUpgrade) Upgrade(ctx context.Context, original v1beta1.OpenTelemetryCollector) error {
	if !u.NeedsUpgrade(original) {
		return nil
	}

	itemLogger := u.Log.WithValues("name", original.Name, "namespace", original.Namespace)
	upgraded, err := u.ManagedInstance(ctx, original)
	if err != nil {
		const msg = "automated update not possible. Configuration must be corrected manually and CR instance must be re-created."
		itemLogger.Info(msg)
		u.Recorder.Event(&original, corev1.EventTypeWarning, "Upgrade", msg)
		return err
	}
	if !reflect.DeepEqual(upgraded, original) {
		// the resource update overrides the status, so, keep it so that we can reset it later
		st := upgraded.Status
		patch := client.MergeFrom(&original)
		if err := u.Client.Patch(ctx, &upgraded, patch); err != nil {
			itemLogger.Error(err, "failed to apply changes to instance")
			return err
		}

		// the status object requires its own update
		upgraded.Status = st
		if err := u.Client.Status().Patch(ctx, &upgraded, patch); err != nil {
			itemLogger.Error(err, "failed to apply changes to instance's status object")
			return err
		}
		itemLogger.Info("instance upgraded", "version", upgraded.Status.Version)
	}

	return nil
}

// ManagedInstance performs the necessary changes to bring the given otelcol instance to the current version.
func (u VersionUpgrade) ManagedInstance(_ context.Context, otelcol v1beta1.OpenTelemetryCollector) (v1beta1.OpenTelemetryCollector, error) {
	// this is likely a new instance, assume it's already up to date
	if otelcol.Status.Version == "" {
		return otelcol, nil
	}

	instanceV, err := semver.NewVersion(otelcol.Status.Version)
	if err != nil {
		u.Log.Error(err, "failed to parse version for OpenTelemetry Collector instance", "name", otelcol.Name, "namespace", otelcol.Namespace, "version", otelcol.Status.Version)
		return otelcol, err
	}

	updated := *(otelcol.DeepCopy())
	if instanceV.GreaterThan(u.semVer()) {
		// Update with the latest known version, which is what we have from versions.txt
		u.Log.V(4).Info("no upgrade routines are needed for the OpenTelemetry instance", "name", updated.Name, "namespace", updated.Namespace, "version", updated.Status.Version, "latest", u.semVer().String())

		otelColV, err := semver.NewVersion(u.Version.OpenTelemetryCollector)
		if err != nil {
			return updated, err
		}
		if instanceV.LessThan(otelColV) {
			u.Log.Info("upgraded OpenTelemetry Collector version", "name", updated.Name, "namespace", updated.Namespace, "version", updated.Status.Version)
			updated.Status.Version = u.Version.OpenTelemetryCollector
		} else {
			u.Log.V(4).Info("skipping upgrade for OpenTelemetry Collector instance", "name", updated.Name, "namespace", updated.Namespace)
		}

		return updated, nil
	}

	for _, available := range versions {
		// Don't run upgrades for versions after the webhook's set version.
		// This is important only for testing.
		if available.GreaterThan(u.semVer()) {
			continue
		}
		if available.GreaterThan(instanceV) {
			if available.upgrade != nil {
				otelcolV1alpha1 := &v1alpha1.OpenTelemetryCollector{}
				if err := otelcolV1alpha1.ConvertFrom(&updated); err != nil {
					return updated, err
				}

				upgradedV1alpha1, err := available.upgrade(u, otelcolV1alpha1)
				if err != nil {
					u.Log.Error(err, "failed to upgrade managed otelcol instance", "name", updated.Name, "namespace", updated.Namespace)
					return updated, err
				}
				upgradedV1alpha1.Status.Version = available.String()

				if err := upgradedV1alpha1.ConvertTo(&updated); err != nil {
					return updated, err
				}
				u.Log.V(1).Info("step upgrade", "name", updated.Name, "namespace", updated.Namespace, "version", available.String())
			} else {

				upgraded, err := available.upgradeV1beta1(u, &updated) //available.upgrade(params., &updated)
				if err != nil {
					u.Log.Error(err, "failed to upgrade managed otelcol instance", "name", updated.Name, "namespace", updated.Namespace)
					return updated, err
				}

				u.Log.V(1).Info("step upgrade", "name", updated.Name, "namespace", updated.Namespace, "version", available.String())
				upgraded.Status.Version = available.String()
				updated = *upgraded
			}
		}
	}
	// Update with the latest known version, which is what we have from versions.txt
	updated.Status.Version = u.Version.OpenTelemetryCollector

	u.Log.V(1).Info("final version", "name", updated.Name, "namespace", updated.Namespace, "version", updated.Status.Version)
	return updated, nil
}
