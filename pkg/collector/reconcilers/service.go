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

package reconcilers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/reconcile"
)

// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Services reconciles the service(s) required for the instance in the current context.
func Services(ctx context.Context, params reconcile.Params) error {
	desired := []corev1.Service{}
	if params.Instance.Spec.Mode != v1alpha1.ModeSidecar {
		type builder func(context.Context, reconcile.Params) *corev1.Service
		for _, builder := range []builder{desiredService, headless, monitoringService} {
			svc := builder(ctx, params)
			// add only the non-nil to the list
			if svc != nil {
				desired = append(desired, *svc)
			}
		}
	}

	return reconcile.Services(ctx, params, desired)
}

func desiredService(ctx context.Context, params reconcile.Params) *corev1.Service {
	labels := collector.Labels(params.Instance)
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

func headless(ctx context.Context, params reconcile.Params) *corev1.Service {
	h := desiredService(ctx, params)
	if h == nil {
		return nil
	}

	h.Name = naming.HeadlessService(params.Instance)
	h.Spec.ClusterIP = "None"
	return h
}

func monitoringService(ctx context.Context, params reconcile.Params) *corev1.Service {
	labels := collector.Labels(params.Instance)
	labels["app.kubernetes.io/name"] = naming.MonitoringService(params.Instance)

	selector := collector.Labels(params.Instance)
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
