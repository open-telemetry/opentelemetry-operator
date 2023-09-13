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
	"strconv"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	collectorupgrade "github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
)

// Self updates this instance's self data. This should be the last item in the reconciliation, as it causes changes
// making params.Instance obsolete. Default values should be set in the Defaulter webhook, this should only be used
// for the Status, which can't be set by the defaulter.
func Self(ctx context.Context, params manifests.Params) error {
	changed := params.Instance

	up := &collectorupgrade.VersionUpgrade{
		Log:      params.Log,
		Version:  version.Get(),
		Client:   params.Client,
		Recorder: params.Recorder,
	}
	changed, err := up.ManagedInstance(ctx, changed)
	if err != nil {
		return fmt.Errorf("failed to upgrade the OpenTelemetry CR: %w", err)
	}

	// this field is only changed for new instances: on existing instances this
	// field is reconciled when the operator is first started, i.e. during
	// the upgrade mechanism
	if params.Instance.Status.Version == "" {
		// a version is not set, otherwise let the upgrade mechanism take care of it!
		changed.Status.Version = version.OpenTelemetryCollector()
	}

	if err = updateScaleSubResourceStatus(ctx, params.Client, &changed); err != nil {
		return fmt.Errorf("failed to update the scale subresource status for the OpenTelemetry CR: %w", err)
	}

	statusPatch := client.MergeFrom(&params.Instance)
	if err := params.Client.Status().Patch(ctx, &changed, statusPatch); err != nil {
		return fmt.Errorf("failed to apply status changes to the OpenTelemetry CR: %w", err)
	}

	return nil
}

func updateScaleSubResourceStatus(ctx context.Context, cli client.Client, changed *v1alpha1.OpenTelemetryCollector) error {
	mode := changed.Spec.Mode
	if mode != v1alpha1.ModeDeployment && mode != v1alpha1.ModeStatefulSet {
		changed.Status.Scale.Replicas = 0
		changed.Status.Scale.Selector = ""

		return nil
	}

	name := naming.Collector(changed.Name)

	// Set the scale selector
	labels := collector.Labels(*changed, name, []string{})
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: labels})
	if err != nil {
		return fmt.Errorf("failed to get selector for labelSelector: %w", err)
	}
	changed.Status.Scale.Selector = selector.String()

	// Set the scale replicas
	objKey := client.ObjectKey{
		Namespace: changed.GetNamespace(),
		Name:      naming.Collector(changed.Name),
	}

	var replicas int32
	var readyReplicas int32
	var statusReplicas string
	var statusImage string

	switch mode { // nolint:exhaustive
	case v1alpha1.ModeDeployment:
		obj := &appsv1.Deployment{}
		if err := cli.Get(ctx, objKey, obj); err != nil {
			return fmt.Errorf("failed to get deployment status.replicas: %w", err)
		}
		replicas = obj.Status.Replicas
		readyReplicas = obj.Status.ReadyReplicas
		statusReplicas = strconv.Itoa(int(readyReplicas)) + "/" + strconv.Itoa(int(replicas))
		statusImage = obj.Spec.Template.Spec.Containers[0].Image

	case v1alpha1.ModeStatefulSet:
		obj := &appsv1.StatefulSet{}
		if err := cli.Get(ctx, objKey, obj); err != nil {
			return fmt.Errorf("failed to get statefulSet status.replicas: %w", err)
		}
		replicas = obj.Status.Replicas
		readyReplicas = obj.Status.ReadyReplicas
		statusReplicas = strconv.Itoa(int(readyReplicas)) + "/" + strconv.Itoa(int(replicas))
		statusImage = obj.Spec.Template.Spec.Containers[0].Image

	case v1alpha1.ModeDaemonSet:
		obj := &appsv1.DaemonSet{}
		if err := cli.Get(ctx, objKey, obj); err != nil {
			return fmt.Errorf("failed to get daemonSet status.replicas: %w", err)
		}
		statusImage = obj.Spec.Template.Spec.Containers[0].Image
	}
	changed.Status.Scale.Replicas = replicas
	changed.Status.Image = statusImage
	changed.Status.Scale.StatusReplicas = statusReplicas

	return nil
}
