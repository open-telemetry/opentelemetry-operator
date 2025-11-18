// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func HTTPRoutes(params manifests.Params) ([]*gatewayv1.HTTPRoute, error) {
	if !params.OtelCol.Spec.HttpRoute.Enabled {
		return nil, nil
	}

	if params.OtelCol.Spec.Mode == v1beta1.ModeSidecar {
		params.Log.V(3).Info("HTTPRoute settings are not supported in sidecar mode")
		return nil, nil
	}

	// Gateway name is required
	if params.OtelCol.Spec.HttpRoute.Gateway == "" {
		params.Log.V(1).Info(
			"HTTPRoute is enabled but gateway name is not specified, skipping HTTPRoute",
			"instance.name", params.OtelCol.Name,
			"instance.namespace", params.OtelCol.Namespace,
		)
		return nil, nil
	}

	ports, err := servicePortsFromCfg(params.Log, params.OtelCol)
	if len(ports) == 0 || err != nil {
		params.Log.V(1).Info(
			"the instance's configuration didn't yield any ports to open, skipping HTTPRoute",
			"instance.name", params.OtelCol.Name,
			"instance.namespace", params.OtelCol.Namespace,
		)
		return nil, err
	}

	httpRouteConfig := &params.OtelCol.Spec.HttpRoute
	var httpRoutes []*gatewayv1.HTTPRoute

	for _, port := range ports {
		// dont create HTTPRoute for gRPC ports
		if port.AppProtocol != nil && (*port.AppProtocol == "grpc" || *port.AppProtocol == "h2c") {
			params.Log.V(2).Info(
				"skipping gRPC port for HTTPRoute",
				"instance.name", params.OtelCol.Name,
				"instance.namespace", params.OtelCol.Namespace,
				"port.name", port.Name,
			)
			continue
		}

		name := naming.HTTPRoute(params.OtelCol.Name, port.Name)
		labels := manifestutils.Labels(params.OtelCol.ObjectMeta, name, params.OtelCol.Spec.Image, ComponentOpenTelemetryCollector, params.Config.LabelsFilter)

		// auto-generate ParentRefs from gateway name and namespace
		gatewayNamespace := httpRouteConfig.GatewayNamespace
		if gatewayNamespace == "" {
			// default to collector namespace if gateway namespace is not specified or it should be default?
			gatewayNamespace = params.OtelCol.Namespace
		}

		namespace := gatewayv1.Namespace(gatewayNamespace)
		group := gatewayv1.Group("gateway.networking.k8s.io")
		kind := gatewayv1.Kind("Gateway")

		parentRefs := []gatewayv1.ParentReference{
			{
				Group:     &group,
				Kind:      &kind,
				Name:      gatewayv1.ObjectName(httpRouteConfig.Gateway),
				Namespace: &namespace,
			},
		}

		var hostnames []gatewayv1.Hostname
		for _, hostname := range httpRouteConfig.Hostnames {
			hostnames = append(hostnames, gatewayv1.Hostname(hostname))
		}

		// use PathPrefix as match type so that multiple routes can be created without conflict
		pathMatchType := gatewayv1.PathMatchPathPrefix
		pathPrefix := fmt.Sprintf("/%s", port.Name)
		portNumber := gatewayv1.PortNumber(port.Port)

		replacePrefixMatch := "/"
		pathModifierType := gatewayv1.PrefixMatchHTTPPathModifier

		httpRoute := &gatewayv1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: params.OtelCol.Namespace,
				Labels:    labels,
			},
			Spec: gatewayv1.HTTPRouteSpec{
				CommonRouteSpec: gatewayv1.CommonRouteSpec{
					ParentRefs: parentRefs,
				},
				Hostnames: hostnames,
				Rules: []gatewayv1.HTTPRouteRule{
					{
						Matches: []gatewayv1.HTTPRouteMatch{
							{
								Path: &gatewayv1.HTTPPathMatch{
									Type:  &pathMatchType,
									Value: &pathPrefix,
								},
							},
						},
						Filters: []gatewayv1.HTTPRouteFilter{
							{
								Type: gatewayv1.HTTPRouteFilterURLRewrite,
								URLRewrite: &gatewayv1.HTTPURLRewriteFilter{
									Path: &gatewayv1.HTTPPathModifier{
										Type:               pathModifierType,
										ReplacePrefixMatch: &replacePrefixMatch,
									},
								},
							},
						},
						BackendRefs: []gatewayv1.HTTPBackendRef{
							{
								BackendRef: gatewayv1.BackendRef{
									BackendObjectReference: gatewayv1.BackendObjectReference{
										Name: gatewayv1.ObjectName(naming.Service(params.OtelCol.Name)),
										Port: &portNumber,
									},
								},
							},
						},
					},
				},
			},
		}

		httpRoutes = append(httpRoutes, httpRoute)
	}

	return httpRoutes, nil
}
