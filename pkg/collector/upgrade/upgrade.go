// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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

// ManagedInstances finds all the otelcol instances for the current operator and upgrades them, if necessary.
func (u VersionUpgrade) ManagedInstances(ctx context.Context) error {
	u.Log.Info("looking for managed instances to upgrade")
	list := &v1beta1.OpenTelemetryCollectorList{}
	if err := u.Client.List(ctx, list); err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	for i := range list.Items {
		original := list.Items[i]
		itemLogger := u.Log.WithValues("name", original.Name, "namespace", original.Namespace)

		if original.Spec.ManagementState == v1beta1.ManagementStateUnmanaged {
			itemLogger.Info("skipping upgrade because instance is not managed")
			continue
		}

		if original.Spec.UpgradeStrategy == v1beta1.UpgradeStrategyNone {
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
					u.Log.Error(err, "failed to upgrade managed otelcol instances", "name", updated.Name, "namespace", updated.Namespace)
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
					u.Log.Error(err, "failed to upgrade managed otelcol instances", "name", updated.Name, "namespace", updated.Namespace)
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
