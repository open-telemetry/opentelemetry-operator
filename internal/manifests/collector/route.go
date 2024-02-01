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

	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha2"
	"github.com/open-telemetry/opentelemetry-operator/internal/api/convert"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/openshift"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func Routes(params manifests.Params) ([]*routev1.Route, error) {
	otelCol, err := convert.V1Alpha1to2(params.OtelCol)
	if err != nil {
		return nil, err
	}
	if otelCol.Spec.Ingress.Type != v1alpha2.IngressTypeRoute || params.Config.OpenShiftRoutesAvailability() != openshift.RoutesAvailable {
		return nil, nil
	}

	if otelCol.Spec.Mode == v1alpha2.ModeSidecar {
		params.Log.V(3).Info("ingress settings are not supported in sidecar mode")
		return nil, nil
	}

	var tlsCfg *routev1.TLSConfig
	switch otelCol.Spec.Ingress.Route.Termination {
	case v1alpha2.TLSRouteTerminationTypeInsecure:
		// NOTE: insecure, no tls cfg.
	case v1alpha2.TLSRouteTerminationTypeEdge:
		tlsCfg = &routev1.TLSConfig{Termination: routev1.TLSTerminationEdge}
	case v1alpha2.TLSRouteTerminationTypePassthrough:
		tlsCfg = &routev1.TLSConfig{Termination: routev1.TLSTerminationPassthrough}
	case v1alpha2.TLSRouteTerminationTypeReencrypt:
		tlsCfg = &routev1.TLSConfig{Termination: routev1.TLSTerminationReencrypt}
	default: // NOTE: if unsupported, end here.
		return nil, nil
	}

	ports, err := servicePortsFromCfg(params.Log, otelCol)

	// if we have no ports, we don't need a ingress entry
	if len(ports) == 0 || err != nil {
		params.Log.V(1).Info(
			"the instance's configuration didn't yield any ports to open, skipping ingress",
			"instance.name", otelCol.Name,
			"instance.namespace", otelCol.Namespace,
		)
		return nil, err
	}

	routes := make([]*routev1.Route, len(ports))
	for i, p := range ports {
		portName := naming.PortName(p.Name, p.Port)
		host := ""
		if otelCol.Spec.Ingress.Hostname != "" {
			host = fmt.Sprintf("%s.%s", portName, params.OtelCol.Spec.Ingress.Hostname)
		}

		routes[i] = &routev1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name:        naming.Route(params.OtelCol.Name, p.Name),
				Namespace:   otelCol.Namespace,
				Annotations: otelCol.Spec.Ingress.Annotations,
				Labels: map[string]string{
					"app.kubernetes.io/name":       naming.Route(otelCol.Name, p.Name),
					"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", otelCol.Namespace, otelCol.Name),
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
	return routes, nil
}
