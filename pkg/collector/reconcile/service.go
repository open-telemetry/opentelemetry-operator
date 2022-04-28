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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator"
)

// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Services reconciles the service(s) required for the instance in the current context.
func Services(ctx context.Context, params Params) error {
	desired := []corev1.Service{}
	if params.Instance.Spec.Mode != v1alpha1.ModeSidecar {
		type builder func(context.Context, Params) *corev1.Service
		for _, builder := range []builder{desiredService, headless, monitoringService} {
			svc := builder(ctx, params)
			// add only the non-nil to the list
			if svc != nil {
				desired = append(desired, *svc)
			}
		}
	}

	if params.Instance.Spec.TargetAllocator.Enabled {
		desired = append(desired, desiredTAService(params))
	}

	// first, handle the create/update parts
	if err := expectedServices(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected services: %w", err)
	}

	// then, delete the extra objects
	if err := deleteServices(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the services to be deleted: %w", err)
	}

	return nil
}

func desiredService(ctx context.Context, params Params) *corev1.Service {
	labels := collector.Labels(params.Instance, []string{})
	labels["app.kubernetes.io/name"] = naming.Service(params.Instance)

	// by coincidence, the selector is the same as the label, but note that the selector points to the deployment
	// whereas 'labels' refers to the service
	selector := labels

	config, err := adapters.ConfigFromString(params.Instance.Spec.Config)
	if err != nil {
		params.Log.Error(err, "couldn't extract the configuration from the context")
		return nil
	}

	ports, err := adapters.ConfigToReceiverPorts(params.Log, config)
	if err != nil {
		params.Log.Error(err, "couldn't build the service for this instance")
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

	// if we have no ports, we don't need a service
	if len(ports) == 0 {
		params.Log.V(1).Info("the instance's configuration didn't yield any ports to open, skipping service", "instance.name", params.Instance.Name, "instance.namespace", params.Instance.Namespace)
		return nil
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.Service(params.Instance),
			Namespace:   params.Instance.Namespace,
			Labels:      labels,
			Annotations: params.Instance.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Selector:  selector,
			ClusterIP: "",
			Ports:     ports,
		},
	}
}

func desiredTAService(params Params) corev1.Service {
	labels := targetallocator.Labels(params.Instance)
	labels["app.kubernetes.io/name"] = naming.TAService(params.Instance)

	selector := targetallocator.Labels(params.Instance)
	selector["app.kubernetes.io/name"] = naming.TargetAllocator(params.Instance)

	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      naming.TAService(params.Instance),
			Namespace: params.Instance.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Ports: []corev1.ServicePort{{
				Name:       "targetallocation",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
}

func headless(ctx context.Context, params Params) *corev1.Service {
	h := desiredService(ctx, params)
	if h == nil {
		return nil
	}

	h.Name = naming.HeadlessService(params.Instance)

	// copy to avoid modifying params.Instance.Annotations
	annotations := map[string]string{
		"service.beta.openshift.io/serving-cert-secret-name": fmt.Sprintf("%s-tls", h.Name),
	}
	for k, v := range h.Annotations {
		annotations[k] = v
	}
	h.Annotations = annotations

	h.Spec.ClusterIP = "None"
	return h
}

func monitoringService(ctx context.Context, params Params) *corev1.Service {
	labels := collector.Labels(params.Instance, []string{})
	labels["app.kubernetes.io/name"] = naming.MonitoringService(params.Instance)

	selector := collector.Labels(params.Instance, []string{})
	selector["app.kubernetes.io/name"] = fmt.Sprintf("%s-collector", params.Instance.Name)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        naming.MonitoringService(params.Instance),
			Namespace:   params.Instance.Namespace,
			Labels:      labels,
			Annotations: params.Instance.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Selector:  selector,
			ClusterIP: "",
			Ports: []corev1.ServicePort{{
				Name: "monitoring",
				Port: 8888,
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
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}
		if updated.Labels == nil {
			updated.Labels = map[string]string{}
		}
		updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences

		for k, v := range desired.ObjectMeta.Annotations {
			updated.ObjectMeta.Annotations[k] = v
		}
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
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
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
				break
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

func filterPort(logger logr.Logger, candidate corev1.ServicePort, portNumbers map[int32]bool, portNames map[string]bool) *corev1.ServicePort {
	if portNumbers[candidate.Port] {
		return nil
	}

	// do we have the port name there already?
	if portNames[candidate.Name] {
		// there's already a port with the same name! do we have a 'port-%d' already?
		fallbackName := fmt.Sprintf("port-%d", candidate.Port)
		if portNames[fallbackName] {
			// that wasn't expected, better skip this port
			logger.V(2).Info("a port name specified in the CR clashes with an inferred port name, and the fallback port name clashes with another port name! Skipping this port.",
				"inferred-port-name", candidate.Name,
				"fallback-port-name", fallbackName,
			)
			return nil
		}

		candidate.Name = fallbackName
		return &candidate
	}

	// this port is unique, return as is
	return &candidate
}

func extractPortNumbersAndNames(ports []corev1.ServicePort) (map[int32]bool, map[string]bool) {
	numbers := map[int32]bool{}
	names := map[string]bool{}

	for _, port := range ports {
		numbers[port.Port] = true
		names[port.Name] = true
	}

	return numbers, names
}
