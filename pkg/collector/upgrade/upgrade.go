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
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/signalfx/splunk-otel-operator/api/v1alpha1"
	"github.com/signalfx/splunk-otel-operator/internal/version"
)

// ManagedInstances finds all the otelcol instances for the current operator and upgrades them, if necessary.
func ManagedInstances(ctx context.Context, logger logr.Logger, ver version.Version, cl client.Client) error {
	logger.Info("looking for managed instances to upgrade")

	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/managed-by": "splunk-otel-operator",
		}),
	}
	list := &v1alpha1.SplunkOtelAgentList{}
	if err := cl.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	for i := range list.Items {
		original := list.Items[i]
		upgraded, err := ManagedInstance(ctx, logger, ver, cl, original)
		if err != nil {
			// nothing to do at this level, just go to the next instance
			continue
		}

		if !reflect.DeepEqual(upgraded, list.Items[i]) {
			// the resource update overrides the status, so, keep it so that we can reset it later
			st := upgraded.Status
			patch := client.MergeFrom(&original)
			if err := cl.Patch(ctx, &upgraded, patch); err != nil {
				logger.Error(err, "failed to apply changes to instance", "name", upgraded.Name, "namespace", upgraded.Namespace)
				continue
			}

			// the status object requires its own update
			upgraded.Status = st
			if err := cl.Status().Patch(ctx, &upgraded, patch); err != nil {
				logger.Error(err, "failed to apply changes to instance's status object", "name", upgraded.Name, "namespace", upgraded.Namespace)
				continue
			}

			logger.Info("instance upgraded", "name", upgraded.Name, "namespace", upgraded.Namespace, "version", upgraded.Status.Version)
		}
	}

	if len(list.Items) == 0 {
		logger.Info("no instances to upgrade")
	}

	return nil
}

// ManagedInstance performs the necessary changes to bring the given otelcol instance to the current version.
func ManagedInstance(ctx context.Context, logger logr.Logger, currentV version.Version, cl client.Client, otelcol v1alpha1.SplunkOtelAgent) (v1alpha1.SplunkOtelAgent, error) {
	// this is likely a new instance, assume it's already up to date
	if otelcol.Status.Version == "" {
		return otelcol, nil
	}

	instanceV, err := semver.NewVersion(otelcol.Status.Version)
	if err != nil {
		logger.Error(err, "failed to parse version for OpenTelemetry Collector instance", "name", otelcol.Name, "namespace", otelcol.Namespace, "version", otelcol.Status.Version)
		return otelcol, err
	}

	if instanceV.GreaterThan(&Latest.Version) {
		logger.Info("skipping upgrade for OpenTelemetry Collector instance, as it's newer than our latest version", "name", otelcol.Name, "namespace", otelcol.Namespace, "version", otelcol.Status.Version, "latest", Latest.Version.String())
		return otelcol, nil
	}

	for _, available := range versions {
		if available.GreaterThan(instanceV) {
			upgraded, err := available.upgrade(cl, &otelcol)

			if err != nil {
				logger.Error(err, "failed to upgrade managed otelcol instances", "name", otelcol.Name, "namespace", otelcol.Namespace)
				return otelcol, err
			}

			logger.V(1).Info("step upgrade", "name", otelcol.Name, "namespace", otelcol.Namespace, "version", available.String())
			upgraded.Status.Version = available.String()
			otelcol = *upgraded
		}
	}

	// at the end of the process, we are up to date with the latest known version, which is what we have from versions.txt
	otelcol.Status.Version = currentV.SplunkOtelCollector

	logger.V(1).Info("final version", "name", otelcol.Name, "namespace", otelcol.Namespace, "version", otelcol.Status.Version)
	return otelcol, nil
}
