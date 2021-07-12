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

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/pkg/loadbalancer"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Services reconciles the service(s) required for the instance in the current context.
func Services(ctx context.Context, params Params) error {
	desired := []corev1.Service{}
	_, err := checkConfig(params)
	if err != nil {
		return fmt.Errorf("failed to parse Promtheus config: %v", err)
	}

	if checkMode(params.Instance.Spec.Mode, params.Instance.Spec.LoadBalancer.Mode) {
		desired = append(desired, desiredService(params))
	}

	// first, handle the create/update parts
	if err := expectedServices(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected services: %v", err)
	}

	// then, delete the extra objects
	if err := deleteServices(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the services to be deleted: %v", err)
	}

	return nil
}

func desiredService(params Params) corev1.Service {
	labels := loadbalancer.Labels(params.Instance)
	labels["app.kubernetes.io/name"] = naming.LBService(params.Instance)

	selector := loadbalancer.Labels(params.Instance)
	selector["app.kubernetes.io/name"] = naming.LoadBalancer(params.Instance)

	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.LBService(params.Instance),
			Namespace: params.Instance.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Ports: []corev1.ServicePort{{
				Name:       "loadbalancing",
				Port:       80,
				TargetPort: intstr.FromInt(80),
			}},
		},
	}
}

func expectedServices(ctx context.Context, params Params, expected []corev1.Service) error {
	for _, obj := range expected {
		desired := obj

		if err := controllerutil.SetControllerReference(&params.Instance, &desired, params.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}

		existing := &corev1.Service{}
		nns := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
		err := params.Client.Get(ctx, nns, existing)
		if err != nil && k8serrors.IsNotFound(err) {
			if err := params.Client.Create(ctx, &desired); err != nil {
				return fmt.Errorf("failed to create: %w", err)
			}
			params.Log.V(2).Info("created", "service.name", desired.Name, "service.namespace", desired.Namespace)
			continue
		} else if err != nil {
			return fmt.Errorf("failed to get: %w", err)
		}

		// it exists already, merge the two if the end result isn't identical to the existing one
		updated := existing.DeepCopy()
		if updated.Labels == nil {
			updated.Labels = map[string]string{}
		}
		updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences

		for k, v := range desired.ObjectMeta.Labels {
			updated.ObjectMeta.Labels[k] = v
		}
		updated.Spec.Ports = desired.Spec.Ports

		patch := client.MergeFrom(existing)

		if err := params.Client.Patch(ctx, updated, patch); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}

		params.Log.V(2).Info("applied", "service.name", desired.Name, "service.namespace", desired.Namespace)
	}

	return nil
}

func deleteServices(ctx context.Context, params Params, expected []corev1.Service) error {
	opts := []client.ListOption{
		client.InNamespace(params.Instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Name, "loadbalancer"),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &corev1.ServiceList{}
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
			params.Log.V(2).Info("deleted", "service.name", existing.Name, "service.namespace", existing.Namespace)
		}
	}

	return nil
}
