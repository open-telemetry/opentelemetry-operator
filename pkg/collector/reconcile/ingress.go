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
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

func desiredIngresses(_ context.Context, params Params) *networkingv1.Ingress {
	if params.Instance.Spec.Ingress.Type != v1alpha1.IngressTypeNginx {
		return nil
	}

	config, err := adapters.ConfigFromString(params.Instance.Spec.Config)
	if err != nil {
		params.Log.Error(err, "couldn't extract the configuration from the context")
		return nil
	}

	ports, err := adapters.ConfigToReceiverPorts(params.Log, config)
	if err != nil {
		params.Log.Error(err, "couldn't build the ingress for this instance")
		return nil
	}

	if len(params.Instance.Spec.Ports) > 0 {
		// we should add all the ports from the CR
		// there are two cases where problems might occur:
		// 1) when the port number is already being used by a receiver
		// 2) same, but for the port name
		//
		// in the first case, we remove the port we inferred from the list
		// in the second case, we rename our inferred port to something like "port-%d"
		portNumbers, portNames := extractPortNumbersAndNames(params.Instance.Spec.Ports)
		resultingInferredPorts := []corev1.ServicePort{}
		for _, inferred := range ports {
			if filtered := filterPort(params.Log, inferred, portNumbers, portNames); filtered != nil {
				resultingInferredPorts = append(resultingInferredPorts, *filtered)
			}
		}

		ports = append(params.Instance.Spec.Ports, resultingInferredPorts...)
	}

	// if we have no ports, we don't need a ingress entry
	if len(ports) == 0 {
		params.Log.V(1).Info(
			"the instance's configuration didn't yield any ports to open, skipping ingress",
			"instance.name", params.Instance.Name,
			"instance.namespace", params.Instance.Namespace,
		)
		return nil
	}

	pathType := networkingv1.PathTypePrefix
	paths := make([]networkingv1.HTTPIngressPath, len(ports))
	for i, p := range ports {
		paths[i] = networkingv1.HTTPIngressPath{
			Path:     "/" + p.Name,
			PathType: &pathType,
			Backend: networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: naming.Service(params.Instance),
					Port: networkingv1.ServiceBackendPort{
						// Valid names must be non-empty and no more than 15 characters long.
						Name: naming.Truncate(p.Name, 15),
					},
				},
			},
		}
	}

	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Ingress(params.Instance),
			Namespace:   params.Instance.Namespace,
			Annotations: params.Instance.Spec.Ingress.Annotations,
			Labels: map[string]string{
				"app.kubernetes.io/name":       naming.Ingress(params.Instance),
				"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
			},
		},
		Spec: networkingv1.IngressSpec{
			TLS: params.Instance.Spec.Ingress.TLS,
			Rules: []networkingv1.IngressRule{
				{
					Host: params.Instance.Spec.Ingress.Hostname,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: paths,
						},
					},
				},
			},
			IngressClassName: params.Instance.Spec.Ingress.IngressClassName,
		},
	}
}

// Ingresses reconciles the ingress(s) required for the instance in the current context.
func Ingresses(ctx context.Context, params Params) error {
	isSupportedMode := true
	if params.Instance.Spec.Mode == v1alpha1.ModeSidecar {
		params.Log.V(3).Info("ingress settings are not supported in sidecar mode")
		isSupportedMode = false
	}

	nns := types.NamespacedName{Namespace: params.Instance.Namespace, Name: params.Instance.Name}
	err := params.Client.Get(ctx, nns, &corev1.Service{}) // NOTE: check if service exists.
	serviceExists := err != nil

	var desired []networkingv1.Ingress
	if isSupportedMode && serviceExists {
		if d := desiredIngresses(ctx, params); d != nil {
			desired = append(desired, *d)
		}
	}

	// first, handle the create/update parts
	if err := expectedIngresses(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected ingresses: %w", err)
	}

	// then, delete the extra objects
	if err := deleteIngresses(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the ingresses to be deleted: %w", err)
	}

	return nil
}

func expectedIngresses(ctx context.Context, params Params, expected []networkingv1.Ingress) error {
	for _, obj := range expected {
		desired := obj

		if err := controllerutil.SetControllerReference(&params.Instance, &desired, params.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}

		existing := &networkingv1.Ingress{}
		nns := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
		clientGetErr := params.Client.Get(ctx, nns, existing)
		if clientGetErr != nil && k8serrors.IsNotFound(clientGetErr) {
			if err := params.Client.Create(ctx, &desired); err != nil {
				return fmt.Errorf("failed to create: %w", err)
			}
			params.Log.V(2).Info("created", "ingress.name", desired.Name, "ingress.namespace", desired.Namespace)
			return nil
		} else if clientGetErr != nil {
			return fmt.Errorf("failed to get: %w", clientGetErr)
		}

		// it exists already, merge the two if the end result isn't identical to the existing one
		updated := existing.DeepCopy()
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}
		if updated.Labels == nil {
			updated.Labels = map[string]string{}
		}
		updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences
		updated.Spec.Rules = desired.Spec.Rules
		updated.Spec.TLS = desired.Spec.TLS
		updated.Spec.DefaultBackend = desired.Spec.DefaultBackend
		updated.Spec.IngressClassName = desired.Spec.IngressClassName

		for k, v := range desired.ObjectMeta.Annotations {
			updated.ObjectMeta.Annotations[k] = v
		}
		for k, v := range desired.ObjectMeta.Labels {
			updated.ObjectMeta.Labels[k] = v
		}

		patch := client.MergeFrom(existing)

		if err := params.Client.Patch(ctx, updated, patch); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}

		params.Log.V(2).Info("applied", "ingress.name", desired.Name, "ingress.namespace", desired.Namespace)
	}
	return nil
}

func deleteIngresses(ctx context.Context, params Params, expected []networkingv1.Ingress) error {
	opts := []client.ListOption{
		client.InNamespace(params.Instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &networkingv1.IngressList{}
	if err := params.Client.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	for i := range list.Items {
		existing := list.Items[i]
		del := true
		for _, keep := range expected {
			if keep.Name == existing.Name && keep.Namespace == existing.Namespace {
				del = false
				break
			}
		}

		if del {
			if err := params.Client.Delete(ctx, &existing); err != nil {
				return fmt.Errorf("failed to delete: %w", err)
			}
			params.Log.V(2).Info("deleted", "ingress.name", existing.Name, "ingress.namespace", existing.Namespace)
		}
	}

	return nil
}
