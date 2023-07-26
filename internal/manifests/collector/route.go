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

package collector

import (
	"fmt"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/reconcileutil/naming"
)

func Routes(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) ([]*routev1.Route, error) {
	if otelcol.Spec.Ingress.Type != v1alpha1.IngressTypeRoute {
		return nil, nil
	}

	if otelcol.Spec.Mode == v1alpha1.ModeSidecar {
		logger.V(3).Info("ingress settings are not supported in sidecar mode")
		return nil, nil
	}

	var tlsCfg *routev1.TLSConfig
	switch otelcol.Spec.Ingress.Route.Termination {
	case v1alpha1.TLSRouteTerminationTypeInsecure:
		// NOTE: insecure, no tls cfg.
	case v1alpha1.TLSRouteTerminationTypeEdge:
		tlsCfg = &routev1.TLSConfig{Termination: routev1.TLSTerminationEdge}
	case v1alpha1.TLSRouteTerminationTypePassthrough:
		tlsCfg = &routev1.TLSConfig{Termination: routev1.TLSTerminationPassthrough}
	case v1alpha1.TLSRouteTerminationTypeReencrypt:
		tlsCfg = &routev1.TLSConfig{Termination: routev1.TLSTerminationReencrypt}
	default: // NOTE: if unsupported, end here.
		return nil, nil
	}

	ports := servicePortsFromCfg(logger, otelcol)

	// if we have no ports, we don't need a ingress entry
	if len(ports) == 0 {
		logger.V(1).Info(
			"the instance's configuration didn't yield any ports to open, skipping ingress",
			"instance.name", otelcol.Name,
			"instance.namespace", otelcol.Namespace,
		)
		return nil, nil
	}

	routes := make([]*routev1.Route, len(ports))
	for i, p := range ports {
		routes[i] = &routev1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name:        naming.Route(otelcol, p.Name),
				Namespace:   otelcol.Namespace,
				Annotations: otelcol.Spec.Ingress.Annotations,
				Labels: map[string]string{
					"app.kubernetes.io/name":       naming.Route(otelcol, p.Name),
					"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", otelcol.Namespace, otelcol.Name),
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
				},
			},
			Spec: routev1.RouteSpec{
				Host: p.Name + "." + otelcol.Spec.Ingress.Hostname,
				Path: "/" + p.Name,
				To: routev1.RouteTargetReference{
					Kind: "Service",
					Name: naming.Service(otelcol),
				},
				Port: &routev1.RoutePort{
					// Valid names must be non-empty and no more than 15 characters long.
					TargetPort: intstr.FromString(naming.Truncate(p.Name, 15)),
				},
				WildcardPolicy: routev1.WildcardPolicyNone,
				TLS:            tlsCfg,
			},
		}
	}
	return routes, nil
}
