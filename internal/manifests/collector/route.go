// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"fmt"

	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func Routes(params manifests.Params) ([]client.Object, error) {
	if params.OtelCol.Spec.Ingress.Type != v1beta1.IngressTypeRoute || params.Config.OpenShiftRoutesAvailability() != openshift.RoutesAvailable {
		return nil, nil
	}

	if params.OtelCol.Spec.Mode == v1beta1.ModeSidecar {
		params.Log.V(3).Info("ingress settings are not supported in sidecar mode")
		return nil, nil
	}

	var tlsCfg *routev1.TLSConfig
	switch params.OtelCol.Spec.Ingress.Route.Termination {
	case v1beta1.TLSRouteTerminationTypeInsecure:
		// NOTE: insecure, no tls cfg.
	case v1beta1.TLSRouteTerminationTypeEdge:
		tlsCfg = &routev1.TLSConfig{Termination: routev1.TLSTerminationEdge}
	case v1beta1.TLSRouteTerminationTypePassthrough:
		tlsCfg = &routev1.TLSConfig{Termination: routev1.TLSTerminationPassthrough}
	case v1beta1.TLSRouteTerminationTypeReencrypt:
		tlsCfg = &routev1.TLSConfig{Termination: routev1.TLSTerminationReencrypt}
	default: // NOTE: if unsupported, end here.
		return nil, nil
	}

	ports, err := servicePortsFromCfg(params.Log, params.OtelCol)

	// if we have no ports, we don't need a ingress entry
	if len(ports) == 0 || err != nil {
		params.Log.V(1).Info(
			"the instance's configuration didn't yield any ports to open, skipping ingress",
			"instance.name", params.OtelCol.Name,
			"instance.namespace", params.OtelCol.Namespace,
		)
		return nil, err
	}

	routes := make([]*routev1.Route, len(ports))
	for i, p := range ports {
		portName := naming.PortName(p.Name, p.Port)
		host := ""
		if params.OtelCol.Spec.Ingress.Hostname != "" {
			host = fmt.Sprintf("%s.%s", portName, params.OtelCol.Spec.Ingress.Hostname)
		}

		routes[i] = &routev1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name:        naming.Route(params.OtelCol.Name, p.Name),
				Namespace:   params.OtelCol.Namespace,
				Annotations: params.OtelCol.Spec.Ingress.Annotations,
				Labels: map[string]string{
					"app.kubernetes.io/name":       naming.Route(params.OtelCol.Name, p.Name),
					"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.OtelCol.Namespace, params.OtelCol.Name),
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
					"app.kubernetes.io/component":  "opentelemetry-collector",
				},
			},
			Spec: routev1.RouteSpec{
				Host: host,
				To: routev1.RouteTargetReference{
					Kind: "Service",
					Name: naming.Service(params.OtelCol.Name),
				},
				Port: &routev1.RoutePort{
					TargetPort: intstr.FromString(portName),
				},
				WildcardPolicy: routev1.WildcardPolicyNone,
				TLS:            tlsCfg,
			},
		}
	}

	// Convert []*routev1.Route to []client.Object
	result := make([]client.Object, len(routes))
	for i, route := range routes {
		result[i] = route
	}

	return result, nil
}
