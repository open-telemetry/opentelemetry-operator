// Copyright The OpenTelemetry Authors
// Copyright Splunk Inc.
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

package reconcile

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/signalfx/splunk-otel-operator/pkg/collector"
	"github.com/signalfx/splunk-otel-operator/pkg/naming"
)

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// ConfigMaps reconciles the namespace(s) required for the instance in the current context.
func Namespaces(ctx context.Context, params Params) error {
	desired := desiredNamespace(ctx, params)

	if err := expectedNamespace(ctx, params, desired, true); err != nil {
		return fmt.Errorf("failed to reconcile the expected configmaps: %w", err)
	}

	return nil
}

func desiredNamespace(_ context.Context, params Params) corev1.Namespace {
	name := naming.Namespace(params.Instance)
	labels := collector.Labels(params.Instance)
	labels["app.kubernetes.io/name"] = name

	return corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      labels,
			Annotations: params.Instance.Annotations,
		},
	}
}

func expectedNamespace(ctx context.Context, params Params, expected corev1.Namespace, retry bool) error {
	if err := controllerutil.SetControllerReference(&params.Instance, &expected, params.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	existing := &corev1.Namespace{}
	err := params.Client.Get(ctx, client.ObjectKey{Name: expected.Name}, existing)
	if err != nil && errors.IsNotFound(err) {
		// create namespace
		if err := params.Client.Create(ctx, &expected); err != nil {
			if errors.IsAlreadyExists(err) && retry {
				// let's try again? we probably had multiple updates at one, and now it exists already
				if err := expectedNamespace(ctx, params, expected, false); err != nil {
					// somethin else happened now...
					return err
				}

				// we succeeded in the retry, exit this attempt
				return nil
			}
			return fmt.Errorf("failed to create: %w", err)
		}
		params.Log.V(2).Info("created", "namespace", expected.Name)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get: %w", err)
	}

	updated := existing.DeepCopy()
	if updated.Annotations == nil {
		updated.Annotations = map[string]string{}
	}
	if updated.Labels == nil {
		updated.Labels = map[string]string{}
	}

	updated.ObjectMeta.OwnerReferences = expected.ObjectMeta.OwnerReferences

	for k, v := range expected.ObjectMeta.Annotations {
		updated.ObjectMeta.Annotations[k] = v
	}
	for k, v := range expected.ObjectMeta.Labels {
		updated.ObjectMeta.Labels[k] = v
	}

	patch := client.MergeFrom(existing)

	if err := params.Client.Patch(ctx, updated, patch); err != nil {
		return fmt.Errorf("failed to apply changes: %w", err)
	}

	params.Log.V(2).Info("applied", "namespace.name", expected.Name)

	return nil
}
