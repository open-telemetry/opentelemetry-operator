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

package reconcile

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/pkg/loadbalancer"
	lbadapters "github.com/open-telemetry/opentelemetry-operator/pkg/loadbalancer/adapters"
)

// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Deployments reconciles the deployment(s) required for the instance in the current context.
func Deployments(ctx context.Context, params Params) error {
	desired := []appsv1.Deployment{}
	config, err := adapters.ConfigFromString(params.Instance.Spec.Config)
	if err != nil {
		return fmt.Errorf("failed to parse the config: %v", err)
	}

	// Notify returns an empty string if there is a valid Prometheus configuration
	_, notify := lbadapters.ConfigToPromConfig(config)

	if params.Instance.Spec.Mode == v1alpha1.ModeStatefulSet && len(params.Instance.Spec.LoadBalancer.Mode) > 0 && notify == "" {
		desired = append(desired, loadbalancer.Deployment(params.Config, params.Log, params.Instance))
	}

	// first, handle the create/update parts
	if err := expectedDeployments(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected deployments: %v", err)
	}

	// then, delete the extra objects
	if err := deleteDeployments(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the deployments to be deleted: %v", err)
	}

	return nil
}

func expectedDeployments(ctx context.Context, params Params, expected []appsv1.Deployment) error {
	for _, obj := range expected {
		desired := obj

		if err := controllerutil.SetControllerReference(&params.Instance, &desired, params.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}

		existing := &appsv1.Deployment{}
		nns := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
		err := params.Client.Get(ctx, nns, existing)
		if err != nil && k8serrors.IsNotFound(err) {
			if err := params.Client.Create(ctx, &desired); err != nil {
				return fmt.Errorf("failed to create: %w", err)
			}
			params.Log.V(2).Info("created", "deployment.name", desired.Name, "deployment.namespace", desired.Namespace)
			continue
		} else if err != nil {
			return fmt.Errorf("failed to get: %w", err)
		} else if !deploymentImageChanged(&desired, existing) {
			continue
		}

		// it exists already, merge the two if the end result isn't identical to the existing one
		updated := existing.DeepCopy()
		if updated.Labels == nil {
			updated.Labels = map[string]string{}
		}

		updated.Spec = desired.Spec
		updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences

		for k, v := range desired.ObjectMeta.Labels {
			updated.ObjectMeta.Labels[k] = v
		}

		patch := client.MergeFrom(existing)

		if err := params.Client.Patch(ctx, updated, patch); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}

		params.Log.V(2).Info("applied", "deployment.name", desired.Name, "deployment.namespace", desired.Namespace)
	}

	return nil
}

func deleteDeployments(ctx context.Context, params Params, expected []appsv1.Deployment) error {
	opts := []client.ListOption{
		client.InNamespace(params.Instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, "loadbalancer"),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &appsv1.DeploymentList{}
	if err := params.Client.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	for i := range list.Items {
		existing := list.Items[i]
		del := true
		for _, keep := range expected {
			if keep.Name == existing.Name && keep.Namespace == existing.Namespace {
				del = false
			}
		}

		if del {
			if err := params.Client.Delete(ctx, &existing); err != nil {
				return fmt.Errorf("failed to delete: %w", err)
			}
			params.Log.V(2).Info("deleted", "deployment.name", existing.Name, "deployment.namespace", existing.Namespace)
		}
	}

	return nil
}

func deploymentImageChanged(desired *appsv1.Deployment, actual *appsv1.Deployment) bool {
	return !reflect.DeepEqual(desired.Spec.Template.Spec.Containers[0].Image, actual.Spec.Template.Spec.Containers[0].Image)
}
